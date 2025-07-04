package model

// EmailInfo contains the per-email information for a job.
type EmailInfo struct {
	RecordID       int64  `json:"record_id"`
	RecipientEmail string `json:"recipient_email"`
}

// EmailJobPayload defines the structure of a job pushed to the email sending queue.
// It is now designed to hold a batch of emails for a single sender,
// significantly reducing the number of jobs enqueued.
type EmailJobPayload struct {
	Emails        []EmailInfo   `json:"emails"`
	Subject       string        `json:"subject"`
	Body          string        `json:"body"`
	AccountSender AccountSender `json:"account_sender"` // Pre-fetched sender details are shared across the batch
	AliyunTagName string        `json:"aliyun_tag_name"`
}
