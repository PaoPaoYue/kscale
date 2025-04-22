package config

import (
	"github.com/joho/godotenv"
	"log/slog"
	"os"
	"strconv"
)

var C Config

type Config struct {
	Port                       int
	APIEndpoint                string
	Environment                string
	LabelSelector              string
	OutputFilePath             string
	MetricsStorePath           string
	MaxRetryCount              int
	MetricsAggregationInterval int // in seconds
	MaxQueueSize               int
	MaxRetryQueueSize          int
	ShutdownPeriod             int // in seconds
	ImageStorePath             string
	APITimeout                 int // in seconds
}

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("No .env file found, using default values", "error", err)
	}

	C = Config{
		Port:                       getEnvInt("PORT", 8080),
		APIEndpoint:                getEnv("API_ENDPOINT", "ray-service:8000"),
		Environment:                getEnv("ENVIRONMENT", "ypp"),
		LabelSelector:              getEnv("LABEL_SELECTOR", "worker"),
		OutputFilePath:             getEnv("OUTPUT_FILE_PATH", "/tmp/output"),
		MetricsStorePath:           getEnv("METRICS_STORE_PATH", "/tmp/metrics"),
		ImageStorePath:             getEnv("IMAGE_STORE_PATH", "/tmp/image"),
		MaxRetryCount:              getEnvInt("MAX_RETRY_COUNT", 1),
		MetricsAggregationInterval: getEnvInt("METRICS_AGGREGATION_INTERVAL", 1),
		MaxQueueSize:               getEnvInt("MAX_QUEUE_SIZE", 60000),
		MaxRetryQueueSize:          getEnvInt("MAX_RETRY_QUEUE_SIZE", 1000),
		ShutdownPeriod:             getEnvInt("SHUTDOWN_PERIOD", 2),
		APITimeout:                 getEnvInt("API_TIMEOUT", 3600),
	}

	_ = os.MkdirAll(C.OutputFilePath, os.ModePerm)
	_ = os.MkdirAll(C.MetricsStorePath, os.ModePerm)
	_ = os.MkdirAll(C.ImageStorePath, os.ModePerm)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}
