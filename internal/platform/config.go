package platform

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL  string
	HttpAddress  string
	ProviderAURL string
	ProviderBURL string
}

func NewConfig() *Config {
	return &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"), // "postgres://postgres:postgres@localhost:5432/shop?sslmode=disable",
		HttpAddress:  os.Getenv("HTTP_ADDRESS"), // "0.0.0.0:8080",
		ProviderAURL: os.Getenv("PROVIDER_A_URL"),
		ProviderBURL: os.Getenv("PROVIDER_B_URL"),
	}
}

type ProviderConfig struct {
	DatabaseURL string
	HttpAddress string

	DoubleIssueRate float64
	FailureRate     float64
	TimeoutRate     float64

	Timeout time.Duration
}

func NewProviderConfig() (*ProviderConfig, error) {
	doubleIssueRateStr := os.Getenv("DOUBLE_ISSUE_RATE")
	doubleIssueRate, err := strconv.ParseFloat(doubleIssueRateStr, 64)
	if err != nil {
		return nil, errors.New("invalid double issue rate")
	}

	if doubleIssueRate < 0 || doubleIssueRate > 1 {
		return nil, errors.New("double issue rate must be between 0 and 1")
	}

	failureRateStr := os.Getenv("FAILURE_RATE")
	failureRate, err := strconv.ParseFloat(failureRateStr, 64)
	if err != nil {
		return nil, errors.New("invalid failure rate")
	}

	if failureRate < 0 || failureRate > 1 {
		return nil, errors.New("failure rate must be between 0 and 1")
	}

	timeoutRateStr := os.Getenv("TIMEOUT_RATE")
	timeoutRate, err := strconv.ParseFloat(timeoutRateStr, 64)
	if err != nil {
		return nil, errors.New("invalid timeout rate")
	}

	if timeoutRate < 0 || timeoutRate > 1 {
		return nil, errors.New("timeout rate must be between 0 and 1")
	}

	if failureRate+timeoutRate > 1 {
		return nil, errors.New("failure rate + timeout rate + double issue rate must not exceed 1")
	}

	timeout, err := time.ParseDuration(os.Getenv("TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %v", err)
	}

	return &ProviderConfig{
		DatabaseURL:     os.Getenv("DATABASE_URL"), // "postgres://postgres:postgres@localhost:5432/shop?sslmode=disable",
		HttpAddress:     os.Getenv("HTTP_ADDRESS"), // "0.0.0.0:8080",
		DoubleIssueRate: doubleIssueRate,
		FailureRate:     failureRate,
		TimeoutRate:     timeoutRate,
		Timeout:         timeout,
	}, nil
}

type SeedConfig struct {
	DatabaseURL string
}

func NewSeedConfig() *SeedConfig {
	return &SeedConfig{
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}
