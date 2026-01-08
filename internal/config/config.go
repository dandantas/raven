package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// MongoDB Configuration
	MongoURI      string
	MongoDatabase string
	MongoTimeout  time.Duration

	// HTTP Server Configuration
	HTTPPort         string
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration

	// Worker Pool Configuration
	WorkerPoolSize    int
	MaxConcurrentJobs int

	// Logging Configuration
	LogLevel  string
	LogFormat string

	// Timeout Configuration
	DefaultAPITimeout     time.Duration
	DefaultWebhookTimeout time.Duration

	// CORS Configuration
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
	CORSMaxAge           int

	// Scheduler Configuration
	SchedulerEnabled      bool
	SchedulerTickInterval time.Duration
	SchedulerLockTTL      time.Duration
	SchedulerConcurrency  int
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		// MongoDB
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017/raven_alert?authSource=admin"),
		MongoDatabase: getEnv("MONGO_DATABASE", "raven_alert"),
		MongoTimeout:  getDurationEnv("MONGO_TIMEOUT_SEC", 10) * time.Second,

		// HTTP Server
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		HTTPReadTimeout:  getDurationEnv("HTTP_READ_TIMEOUT_SEC", 30) * time.Second,
		HTTPWriteTimeout: getDurationEnv("HTTP_WRITE_TIMEOUT_SEC", 30) * time.Second,

		// Worker Pool
		WorkerPoolSize:    getIntEnv("WORKER_POOL_SIZE", 10),
		MaxConcurrentJobs: getIntEnv("MAX_CONCURRENT_JOBS", 1000),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Timeouts
		DefaultAPITimeout:     getDurationEnv("DEFAULT_API_TIMEOUT_SEC", 30) * time.Second,
		DefaultWebhookTimeout: getDurationEnv("DEFAULT_WEBHOOK_TIMEOUT_SEC", 10) * time.Second,

		// CORS
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods:   getEnv("CORS_ALLOWED_METHODS", "GET, POST, PUT, DELETE, OPTIONS, PATCH"),
		CORSAllowedHeaders:   getEnv("CORS_ALLOWED_HEADERS", "*"),
		CORSAllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
		CORSMaxAge:           getIntEnv("CORS_MAX_AGE", 3600),

		// Scheduler
		SchedulerEnabled:      getBoolEnv("SCHEDULER_ENABLED", true),
		SchedulerTickInterval: getDurationEnv("SCHEDULER_TICK_INTERVAL_SEC", 60) * time.Second,
		SchedulerLockTTL:      getDurationEnv("SCHEDULER_LOCK_TTL_SEC", 300) * time.Second,
		SchedulerConcurrency:  getIntEnv("SCHEDULER_CONCURRENCY", 10),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: Invalid integer value for %s, using default %d", key, defaultValue)
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return time.Duration(intVal)
		}
		log.Printf("Warning: Invalid duration value for %s, using default %d", key, defaultValue)
	}
	return time.Duration(defaultValue)
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		log.Printf("Warning: Invalid boolean value for %s, using default %t", key, defaultValue)
	}
	return defaultValue
}
