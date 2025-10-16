package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	"github.com/harshaweb/Queue/internal/api"
	"github.com/harshaweb/Queue/internal/auth"
	"github.com/harshaweb/Queue/internal/config"
	"github.com/harshaweb/Queue/internal/metrics"
	"github.com/harshaweb/Queue/pkg/redis"
)

// Config represents the server configuration
type Config struct {
	Server struct {
		Port        int           `mapstructure:"port"`
		MetricsPort int           `mapstructure:"metrics_port"`
		Timeout     time.Duration `mapstructure:"timeout"`
		EnableAuth  bool          `mapstructure:"enable_auth"`
	} `mapstructure:"server"`

	Redis redis.Config `mapstructure:"redis"`

	Queue struct {
		DefaultVisibilityTimeout time.Duration `mapstructure:"default_visibility_timeout"`
		DefaultMaxRetries        int           `mapstructure:"default_max_retries"`
		CleanupInterval          time.Duration `mapstructure:"cleanup_interval"`
		MaxStreamLength          int64         `mapstructure:"max_stream_length"`
		ConsumerTimeout          time.Duration `mapstructure:"consumer_timeout"`
	} `mapstructure:"queue"`

	Observability struct {
		MetricsEnabled bool    `mapstructure:"metrics_enabled"`
		TracingEnabled bool    `mapstructure:"tracing_enabled"`
		JaegerEndpoint string  `mapstructure:"jaeger_endpoint"`
		SamplingRatio  float64 `mapstructure:"sampling_ratio"`
	} `mapstructure:"observability"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: struct {
			Port        int           `mapstructure:"port"`
			MetricsPort int           `mapstructure:"metrics_port"`
			Timeout     time.Duration `mapstructure:"timeout"`
			EnableAuth  bool          `mapstructure:"enable_auth"`
		}{
			Port:        8080,
			MetricsPort: 9090,
			Timeout:     30 * time.Second,
			EnableAuth:  false,
		},
		Redis: *redis.DefaultConfig(),
		Queue: struct {
			DefaultVisibilityTimeout time.Duration `mapstructure:"default_visibility_timeout"`
			DefaultMaxRetries        int           `mapstructure:"default_max_retries"`
			CleanupInterval          time.Duration `mapstructure:"cleanup_interval"`
			MaxStreamLength          int64         `mapstructure:"max_stream_length"`
			ConsumerTimeout          time.Duration `mapstructure:"consumer_timeout"`
		}{
			DefaultVisibilityTimeout: 30 * time.Second,
			DefaultMaxRetries:        3,
			CleanupInterval:          5 * time.Minute,
			MaxStreamLength:          10000,
			ConsumerTimeout:          60 * time.Second,
		},
		Observability: struct {
			MetricsEnabled bool    `mapstructure:"metrics_enabled"`
			TracingEnabled bool    `mapstructure:"tracing_enabled"`
			JaegerEndpoint string  `mapstructure:"jaeger_endpoint"`
			SamplingRatio  float64 `mapstructure:"sampling_ratio"`
		}{
			MetricsEnabled: true,
			TracingEnabled: true,
			JaegerEndpoint: "http://localhost:14268/api/traces",
			SamplingRatio:  0.1,
		},
		Logging: struct {
			Level  string `mapstructure:"level"`
			Format string `mapstructure:"format"`
		}{
			Level:  "info",
			Format: "json",
		},
	}
}

// Server represents the queue server
type Server struct {
	config          *Config
	httpServer      *http.Server
	metricsServer   *http.Server
	queueManager    *redis.StreamQueue
	metricsProvider *metrics.Metrics
	tracingProvider *metrics.TracingProvider
	authenticator   auth.Authenticator
}

// NewServer creates a new server instance
func NewServer(config *Config) (*Server, error) {
	// Create Redis client
	redisClient, err := redis.NewClient(&config.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Test Redis connection
	if err := redisClient.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create queue manager
	queueConfig := &redis.StreamQueueConfig{
		DefaultVisibilityTimeout: config.Queue.DefaultVisibilityTimeout,
		DefaultMaxRetries:        config.Queue.DefaultMaxRetries,
		CleanupInterval:          config.Queue.CleanupInterval,
		MaxStreamLength:          config.Queue.MaxStreamLength,
		ConsumerTimeout:          config.Queue.ConsumerTimeout,
	}
	queueManager := redis.NewStreamQueue(redisClient, queueConfig)

	// Create metrics provider
	var metricsProvider *metrics.Metrics
	if config.Observability.MetricsEnabled {
		metricsProvider = metrics.NewMetrics()
	}

	// Create tracing provider
	var tracingProvider *metrics.TracingProvider
	if config.Observability.TracingEnabled {
		tracingConfig := &metrics.TracingConfig{
			ServiceName:    "redis-queue-server",
			ServiceVersion: "1.0.0",
			Environment:    "production",
			JaegerEndpoint: config.Observability.JaegerEndpoint,
			SamplingRatio:  config.Observability.SamplingRatio,
			Enabled:        true,
		}
		tracingProvider, err = metrics.NewTracingProvider(tracingConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create tracing provider: %w", err)
		}
	}

	// Create authenticator
	var authenticator auth.Authenticator
	if config.Server.EnableAuth {
		// In production, you would configure JWT or API key auth
		// For now, using no-op authenticator
		authenticator = auth.NewNoOpAuthenticator()
	} else {
		authenticator = auth.NewNoOpAuthenticator()
	}

	return &Server{
		config:          config,
		queueManager:    queueManager,
		metricsProvider: metricsProvider,
		tracingProvider: tracingProvider,
		authenticator:   authenticator,
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Start cleanup routine
	go s.startCleanupRoutine()

	// Start metrics collection routine
	if s.metricsProvider != nil {
		go s.startMetricsCollection()
	}

	// Setup HTTP server
	if err := s.setupHTTPServer(); err != nil {
		return fmt.Errorf("failed to setup HTTP server: %w", err)
	}

	// Setup metrics server
	if err := s.setupMetricsServer(); err != nil {
		return fmt.Errorf("failed to setup metrics server: %w", err)
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on port %d", s.config.Server.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start metrics server
	go func() {
		log.Printf("Starting metrics server on port %d", s.config.Server.MetricsPort)
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Shutdown metrics server
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
		}
	}

	// Close tracing provider
	if s.tracingProvider != nil {
		if err := s.tracingProvider.Close(ctx); err != nil {
			log.Printf("Tracing provider shutdown error: %v", err)
		}
	}

	// Close queue manager
	if s.queueManager != nil {
		if err := s.queueManager.Close(); err != nil {
			log.Printf("Queue manager shutdown error: %v", err)
		}
	}

	log.Println("Server shutdown complete")
	return nil
}

func (s *Server) setupHTTPServer() error {
	// Create REST API
	restConfig := &api.RestConfig{
		EnableAuth:     s.config.Server.EnableAuth,
		AllowedOrigins: []string{"*"}, // Configure as needed
		RateLimit:      100,           // requests per minute
		Timeout:        s.config.Server.Timeout,
	}

	restAPI := api.NewRestAPI(s.queueManager, s.authenticator, restConfig)
	router := restAPI.SetupRoutes()

	// Add middleware for metrics and tracing
	if s.metricsProvider != nil {
		router.Use(s.metricsMiddleware())
	}

	if s.tracingProvider != nil {
		router.Use(s.tracingMiddleware())
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:      router,
		ReadTimeout:  s.config.Server.Timeout,
		WriteTimeout: s.config.Server.Timeout,
		IdleTimeout:  2 * s.config.Server.Timeout,
	}

	return nil
}

func (s *Server) setupMetricsServer() error {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health check endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := s.queueManager.HealthCheck(r.Context()); err != nil {
			http.Error(w, "Unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	s.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Server.MetricsPort),
		Handler: mux,
	}

	return nil
}

func (s *Server) startCleanupRoutine() {
	ticker := time.NewTicker(s.config.Queue.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := s.queueManager.CleanupExpiredMessages(ctx); err != nil {
				log.Printf("Cleanup failed: %v", err)
			}
			cancel()
		}
	}
}

func (s *Server) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			// Update system uptime
			s.metricsProvider.UpdateSystemUptime(time.Since(startTime))

			// Update active queues count
			queues, err := s.queueManager.ListQueues(context.Background())
			if err == nil {
				s.metricsProvider.UpdateActiveQueues(float64(len(queues)))

				// Update queue-specific metrics
				for _, queueName := range queues {
					stats, err := s.queueManager.GetQueueStats(context.Background(), queueName)
					if err == nil {
						s.metricsProvider.UpdateQueueLength(queueName, float64(stats.Length))
						s.metricsProvider.UpdateInFlightMessages(queueName, float64(stats.InFlight))
						s.metricsProvider.UpdateConsumerCount(queueName, float64(stats.ConsumerCount))
						s.metricsProvider.UpdateDeadLetterSize(queueName, float64(stats.DeadLetterSize))
					}
				}
			}
		}
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		duration := time.Since(startTime)
		status := "success"
		if c.Writer.Status() >= 400 {
			status = "error"
		}

		s.metricsProvider.RecordOperation(
			c.Request.Method,
			c.FullPath(),
			status,
			duration,
		)
	}
}

func (s *Server) tracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := s.tracingProvider.StartSpan(
			c.Request.Context(),
			fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if c.Writer.Status() >= 400 {
			s.tracingProvider.SetSpanStatus(ctx, 2, "HTTP Error") // StatusCodeError = 2
		}
	}
}

func main() {
	// Load environment configuration first
	envConfig, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load environment config: %v", err)
	}

	// Validate configuration
	if envConfig != nil {
		if err := envConfig.ValidateConfig(); err != nil {
			log.Printf("Configuration validation warning: %v", err)
		}
		envConfig.Print()
	}

	// Load configuration
	config := DefaultConfig()

	// Override with environment values if available
	if envConfig != nil {
		config.Server.Port = envConfig.Server.Port
		config.Server.MetricsPort = envConfig.Server.MetricsPort
		config.Server.Timeout = envConfig.Server.Timeout
		config.Server.EnableAuth = envConfig.Server.EnableAuth

		config.Redis.Addresses = envConfig.Redis.Addresses
		config.Redis.Password = envConfig.Redis.Password
		config.Redis.DB = envConfig.Redis.DB

		config.Queue.DefaultVisibilityTimeout = envConfig.Queue.DefaultVisibilityTimeout
		config.Queue.DefaultMaxRetries = envConfig.Queue.DefaultMaxRetries
		config.Queue.CleanupInterval = envConfig.Queue.CleanupInterval
		config.Queue.MaxStreamLength = envConfig.Queue.MaxStreamLength

		config.Observability.MetricsEnabled = envConfig.Observability.PrometheusEnabled
		config.Observability.TracingEnabled = envConfig.Observability.TracingEnabled
		config.Observability.JaegerEndpoint = envConfig.Observability.JaegerEndpoint

		config.Logging.Level = envConfig.Logging.Level
		config.Logging.Format = envConfig.Logging.Format
	}

	// Load config from file if specified
	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Failed to read config file: %v", err)
		} else {
			if err := viper.Unmarshal(config); err != nil {
				log.Fatalf("Failed to unmarshal config: %v", err)
			}
		}
	}

	// Override with environment variables
	viper.AutomaticEnv()

	// Create and start server
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Failed to stop server gracefully: %v", err)
	}
}
