package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	redisCache "github.com/aniladanir/auto-messender-service/internal/cache/redis"
	"github.com/aniladanir/auto-messender-service/internal/domain"
	httpHandler "github.com/aniladanir/auto-messender-service/internal/handler/http"
	"github.com/aniladanir/auto-messender-service/internal/persistant/postgresql"
	messageRepo "github.com/aniladanir/auto-messender-service/internal/repository/message"
	"github.com/aniladanir/auto-messender-service/internal/service"
	"gorm.io/gorm"
)

var (
	configFile = flag.String("config", "config.json", "config file path")
)

func main() {
	// create root context
	appCtx, appCtxCancel := context.WithCancel(context.Background())
	defer appCtxCancel()

	// listen for terminate signal
	notifyCtx, stop := signal.NotifyContext(appCtx, syscall.SIGTERM)
	defer stop()

	// parse flags
	flag.Parse()

	// parse config
	config, err := ReadConfigJson(*configFile)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	// initialize external dependencies
	db, rClient, err := initExternalDependencies(notifyCtx, config)
	if err != nil {
		log.Fatalf("failed to initialize external dependencies: %v", err)
	}

	// setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// init message repository
	msgRepo := messageRepo.NewMessageRepository(db, rClient)

	// init message sender service
	msgSender, err := service.NewMessageSenderService(
		msgRepo,
		logger.With(slog.String("component", "messageSender")),
		config.WebHookUrl,
		&config.MsgMaxRetry,
		config.MsgBatchSize,
		config.MsgSendInterval,
	)
	if err != nil {
		log.Fatalf("failed to initiate message sender service: %v", err)
	}

	// populate database with dummy data
	if err := populateDatabase(db); err != nil {
		log.Fatalf("failed to populate db: %v", err)
	}

	// init http handler
	httpHandler := httpHandler.NewHttpHandler(
		fmt.Sprintf(":%d", config.HttpPort),
		msgSender,
	)

	// Start Scheduler automatically on deployment as requested
	msgSender.Start()

	wg := sync.WaitGroup{}
	// run http handler
	wg.Go(func() {
		if err := httpHandler.Run(); err != nil {
			logger.Error("http server encountered with an error and closed", "error", err.Error())
		}
		// cancel app context if http handler fails
		appCtxCancel()
	})

	// graceful shutdown
	wg.Go(func() {
		<-notifyCtx.Done()
		logger.Info("application shutting down...")

		shutDownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		msgSender.Stop()
		httpHandler.Shutdown(shutDownCtx)
		postgresql.Close(db)
	})

	wg.Wait()
	os.Exit(0)
}

func initExternalDependencies(ctx context.Context, config *Config) (db *gorm.DB, rCache *redisCache.RedisCache, err error) {
	// initialize database
	db, err = postgresql.Initialize(config.DbConnString, []any{&domain.Message{}})
	if err != nil {
		return
	}

	// initialize cache
	rCache, err = redisCache.NewRedisCache(ctx, config.RedisAddr)

	return
}

func populateDatabase(db *gorm.DB) error {
	var msgCount int64
	if err := db.Model(&domain.Message{}).Count(&msgCount).Error; err != nil {
		return err
	}
	if msgCount == 0 {
		messages := []domain.Message{
			{Content: "Hello World 1", PhoneNumber: "+905549998877"},
			{Content: "Hello World 2", PhoneNumber: "+905549998876"},
			{Content: "Hello World 3", PhoneNumber: "+905549998875"},
			{Content: "Hello World 4", PhoneNumber: "+905549998874"},
			{Content: "Hello World 5", PhoneNumber: "+905549998873"},
			{Content: "Hello World 6", PhoneNumber: "+905549998872"},
			{Content: "Hello World 7", PhoneNumber: "+905549998871"},
			{Content: "Hello World 8", PhoneNumber: "+905549998870"},
		}
		if err := db.Create(&messages).Error; err != nil {
			return err
		}
	}

	return nil
}
