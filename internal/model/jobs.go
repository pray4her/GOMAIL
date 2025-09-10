package model

// EmailJobPayload defines the structure of a job pushed to the email sending queue.
// It contains all necessary information for the worker to send an email without
// needing to query the database for sender or task details, thus preventing N+1 queries.
type EmailJobPayload struct {
	RecordID       int64         `json:"record_id"`
	TaskID         int64         `json:"task_id"`
	RecipientEmail string        `json:"recipient_email"`
	Subject        string        `json:"subject"`
	Body           string        `json:"body"`
	AccountSender  AccountSender `json:"account_sender"` // Pre-fetched sender details (includes Account and Sender)
	AliyunTagName  string        `json:"aliyun_tag_name"`
}
