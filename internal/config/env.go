package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env files
func LoadEnv(filenames ...string) error {
	// Default files to load
	defaultFiles := []string{".env.local", ".env"}
	
	// Use provided filenames or defaults
	files := filenames
	if len(files) == 0 {
		files = defaultFiles
	}
	
	// Try to load each file (ignore errors for non-existent files)
	for _, file := range files {
		if err := godotenv.Load(file); err == nil {
			fmt.Printf("Loaded environment from: %s\n", file)
		}
	}
	
	return nil
}

// EnvConfig holds all environment configuration
type EnvConfig struct {
	// Redis Configuration
	Redis RedisEnvConfig `json:"redis"`
	
	// Server Configuration
	Server ServerEnvConfig `json:"server"`
	
	// Security Configuration
	Security SecurityEnvConfig `json:"security"`
	
	// Queue Configuration
	Queue QueueEnvConfig `json:"queue"`
	
	// Database URLs
	Database DatabaseEnvConfig `json:"database"`
	
	// Observability Configuration
	Observability ObservabilityEnvConfig `json:"observability"`
	
	// Logging Configuration
	Logging LoggingEnvConfig `json:"logging"`
	
	// Development Configuration
	Development DevelopmentEnvConfig `json:"development"`
}

type RedisEnvConfig struct {
	Addresses        []string      `json:"addresses"`
	Password         string        `json:"password"`
	DB               int           `json:"db"`
	Username         string        `json:"username"`
	ClusterEnabled   bool          `json:"cluster_enabled"`
	TLSEnabled       bool          `json:"tls_enabled"`
	TLSCertFile      string        `json:"tls_cert_file"`
	TLSKeyFile       string        `json:"tls_key_file"`
	TLSCAFile        string        `json:"tls_ca_file"`
	TLSInsecure      bool          `json:"tls_insecure"`
	MaxIdleConns     int           `json:"max_idle_conns"`
	MaxActiveConns   int           `json:"max_active_conns"`
	IdleTimeout      time.Duration `json:"idle_timeout"`
	ConnectTimeout   time.Duration `json:"connect_timeout"`
	ReadTimeout      time.Duration `json:"read_timeout"`
	WriteTimeout     time.Duration `json:"write_timeout"`
}

type ServerEnvConfig struct {
	Port        int           `json:"port"`
	MetricsPort int           `json:"metrics_port"`
	GRPCPort    int           `json:"grpc_port"`
	Host        string        `json:"host"`
	Timeout     time.Duration `json:"timeout"`
	EnableAuth  bool          `json:"enable_auth"`
}

type SecurityEnvConfig struct {
	JWTSecret           string   `json:"jwt_secret"`
	APIKeys             []string `json:"api_keys"`
	JWTExpiry           time.Duration `json:"jwt_expiry"`
	CORSAllowedOrigins  []string `json:"cors_allowed_origins"`
	TLSEnabled          bool     `json:"tls_enabled"`
	TLSCertFile         string   `json:"tls_cert_file"`
	TLSKeyFile          string   `json:"tls_key_file"`
}

type QueueEnvConfig struct {
	DefaultVisibilityTimeout time.Duration `json:"default_visibility_timeout"`
	DefaultMaxRetries        int           `json:"default_max_retries"`
	DefaultMessageRetention  time.Duration `json:"default_message_retention"`
	CleanupInterval          time.Duration `json:"cleanup_interval"`
	MaxStreamLength          int64         `json:"max_stream_length"`
	ConsumerTimeout          time.Duration `json:"consumer_timeout"`
	MaxMessageSize           string        `json:"max_message_size"`
	BatchSize                int           `json:"batch_size"`
}

type DatabaseEnvConfig struct {
	PostgresURL  string `json:"postgres_url"`
	MySQLURL     string `json:"mysql_url"`
	MongoURL     string `json:"mongo_url"`
	PostgresHost string `json:"postgres_host"`
	PostgresPort int    `json:"postgres_port"`
	PostgresUser string `json:"postgres_user"`
	PostgresPass string `json:"postgres_password"`
	PostgresDB   string `json:"postgres_database"`
	MySQLHost    string `json:"mysql_host"`
	MySQLPort    int    `json:"mysql_port"`
	MySQLUser    string `json:"mysql_user"`
	MySQLPass    string `json:"mysql_password"`
	MySQLDB      string `json:"mysql_database"`
	MongoHost    string `json:"mongo_host"`
	MongoPort    int    `json:"mongo_port"`
	MongoDB      string `json:"mongo_database"`
	MongoUser    string `json:"mongo_username"`
	MongoPass    string `json:"mongo_password"`
}

