package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aniladanir/auto-messender-service/internal/cache"
	"github.com/aniladanir/auto-messender-service/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	FetchAndLockMessages(limit int) ([]domain.Message, error)
	UpdateStatus(msg *domain.Message, status domain.MessageStatus) error
	GetSentMessages() ([]domain.Message, error)
	CacheMessage(ctx context.Context, msgID string, sentTime time.Time) error
}

type repo struct {
	db    *gorm.DB
	cache cache.Cache
}

func NewMessageRepository(db *gorm.DB, cache cache.Cache) Repository {
	return &repo{db: db, cache: cache}
}

// FetchAndLockMessages retrieves pending or failed messages and sets their status to processing
func (r *repo) FetchAndLockMessages(limit int) ([]domain.Message, error) {
	var messages []domain.Message
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Select pending messages by locking selected rows
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", domain.StatusPending).Limit(limit).Find(&messages).Error; err != nil {
			return err
		}

		// Update status of selected messages as processing so they can't be fetched
		// by any other process until the transaction completes
		ids := make([]int, 0, len(messages))
		for _, m := range messages {
			ids = append(ids, m.ID)
		}

		return tx.Model(&domain.Message{}).
			Where("id IN ?", ids).
			Update("status", domain.StatusProcessing).Error
	})

	return messages, err
}

// UpdateStatus updates message status to provided status
func (r *repo) UpdateStatus(msg *domain.Message, status domain.MessageStatus) error {
	now := time.Now().UTC()
	msg.UpdatedAt = &now
	msg.Status = int(status)
	return r.db.Save(msg).Error
}

// GetSentMessages returns messages with status 'sent'
func (r *repo) GetSentMessages() ([]domain.Message, error) {
	var messages []domain.Message
	err := r.db.Where("status = ?", domain.StatusSuccess).Find(&messages).Error
	return messages, err
}

// CacheMessage writes given message attributes to cache
func (r *repo) CacheMessage(ctx context.Context, msgID string, sentTime time.Time) error {
	key := fmt.Sprintf("sent_msg:%s", msgID)

	value := map[string]any{
		"messageId": msgID,
		"sentAt":    sentTime,
	}

	jsonVal, _ := json.Marshal(value)
	// Expire after 24 hours to keep memory clean
	return r.cache.Set(ctx, key, string(jsonVal), 24*time.Hour)
}
