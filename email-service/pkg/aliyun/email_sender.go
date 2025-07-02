package aliyun

import (
	"fmt"

	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

// EmailSender encapsulates the logic for sending emails via Aliyun DM.
type EmailSender struct {
	client *dm.Client
}

// NewEmailSender creates a new sender service.
func NewEmailSender(client *dm.Client) *EmailSender {
	return &EmailSender{client: client}
}

// SendSingleEmail sends a single email.
func (s *EmailSender) SendSingleEmail(accountName, fromAlias, toAddress, subject, htmlBody string) error {
	singleSendMailRequest := &dm.SingleSendMailRequest{
		AccountName:    tea.String(accountName),
		AddressType:    tea.Int32(1), // 0 for random, 1 for sender address
		ReplyToAddress: tea.Bool(false),
		ToAddress:      tea.String(toAddress),
		Subject:        tea.String(subject),
		HtmlBody:       tea.String(htmlBody),
		FromAlias:      tea.String(fromAlias),
	}

	_, err := s.client.SingleSendMail(singleSendMailRequest)
	if err != nil {
		return fmt.Errorf("failed to send single email: %w", err)
	}

	return nil
}
