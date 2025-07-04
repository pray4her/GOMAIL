package model

import (
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// Account represents a third-party email service provider account (e.g., Aliyun).
// @Description Holds credentials and configuration for an external email service.
type Account struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"unique;not null"`
	AccessKeyID     string    `json:"access_key_id" gorm:"not null"`
	AccessKeySecret string    `json:"access_key_secret" gorm:"not null"`
	Domain          string    `json:"domain" gorm:"not null"`
	DailySendLimit  int       `json:"daily_send_limit" gorm:"not null;default:5000"`
	Status          string    `json:"status" gorm:"not null;default:'active'"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Sender represents an individual or entity that sends emails.
// @Description Contains information about the sender of an email.
type Sender struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Role        string    `json:"role" gorm:"not null"`
	ContactInfo string    `json:"contact_info"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountSender links a Sender with a specific Account, defining a usable "from" address.
// @Description Represents the many-to-many relationship between accounts and senders, with specific sending configurations.
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

// EmailTemplate represents a reusable email template.
// @Description Stores predefined subject and body for emails.
type EmailTemplate struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"unique;not null"`
	Subject   string    `json:"subject" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailTask represents a batch email sending job.
// @Description Defines a task to send a specific email (or template) to a list of recipients.
type EmailTask struct {
	ID              int64      `json:"id" gorm:"primaryKey"`
	TaskName        string     `json:"task_name" gorm:"not null"`
	TemplateID      *int64     `json:"template_id"`
	Subject         *string    `json:"subject"`
	Body            *string    `json:"body"`
	Status          string     `json:"status" gorm:"not null;default:'pending'"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	Priority        int        `json:"priority" gorm:"not null;default:0"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedByUserID int64      `json:"created_by_user_id" gorm:"not null"`

	// Fields for tracking via Aliyun API
	AliyunTagName    *string `json:"aliyun_tag_name,omitempty" gorm:"size:60"`
	AliyunTagID      *string `json:"aliyun_tag_id,omitempty" gorm:"size:255"`
	OpenCount        int     `json:"total_open_count" gorm:"not null;default:0"`
	ClickCount       int     `json:"total_click_count" gorm:"not null;default:0"`
	UniqueOpenCount  int     `json:"unique_open_count" gorm:"default:0"`
	UniqueClickCount int     `json:"unique_click_count" gorm:"default:0"`
	OpenRate         float64 `gorm:"default:0.0"`
	ClickRate        float64 `gorm:"default:0.0"`
	UniqueOpenRate   float64 `gorm:"default:0.0"`
	UniqueClickRate  float64 `gorm:"default:0.0"`

	Template         EmailTemplate   `json:"template" gorm:"foreignKey:TemplateID"`
	CreatedByUser    User            `json:"created_by_user" gorm:"foreignKey:CreatedByUserID"`
	RecipientGroupID *int64          `json:"recipient_group_id"`
	RecipientGroup   *RecipientGroup `json:"recipient_group,omitempty" gorm:"foreignKey:RecipientGroupID"`
}

// EmailSendRecord tracks the status of a single email sent to a recipient as part of a task.
// @Description Logs the details and delivery status of each individual email.
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

// Recipient represents an email recipient.
// @Description Stores information about a person who can receive emails, including custom data.
type Recipient struct {
	ID        int64           `json:"id" gorm:"primaryKey"`
	Email     string          `json:"email" gorm:"unique;not null"`
	FirstName *string         `json:"first_name"`
	LastName  *string         `json:"last_name"`
	Status    string          `json:"status" gorm:"not null;default:'subscribed'"`
	Metadata  json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb" swaggertype:"object,string"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// User represents an application user with login credentials and permissions.
// @Description Defines a user of this email service platform, with roles and access rights.
type User struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"unique;not null"`
	Email        string    `json:"email" gorm:"unique;not null"`
	PasswordHash string    `json:"-" gorm:"not null"` // Never expose hash
	IsAdmin      bool      `json:"is_admin" gorm:"not null;default:false"`
	IsActive     bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SetPassword hashes the password using bcrypt and sets it on the user model
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password matches the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// SendStatistics represents the send_statistics table, storing daily aggregated data.
type SendStatistics struct {
	ID               int64     `json:"id" gorm:"primaryKey"`
	StatDate         time.Time `json:"stat_date" gorm:"type:date;uniqueIndex:idx_stat_date_account_sender;not null"`
	AccountID        int64     `json:"account_id" gorm:"uniqueIndex:idx_stat_date_account_sender;not null"`
	AccountSenderID  int64     `json:"account_sender_id" gorm:"uniqueIndex:idx_stat_date_account_sender;not null"`
	SentCount        int       `json:"sent_count" gorm:"default:0"`
	OpenCount        int       `json:"open_count" gorm:"default:0"`
	UniqueOpenCount  int       `json:"unique_open_count" gorm:"default:0"`
	ClickCount       int       `json:"click_count" gorm:"default:0"`
	UniqueClickCount int       `json:"unique_click_count" gorm:"default:0"`

	Account       Account       `json:"account" gorm:"foreignKey:AccountID"`
	AccountSender AccountSender `json:"account_sender" gorm:"foreignKey:AccountSenderID"`
}

// RecipientGroup represents a segment of recipients, either static or dynamic.
// @Description Defines a group of recipients, which can be a fixed list (static) or rule-based (dynamic).
type RecipientGroup struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"unique;not null"`
	Description     *string   `json:"description"`
	GroupType       string    `json:"group_type" gorm:"not null"` // 'static' or 'dynamic'
	CreatedByUserID int64     `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Rules         []RecipientGroupRule `json:"rules,omitempty" gorm:"foreignKey:GroupID"`
	Members       []Recipient          `json:"members,omitempty" gorm:"many2many:recipient_group_members;"`
	CreatedByUser User                 `json:"created_by_user" gorm:"foreignKey:CreatedByUserID"`
}

// RecipientGroupRule defines a single rule for a dynamic recipient group.
// @Description A rule used to dynamically include recipients in a group based on their attributes.
type RecipientGroupRule struct {
	ID       int64  `json:"id" gorm:"primaryKey"`
	GroupID  int64  `json:"group_id" gorm:"not null"`
	Field    string `json:"field" gorm:"not null"`    // e.g., "status", "metadata.country"
	Operator string `json:"operator" gorm:"not null"` // e.g., "equals", "contains"
	Value    string `json:"value" gorm:"not null"`
}

// RecipientGroupMember is the explicit join table for static groups.
// GORM can use this implicitly, but defining it helps with clarity.
type RecipientGroupMember struct {
	GroupID     int64 `gorm:"primaryKey"`
	RecipientID int64 `gorm:"primaryKey"`
}
