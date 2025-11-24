package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/aniladanir/auto-messender-service/internal/domain"
	messageRepo "github.com/aniladanir/auto-messender-service/internal/repository/message"
	"github.com/aniladanir/retry"
	"github.com/google/uuid"
)

type MessageSender interface {
	Start()
	Stop()
	GetSentMessages() ([]domain.Message, error)
}

type service struct {
	messageRepo  messageRepo.Repository
	webhookURL   string
	stopChan     chan struct{}
	isRunning    bool
	mtx          sync.Mutex
	retrier      *retry.Retrier
	httpClient   *http.Client
	logger       *slog.Logger
	msgBatchSize int
	sendInterval time.Duration
}

func NewMessageSenderService(messageRepo messageRepo.Repository, logger *slog.Logger, webhookURL string, maxRetryOnFail *int, msgBatchSize int, sendInterval time.Duration) (MessageSender, error) {
	// initialize retrier
	retrierOpts := make([]retry.Option, 0)
	if maxRetryOnFail != nil {
		retrierOpts = append(retrierOpts, retry.WithMaxAttemps(*maxRetryOnFail))
	}
	retrier, err := retry.New(retrierOpts...)
	if err != nil {
		return nil, fmt.Errorf("encountered error when initializing retrier: %w", err)
	}

	return &service{
		messageRepo: messageRepo,
		webhookURL:  webhookURL,
		stopChan:    make(chan struct{}),
		mtx:         sync.Mutex{},
		retrier:     retrier,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
		msgBatchSize: msgBatchSize,
		sendInterval: sendInterval,
	}, nil
}

// Start initializes sender service scheduler
func (s *service) Start() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.isRunning {
		return
	}

	// run scheduler
	ticker := time.NewTicker(s.sendInterval)
	go func(t *time.Ticker) {
		processCtx, processCtxCancel := context.WithCancel(context.Background())
		defer processCtxCancel()

		// initial run
		s.processBatch(processCtx, s.msgBatchSize)

		for {
			select {
			case <-t.C:
				s.processBatch(processCtx, s.msgBatchSize)
			case <-s.stopChan:
				t.Stop()
				processCtxCancel()
				return
			}
		}
	}(ticker)
}

// Stop pauses the sender service scheduler
func (s *service) Stop() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if !s.isRunning {
		return
	}

	s.stopChan <- struct{}{}
	s.isRunning = false
}

// GetSentMessages returns messages that are successfuly consumed by the external api
func (s *service) GetSentMessages() ([]domain.Message, error) {
	return s.messageRepo.GetSentMessages()
}

func (s *service) processBatch(ctx context.Context, batch int) {
	msgs, err := s.messageRepo.FetchAndLockMessages(batch)
	if err != nil {
		log.Printf("Error fetching messages: %v", err)
		return
	}

	if len(msgs) == 0 {
		return
	}

	wg := new(sync.WaitGroup)
	for _, msg := range msgs {
		wg.Go(func() {
			s.sendMessage(ctx, &msg)
		})
	}
	wg.Wait()
}

func (s *service) sendMessage(ctx context.Context, msg *domain.Message) {
	// create a logger with message id
	msgLogger := s.logger.With(slog.Int("dbMessageId", msg.ID))

	retryFunc := func(attempt int) (terminate bool) {
		retryLogger := msgLogger.With(slog.Int("attempt", attempt))

		resp, err := s.doMsgRequest(ctx, msg)
		if err != nil {
			retryLogger.Error("failed to send request", "error", err.Error())
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusAccepted {
			// request was successful
			if err := s.messageRepo.UpdateStatus(msg, domain.StatusSuccess); err != nil {
				retryLogger.Error("failed to update message status to success", "error", err.Error())
			}
			retryLogger.Info("message is successfuly sent", "requestId", resp.Header.Get("X-Request-ID"))

			// save response
			if err = s.saveResponse(ctx, resp.Body); err != nil {
				retryLogger.Error("failed to save message response", "error", err.Error())
			}
		} else if resp.StatusCode >= http.StatusInternalServerError {
			// 5XX status code indicates server error, try retry
			retryLogger.Error("response indicates error",
				"requestId", resp.Header.Get("X-Request-ID"),
				"statusCode", resp.StatusCode)
			return false
		} else if resp.StatusCode >= http.StatusBadRequest {
			// 4XX indicates client error, no need to retry
			retryLogger.Error("response indicates error",
				"requestId", resp.Header.Get("X-Request-ID"),
				"statusCode", resp.StatusCode)
			if err = s.messageRepo.UpdateStatus(msg, domain.StatusFailed); err != nil {
				retryLogger.Error("failed to update message status to failed", "error", err.Error())
			}
		}

		return true
	}

	retrySuccess := <-s.retrier.Retry(ctx, retryFunc, true)

	if !retrySuccess {
		// retrying failed
		if err := s.messageRepo.UpdateStatus(msg, domain.StatusFailed); err != nil {
			msgLogger.Error("failed to update message status to failed", "error", err.Error())
		}

	}
}

func (s *service) doMsgRequest(ctx context.Context, msg *domain.Message) (*http.Response, error) {
	payload := map[string]string{
		"to":      msg.PhoneNumber,
		"content": msg.Content,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Request-ID", uuid.NewString())

	return s.httpClient.Do(req)
}

func (s *service) saveResponse(ctx context.Context, body io.ReadCloser) error {
	var result domain.WebhookResponse
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return err
	} else if result.MessageID != "" {
		if err = s.messageRepo.CacheMessage(ctx, result.MessageID, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}
