package service

import (
	"email-service/internal/model"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// FileProcessor defines the interface for processing different file formats.
type FileProcessor interface {
	ParseCSV(filePath string) ([]map[string]interface{}, error)
	ParseExcel(filePath string) ([]map[string]interface{}, error)
	ParseJSON(filePath string) ([]map[string]interface{}, error)
	ParseCSVWithOffset(filePath string, offset int) ([]map[string]interface{}, error)
	ParseExcelWithOffset(filePath string, offset int) ([]map[string]interface{}, error)
	ParseJSONWithOffset(filePath string, offset int) ([]map[string]interface{}, error)
	ValidateRecipientData(data map[string]interface{}) (*model.Recipient, error)
}

type fileProcessorService struct{}

// NewFileProcessorService creates a new FileProcessor.
func NewFileProcessorService() FileProcessor {
	return &fileProcessorService{}
}

func (f *fileProcessorService) ParseCSV(filePath string) ([]map[string]interface{}, error) {
	return f.ParseCSVWithOffset(filePath, 0)
}

func (f *fileProcessorService) ParseCSVWithOffset(filePath string, offset int) ([]map[string]interface{}, error) {
	file, err := openFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file must contain at least a header row and one data row")
	}

	headers := records[0]
	var results []map[string]interface{}

	// Start from offset+1 to account for header row
	startIndex := offset + 1
	if startIndex >= len(records) {
		return results, nil // Return empty slice if offset is beyond file
	}

	for i, record := range records[startIndex:] {
		if len(record) != len(headers) {
			return nil, fmt.Errorf("row %d has %d columns but expected %d", startIndex+i+1, len(record), len(headers))
		}

		rowData := make(map[string]interface{})
		for j, value := range record {
			header := strings.TrimSpace(headers[j])
			rowData[header] = strings.TrimSpace(value)
		}

		// Parse metadata if present
		if metadataStr, ok := rowData["metadata"].(string); ok && metadataStr != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
				rowData["metadata"] = metadata
			}
		}

		results = append(results, rowData)
	}

	return results, nil
}

func (f *fileProcessorService) ParseExcel(filePath string) ([]map[string]interface{}, error) {
	return f.ParseExcelWithOffset(filePath, 0)
}

func (f *fileProcessorService) ParseExcelWithOffset(filePath string, offset int) ([]map[string]interface{}, error) {
	file, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	// Get the first sheet
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file contains no sheets")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("excel file must contain at least a header row and one data row")
	}

	headers := rows[0]
	var results []map[string]interface{}

	// Start from offset+1 to account for header row
	startIndex := offset + 1
	if startIndex >= len(rows) {
		return results, nil // Return empty slice if offset is beyond file
	}

	for _, row := range rows[startIndex:] {
		rowData := make(map[string]interface{})

		// Handle variable column count (Excel rows might have different lengths)
		for j, header := range headers {
			header = strings.TrimSpace(header)
			var value string
			if j < len(row) {
				value = strings.TrimSpace(row[j])
			}
			rowData[header] = value
		}

		// Parse metadata if present
		if metadataStr, ok := rowData["metadata"].(string); ok && metadataStr != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
				rowData["metadata"] = metadata
			}
		}

		// Skip empty rows
		if f.isRowEmpty(rowData) {
			continue
		}

		results = append(results, rowData)
	}

	return results, nil
}

func (f *fileProcessorService) ParseJSON(filePath string) ([]map[string]interface{}, error) {
	return f.ParseJSONWithOffset(filePath, 0)
}

func (f *fileProcessorService) ParseJSONWithOffset(filePath string, offset int) ([]map[string]interface{}, error) {
	file, err := openFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var jsonData struct {
		Recipients []map[string]interface{} `json:"recipients"`
	}

	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON file: %w", err)
	}

	if len(jsonData.Recipients) == 0 {
		return nil, fmt.Errorf("json file must contain at least one recipient in 'recipients' array")
	}

	// Apply offset to the recipients array
	if offset >= len(jsonData.Recipients) {
		return []map[string]interface{}{}, nil // Return empty slice if offset is beyond array
	}

	return jsonData.Recipients[offset:], nil
}

func (f *fileProcessorService) ValidateRecipientData(data map[string]interface{}) (*model.Recipient, error) {
	// Validate required email field
	email, ok := data["email"].(string)
	if !ok || email == "" {
		return nil, fmt.Errorf("email field is required and must be a non-empty string")
	}

	// Validate email format (basic validation)
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, fmt.Errorf("invalid email format: %s", email)
	}

	recipient := &model.Recipient{
		Email:  email,
		Status: "subscribed", // default status
	}

	// Handle optional first name
	if firstName, ok := data["first_name"].(string); ok && firstName != "" {
		recipient.FirstName = &firstName
	}

	// Handle optional last name
	if lastName, ok := data["last_name"].(string); ok && lastName != "" {
		recipient.LastName = &lastName
	}

	// Handle optional status
	if status, ok := data["status"].(string); ok && status != "" {
		if status != "subscribed" && status != "unsubscribed" {
			return nil, fmt.Errorf("status must be either 'subscribed' or 'unsubscribed', got: %s", status)
		}
		recipient.Status = status
	}

	// Handle optional metadata
	if metadata, ok := data["metadata"]; ok && metadata != nil {
		var metadataBytes []byte
		var err error

		switch v := metadata.(type) {
		case map[string]interface{}:
			metadataBytes, err = json.Marshal(v)
		case string:
			// If it's already a JSON string, validate it
			var temp map[string]interface{}
			if err = json.Unmarshal([]byte(v), &temp); err == nil {
				metadataBytes = []byte(v)
			}
		default:
			return nil, fmt.Errorf("metadata must be a JSON object or valid JSON string")
		}

		if err != nil {
			return nil, fmt.Errorf("invalid metadata format: %w", err)
		}

		recipient.Metadata = json.RawMessage(metadataBytes)
	}

	return recipient, nil
}

// Helper functions
func (f *fileProcessorService) isRowEmpty(row map[string]interface{}) bool {
	for _, value := range row {
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return false
		}
	}
	return true
}

// openFile is a helper function to open files for reading
var openFile = func(filePath string) (io.ReadCloser, error) {
	return os.Open(filePath)
}
