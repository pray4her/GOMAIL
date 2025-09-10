package config

import (
	es "email-service/pkg/elasticsearch"
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	RabbitMQ      RabbitMQConfig
	Aliyun        AliyunConfig
	JWT           JWTConfig
	Logging       LoggingConfig
	Worker        WorkerConfig
	Scheduler     SchedulerConfig
	Elasticsearch es.Config
	FileUpload    FileUploadConfig
	Import        ImportConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host    string `mapstructure:"host"`
	Port    string `mapstructure:"port"`
	Address string `mapstructure:"address"`
}

// WorkerConfig holds background worker configuration
type WorkerConfig struct {
	EmailSenderCount int `mapstructure:"email_sender_count"`
}

// SchedulerConfig holds the task scheduler configuration.
type SchedulerConfig struct {
	PollingInterval string `mapstructure:"polling_interval"`
}

// FileUploadConfig holds file upload configuration
type FileUploadConfig struct {
	MaxFileSize  string   `mapstructure:"max_file_size"`
	UploadPath   string   `mapstructure:"upload_path"`
	AllowedTypes []string `mapstructure:"allowed_types"`
}

// ImportConfig holds import process configuration
type ImportConfig struct {
	BatchSize            int    `mapstructure:"batch_size"`
	ESBatchSize          int    `mapstructure:"es_batch_size"`
	MaxConcurrentTasks   int    `mapstructure:"max_concurrent_tasks"`
	EnableRecovery       bool   `mapstructure:"enable_recovery"`
	RecoveryStartupDelay string `mapstructure:"recovery_startup_delay"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

// RateLimitConfig holds rate limit configuration.
type RateLimitConfig struct {
	Rate  int `mapstructure:"rate"`
	Burst int `mapstructure:"burst"`
}

// AliyunConfig holds Aliyun configuration
type AliyunConfig struct {
	Endpoint  string          `mapstructure:"endpoint"`
	RegionID  string          `mapstructure:"region_id"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

var AppConfig *Config

// LoadConfig loads configuration from file and environment variables
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	AppConfig = &config
	return AppConfig, nil
}
