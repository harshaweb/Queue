package main

import (
	"fmt"
	"log"

	"github.com/harshaweb/Queue/internal/config"
)

func main() {
	fmt.Println("🔧 Environment Configuration Test")
	fmt.Println("=================================")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Print configuration
	cfg.Print()

	// Validate configuration
	if err := cfg.ValidateConfig(); err != nil {
		fmt.Printf("\n⚠️  Configuration warnings: %v\n", err)
	} else {
		fmt.Printf("\n✅ Configuration is valid!\n")
	}

	// Test specific configurations
	fmt.Println("\n📊 Configuration Details:")
	fmt.Printf("Redis Addresses: %v\n", cfg.Redis.Addresses)
	fmt.Printf("Server Port: %d\n", cfg.Server.Port)
	fmt.Printf("Metrics Port: %d\n", cfg.Server.MetricsPort)
	fmt.Printf("Database URLs:\n")
	if cfg.Database.PostgresURL != "" {
		fmt.Printf("  - PostgreSQL: %s\n", cfg.Database.PostgresURL)
	}
	if cfg.Database.MySQLURL != "" {
		fmt.Printf("  - MySQL: %s\n", cfg.Database.MySQLURL)
	}
	if cfg.Database.MongoURL != "" {
		fmt.Printf("  - MongoDB: %s\n", cfg.Database.MongoURL)
	}

	fmt.Printf("Monitoring:\n")
	fmt.Printf("  - Prometheus: %t (port %d)\n", cfg.Observability.PrometheusEnabled, cfg.Observability.PrometheusPort)
	fmt.Printf("  - Tracing: %t\n", cfg.Observability.TracingEnabled)
	fmt.Printf("  - Jaeger: %s\n", cfg.Observability.JaegerEndpoint)

	fmt.Println("\n✅ Environment configuration test completed!")
}