type ObservabilityEnvConfig struct {
	PrometheusEnabled  bool    `json:"prometheus_enabled"`
	PrometheusPort     int     `json:"prometheus_port"`
	TracingEnabled     bool    `json:"tracing_enabled"`
	JaegerEndpoint     string  `json:"jaeger_endpoint"`
	JaegerAgentHost    string  `json:"jaeger_agent_host"`
	JaegerAgentPort    int     `json:"jaeger_agent_port"`
	SamplingRatio      float64 `json:"sampling_ratio"`
	OTELServiceName    string  `json:"otel_service_name"`
	OTELServiceVersion string  `json:"otel_service_version"`
	GrafanaURL         string  `json:"grafana_url"`
	GrafanaUsername    string  `json:"grafana_username"`
	GrafanaPassword    string  `json:"grafana_password"`
}

type LoggingEnvConfig struct {
	Level       string `json:"level"`
	Format      string `json:"format"`
	FilePath    string `json:"file_path"`
	MaxSize     string `json:"max_size"`
	MaxBackups  int    `json:"max_backups"`
	MaxAge      int    `json:"max_age"`
	Compress    bool   `json:"compress"`
}

type DevelopmentEnvConfig struct {
	Environment      string `json:"environment"`
	Debug            bool   `json:"debug"`
	TestRedisURL     string `json:"test_redis_url"`
	TestDatabaseURL  string `json:"test_database_url"`
	EnablePprof      bool   `json:"enable_pprof"`
	PprofPort        int    `json:"pprof_port"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*EnvConfig, error) {
	// Load .env files
	LoadEnv()
	
	config := &EnvConfig{
		Redis: RedisEnvConfig{
			Addresses:        getStringSlice("REDIS_ADDRESSES", []string{"localhost:6379"}),
			Password:         getEnv("REDIS_PASSWORD", ""),
			DB:               getEnvInt("REDIS_DB", 0),
			Username:         getEnv("REDIS_USERNAME", ""),
			ClusterEnabled:   getEnvBool("REDIS_CLUSTER_ENABLED", false),
			TLSEnabled:       getEnvBool("REDIS_TLS_ENABLED", false),
			TLSCertFile:      getEnv("REDIS_TLS_CERT_FILE", ""),
			TLSKeyFile:       getEnv("REDIS_TLS_KEY_FILE", ""),
			TLSCAFile:        getEnv("REDIS_TLS_CA_FILE", ""),
			TLSInsecure:      getEnvBool("REDIS_TLS_INSECURE", false),
			MaxIdleConns:     getEnvInt("REDIS_MAX_IDLE_CONNS", 10),
			MaxActiveConns:   getEnvInt("REDIS_MAX_ACTIVE_CONNS", 100),
			IdleTimeout:      getEnvDuration("REDIS_IDLE_TIMEOUT", 300*time.Second),
			ConnectTimeout:   getEnvDuration("REDIS_CONNECT_TIMEOUT", 5*time.Second),
			ReadTimeout:      getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:     getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Server: ServerEnvConfig{
			Port:        getEnvInt("SERVER_PORT", 8080),
			MetricsPort: getEnvInt("SERVER_METRICS_PORT", 9090),
			GRPCPort:    getEnvInt("SERVER_GRPC_PORT", 9090),
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			Timeout:     getEnvDuration("SERVER_TIMEOUT", 30*time.Second),
			EnableAuth:  getEnvBool("SERVER_ENABLE_AUTH", false),
		},
		Security: SecurityEnvConfig{
			JWTSecret:          getEnv("JWT_SECRET", "change-this-secret-key"),
			APIKeys:            getStringSlice("API_KEYS", []string{"admin-key:admin"}),
			JWTExpiry:          getEnvDuration("JWT_EXPIRY", 24*time.Hour),
			CORSAllowedOrigins: getStringSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
			TLSEnabled:         getEnvBool("TLS_ENABLED", false),
			TLSCertFile:        getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:         getEnv("TLS_KEY_FILE", ""),
		},
		Queue: QueueEnvConfig{
			DefaultVisibilityTimeout: getEnvDuration("DEFAULT_VISIBILITY_TIMEOUT", 30*time.Second),
			DefaultMaxRetries:        getEnvInt("DEFAULT_MAX_RETRIES", 3),
			DefaultMessageRetention:  getEnvDuration("DEFAULT_MESSAGE_RETENTION", 7*24*time.Hour),
			CleanupInterval:          getEnvDuration("CLEANUP_INTERVAL", 5*time.Minute),
			MaxStreamLength:          int64(getEnvInt("MAX_STREAM_LENGTH", 10000)),
			ConsumerTimeout:          getEnvDuration("CONSUMER_TIMEOUT", 60*time.Second),
			MaxMessageSize:           getEnv("MAX_MESSAGE_SIZE", "1MB"),
			BatchSize:                getEnvInt("BATCH_SIZE", 100),
		},
		Database: DatabaseEnvConfig{
			PostgresURL:  getEnv("POSTGRES_URL", ""),
			MySQLURL:     getEnv("MYSQL_URL", ""),
			MongoURL:     getEnv("MONGO_URL", ""),
			PostgresHost: getEnv("POSTGRES_HOST", "localhost"),
			PostgresPort: getEnvInt("POSTGRES_PORT", 5432),
			PostgresUser: getEnv("POSTGRES_USER", "postgres"),
			PostgresPass: getEnv("POSTGRES_PASSWORD", ""),
			PostgresDB:   getEnv("POSTGRES_DATABASE", "queue_metadata"),
			MySQLHost:    getEnv("MYSQL_HOST", "localhost"),
			MySQLPort:    getEnvInt("MYSQL_PORT", 3306),
			MySQLUser:    getEnv("MYSQL_USER", "root"),
			MySQLPass:    getEnv("MYSQL_PASSWORD", ""),
			MySQLDB:      getEnv("MYSQL_DATABASE", "queue_metadata"),
			MongoHost:    getEnv("MONGO_HOST", "localhost"),
			MongoPort:    getEnvInt("MONGO_PORT", 27017),
			MongoDB:      getEnv("MONGO_DATABASE", "queue_system"),
			MongoUser:    getEnv("MONGO_USERNAME", ""),
			MongoPass:    getEnv("MONGO_PASSWORD", ""),
		},
		Observability: ObservabilityEnvConfig{
			PrometheusEnabled:  getEnvBool("PROMETHEUS_ENABLED", true),
			PrometheusPort:     getEnvInt("PROMETHEUS_PORT", 2112),
			TracingEnabled:     getEnvBool("TRACING_ENABLED", true),
			JaegerEndpoint:     getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
			JaegerAgentHost:    getEnv("JAEGER_AGENT_HOST", "localhost"),
			JaegerAgentPort:    getEnvInt("JAEGER_AGENT_PORT", 6831),
			SamplingRatio:      getEnvFloat("JAEGER_SAMPLER_PARAM", 0.1),
			OTELServiceName:    getEnv("OTEL_SERVICE_NAME", "redis-queue-system"),
			OTELServiceVersion: getEnv("OTEL_SERVICE_VERSION", "1.0.0"),
			GrafanaURL:         getEnv("GRAFANA_URL", "http://localhost:3000"),
			GrafanaUsername:    getEnv("GRAFANA_USERNAME", "admin"),
			GrafanaPassword:    getEnv("GRAFANA_PASSWORD", "admin"),
		},
		Logging: LoggingEnvConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			FilePath:   getEnv("LOG_FILE_PATH", "./logs/queue.log"),
			MaxSize:    getEnv("LOG_MAX_SIZE", "100MB"),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 30),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		},
		Development: DevelopmentEnvConfig{
			Environment:     getEnv("ENVIRONMENT", "development"),
			Debug:           getEnvBool("DEBUG", false),
			TestRedisURL:    getEnv("TEST_REDIS_URL", "redis://localhost:6380"),
			TestDatabaseURL: getEnv("TEST_DATABASE_URL", ""),
			EnablePprof:     getEnvBool("ENABLE_PPROF", false),
			PprofPort:       getEnvInt("PPROF_PORT", 6060),
		},
	}
	
	return config, nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// ValidateConfig validates the loaded configuration
func (c *EnvConfig) ValidateConfig() error {
	errors := []string{}
	
	// Validate Redis configuration
	if len(c.Redis.Addresses) == 0 {
		errors = append(errors, "REDIS_ADDRESSES is required")
	}
	
	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, "SERVER_PORT must be between 1 and 65535")
	}
	
	// Validate security configuration
	if c.Security.JWTSecret == "change-this-secret-key" {
		errors = append(errors, "JWT_SECRET must be changed from default value")
	}
	
	if len(c.Security.JWTSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters long")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// Print prints the configuration (with sensitive values masked)
func (c *EnvConfig) Print() {
	fmt.Println("=== Configuration ===")
	fmt.Printf("Environment: %s\n", c.Development.Environment)
	fmt.Printf("Server Port: %d\n", c.Server.Port)
	fmt.Printf("Redis Addresses: %v\n", c.Redis.Addresses)
	fmt.Printf("Prometheus Enabled: %t\n", c.Observability.PrometheusEnabled)
	fmt.Printf("Tracing Enabled: %t\n", c.Observability.TracingEnabled)
	fmt.Printf("Debug Mode: %t\n", c.Development.Debug)
	fmt.Printf("JWT Secret: %s\n", maskSecret(c.Security.JWTSecret))
	fmt.Println("====================")
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***" + secret[len(secret)-4:]
}