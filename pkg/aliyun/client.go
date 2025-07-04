package aliyun

import (
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

// NewClient creates a new Aliyun DM (Direct Mail) client.
// It requires the API endpoint, access key ID, and access key secret.
func NewClient(endpoint, accessKeyID, accessKeySecret string) (*dm.Client, error) {
	if endpoint == "" || accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("endpoint, accessKeyID, and accessKeySecret must be provided")
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	client, err := dm.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun dm client: %w", err)
	}

	return client, nil
}
