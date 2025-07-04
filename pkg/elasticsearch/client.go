package elasticsearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// Config holds the configuration for the Elasticsearch client.
type Config struct {
	Addresses          []string `mapstructure:"addresses"`
	Username           string   `mapstructure:"username"`
	Password           string   `mapstructure:"password"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
}

// NewClient creates and returns a new Elasticsearch client based on the provided configuration.
func NewClient(cfg *Config) (*elasticsearch.Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	// For local development with self-signed certificates, allow skipping verification.
	// This should be disabled in production.
	if cfg.InsecureSkipVerify {
		log.Println("Warning: Elasticsearch TLS certificate verification is disabled.")
		esCfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}

	// Ping the server to check the connection
	res, err := es.Ping()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Fatalf("Error pinging Elasticsearch: %s", res.String())
	}

	log.Println("Successfully connected to Elasticsearch")
	return es, nil
}

// EnsureIndexExists checks if an index exists in Elasticsearch and creates it with the specified mapping if it does not.
func EnsureIndexExists(client *elasticsearch.Client, indexName, mapping string) error {
	ctx := context.Background()

	// Check if the index exists
	res, err := client.Indices.Exists([]string{indexName}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error checking if index exists: %w", err)
	}
	defer res.Body.Close()

	// StatusCode 200 means index exists
	if res.StatusCode == 200 {
		log.Printf("Index '%s' already exists\n", indexName)
		return nil
	}

	// StatusCode 404 means index does not exist, so we create it
	if res.StatusCode != 404 {
		return fmt.Errorf("unexpected status code when checking for index: %d", res.StatusCode)
	}

	log.Printf("Index '%s' not found, creating now...\n", indexName)

	// Create the index with the specific mapping
	createRes, err := client.Indices.Create(
		indexName,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("error creating index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("error response from Elasticsearch while creating index: %s", createRes.String())
	}

	log.Printf("Index '%s' created successfully.\n", indexName)
	return nil
}
