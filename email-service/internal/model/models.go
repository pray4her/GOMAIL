package model

import (
	"encoding/json"
	"time"
)

// Constants for email send record statuses
const (
	RecordStatusPending   = "pending"
	RecordStatusSending   = "sending"
	RecordStatusSent      = "sent"
	RecordStatusFailed    = "failed"
	RecordStatusDelivered = "delivered"
	RecordStatusOpened    = "opened"
	RecordStatusClicked   = "clicked"
	RecordStatusBounce    = "bounce"
)

// Account represents the accounts table
type Account struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"unique;not null"`
	AccessKeyID     string    `json:"access_key_id" gorm:"not null"`
	AccessKeySecret string    `json:"-" gorm:"not null"` // Do not expose secret
	Domain          string    `json:"domain" gorm:"not null"`
	DailySendLimit  int       `json:"daily_send_limit" gorm:"not null;default:5000"`
	Status          string    `json:"status" gorm:"not null;default:'active'"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Sender represents the senders table
type Sender struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Role        string    `json:"role" gorm:"not null"`
	ContactInfo string    `json:"contact_info"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountSender represents the account_senders table
type AccountSender struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	AccountID      int64     `json:"account_id" gorm:"uniqueIndex:idx_account_sender;not null"`
	SenderID       int64     `json:"sender_id" gorm:"uniqueIndex:idx_account_sender;not null"`
	EmailAddress   string    `json:"email_address" gorm:"unique;not null"`
	Weight         int       `json:"weight" gorm:"not null;default:100"`
	DailySendLimit int       `json:"daily_send_limit" gorm:"not null"`
	Status         string    `json:"status" gorm:"not null;default:'active'"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Account Account `json:"account" gorm:"foreignKey:AccountID"`
	Sender  Sender  `json:"sender" gorm:"foreignKey:SenderID"`
}

// EmailTemplate represents the email_templates table
type EmailTemplate struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"unique;not null"`
	Subject   string    `json:"subject" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailTask represents the email_tasks table
type EmailTask struct {
	ID              int64      `json:"id" gorm:"primaryKey"`
	AccountSenderID int64      `json:"account_sender_id" gorm:"not null"`
	TaskName        string     `json:"task_name" gorm:"not null"`
	TemplateID      *int64     `json:"template_id"`
	Subject         *string    `json:"subject"`
	Body            *string    `json:"body"`
	Status          string     `json:"status" gorm:"not null;default:'pending'"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	Priority        int        `json:"priority" gorm:"not null;default:0"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	AccountSender AccountSender `json:"account_sender" gorm:"foreignKey:AccountSenderID"`
	Template      EmailTemplate `json:"template" gorm:"foreignKey:TemplateID"`
	Recipients    []*Recipient  `json:"recipients,omitempty" gorm:"many2many:email_task_recipients;"`
}

// EmailSendRecord represents the email_send_records table
type EmailSendRecord struct {
	ID                 int64      `json:"id" gorm:"primaryKey"`
	TaskID             *int64     `json:"task_id"`
	AccountSenderID    int64      `json:"account_sender_id" gorm:"not null"`
	RecipientEmail     string     `json:"recipient_email" gorm:"not null"`
	Subject            string     `json:"subject" gorm:"not null"`
	Body               string     `json:"body" gorm:"type:text;not null"`
	Status             string     `json:"status" gorm:"not null"`
	AliyunTaskID       *string    `json:"aliyun_task_id"`
	ErrorMessage       *string    `json:"error_message"`
	SentAt             *time.Time `json:"sent_at"`
	LastStatusUpdateAt *time.Time `json:"last_status_update_at"`

	Task          EmailTask     `json:"task" gorm:"foreignKey:TaskID"`
	AccountSender AccountSender `json:"account_sender" gorm:"foreignKey:AccountSenderID"`
}

// Recipient represents the recipients table
type Recipient struct {
	ID        int64           `json:"id" gorm:"primaryKey"`
	Email     string          `json:"email" gorm:"unique;not null"`
	FirstName *string         `json:"first_name"`
	LastName  *string         `json:"last_name"`
	Status    string          `json:"status" gorm:"not null;default:'subscribed'"`
	Metadata  json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SendStatistics represents the send_statistics table
type SendStatistics struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	StatDate        time.Time `json:"stat_date" gorm:"type:date;uniqueIndex:idx_stat_date_account_sender;not null"`
	AccountID       int64     `json:"account_id" gorm:"uniqueIndex:idx_stat_date_account_sender;not null"`
	AccountSenderID int64     `json:"account_sender_id" gorm:"uniqueIndex:idx_stat_date_account_sender;not null"`
	SentCount       int       `json:"sent_count" gorm:"default:0"`
	DeliveredCount  int       `json:"delivered_count" gorm:"default:0"`
	FailedCount     int       `json:"failed_count" gorm:"default:0"`
	OpenCount       int       `json:"open_count" gorm:"default:0"`
	ClickCount      int       `json:"click_count" gorm:"default:0"`
	BounceCount     int       `json:"bounce_count" gorm:"default:0"`

	Account       Account       `json:"account" gorm:"foreignKey:AccountID"`
	AccountSender AccountSender `json:"account_sender" gorm:"foreignKey:AccountSenderID"`
}
