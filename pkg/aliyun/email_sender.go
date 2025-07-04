package aliyun

import (
	"fmt"
	"strconv"

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

// SendSingleEmail sends a single email and returns the Aliyun RequestID.
func (s *EmailSender) SendSingleEmail(accountName, fromAlias, toAddress, subject, htmlBody, tagName string, clickTrace bool) (*string, error) {
	singleSendMailRequest := &dm.SingleSendMailRequest{
		AccountName:    tea.String(accountName),
		AddressType:    tea.Int32(1), // 1 for sender address
		ReplyToAddress: tea.Bool(false),
		ToAddress:      tea.String(toAddress),
		Subject:        tea.String(subject),
		HtmlBody:       tea.String(htmlBody),
		FromAlias:      tea.String(fromAlias),
	}

	// Add the tag to the request if it's provided.
	if tagName != "" {
		singleSendMailRequest.TagName = tea.String(tagName)
	}

	// Enable click tracking if requested.
	if clickTrace {
		singleSendMailRequest.ClickTrace = tea.String("1")
	}

	response, err := s.client.SingleSendMail(singleSendMailRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send single email: %w", err)
	}

	return response.Body.RequestId, nil
}

// CreateTag creates a new tag in Aliyun and returns the TagID.
func (s *EmailSender) CreateTag(tagName, description string) (*string, error) {
	request := &dm.CreateTagRequest{
		TagName: tea.String(tagName),
	}
	if description != "" {
		request.TagDescription = tea.String(description)
	}

	response, err := s.client.CreateTag(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag in aliyun: %w", err)
	}

	return response.Body.TagId, nil
}

// DeleteTag deletes a tag from Aliyun.
func (s *EmailSender) DeleteTag(tagID string) error {
	tagIdInt, err := strconv.Atoi(tagID)
	if err != nil {
		return fmt.Errorf("invalid tagID format: %w", err)
	}

	request := &dm.DeleteTagRequest{
		TagId: tea.Int32(int32(tagIdInt)),
	}

	_, err = s.client.DeleteTag(request)
	if err != nil {
		return fmt.Errorf("failed to delete tag from aliyun: %w", err)
	}
	return nil
}

// GetTrackList queries Aliyun for email tracking information.
// Dates should be in "YYYY-MM-DD" format.
func (s *EmailSender) GetTrackList(accountName, tagName, startTime, endTime string) (*dm.GetTrackListResponseBody, error) {
	request := &dm.GetTrackListRequest{
		AccountName: tea.String(accountName),
		TagName:     tea.String(tagName),
		StartTime:   tea.String(startTime),
		EndTime:     tea.String(endTime),
		// Using PageNumber and PageSize for pagination
		PageNumber: tea.String("1"),
		PageSize:   tea.String("100"), // Max records per page
	}

	response, err := s.client.GetTrackList(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get track list from aliyun: %w", err)
	}

	if response.Body == nil || response.Body.Data == nil {
		// No data returned, which can be a valid case (no sends in that period)
		return nil, nil
	}

	return response.Body, nil
}
