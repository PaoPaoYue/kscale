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
	RayDashboardEndpoint       string
	APIEndpoint                string
	Environment                string
	LabelSelector              string
	OutputFilePath             string
	MaxRetryCount              int
	MetricsAggregationInterval int // in seconds
	MaxQueueSize               int
	MaxRetryQueueSize          int
	ShutdownPeriod             int // in seconds
	ImageStorePath             string
	APITimeout                 int // in seconds

	EnableAutoScaling bool
	InitWorkerCount   int
	MetricsWindow     int // in seconds
	ForecastWindow    int // how many data points to observe
	WorkerCostPerHour float64
	JobReward         float64
	LatencyThreshold  int // in milliseconds
}

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("No .env file found, using default values", "error", err)
	}

	C = Config{
		Port:                       getEnvInt("PORT", 8080),
		RayDashboardEndpoint:       getEnv("RAY_DASHBOARD_ENDPOINT", "ray-service:8265"),
		APIEndpoint:                getEnv("API_ENDPOINT", "ray-service:8000"),
		Environment:                getEnv("ENVIRONMENT", "ypp"),
		LabelSelector:              getEnv("LABEL_SELECTOR", "worker"),
		OutputFilePath:             getEnv("OUTPUT_FILE_PATH", "./tmp/output"),
		MaxRetryCount:              getEnvInt("MAX_RETRY_COUNT", 1),
		MetricsAggregationInterval: getEnvInt("METRICS_AGGREGATION_INTERVAL", 1),
		MaxQueueSize:               getEnvInt("MAX_QUEUE_SIZE", 60000),
		MaxRetryQueueSize:          getEnvInt("MAX_RETRY_QUEUE_SIZE", 1000),
		ShutdownPeriod:             getEnvInt("SHUTDOWN_PERIOD", 2),
		APITimeout:                 getEnvInt("API_TIMEOUT", 3600),

		EnableAutoScaling: getEnvBool("ENABLE_AUTO_SCALING", false),
		InitWorkerCount:   getEnvInt("INIT_WORKER_COUNT", 1),
		MetricsWindow:     getEnvInt("METRICS_WINDOW", 10),
		ForecastWindow:    getEnvInt("FORECAST_WINDOW", 36),
		WorkerCostPerHour: getEnvFloat("WORKER_COST_PER_HOUR", 1),
		JobReward:         getEnvFloat("JOB_REWARD", 0.002),
		LatencyThreshold:  getEnvInt("LATENCY_THRESHOLD", 8000),
	}

	_ = os.MkdirAll(C.OutputFilePath, os.ModePerm)
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return defaultValue
}
