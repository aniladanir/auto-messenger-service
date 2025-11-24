package domain

import (
	"time"
)

type MessageStatus int

const (
	StatusPending MessageStatus = iota
	StatusProcessing
	StatusSuccess
	StatusFailed
)

type Message struct {
	ID          int        `gorm:"primaryKey" json:"id"`
	Content     string     `gorm:"type:varchar(160);not null" json:"content"`
	PhoneNumber string     `gorm:"type:varchar(20);not null" json:"phone_number"`
	Status      int        `gorm:"type:int;not null" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type WebhookResponse struct {
	MessageID string `json:"messageId"`
	Message   string `json:"message"`
}
