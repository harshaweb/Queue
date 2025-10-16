# Environment Configuration Guide

The Redis Queue System supports comprehensive environment-based configuration through `.env` files and environment variables.

## 📁 Environment Files

The system looks for configuration in this order:

1. `.env.local` (highest priority, git-ignored)
2. `.env` (default configuration, git-ignored)
3. `.env.example` (template file, committed to git)
4. Environment variables (override everything)

## 🚀 Quick Start

1. **Copy the example file:**
   ```bash
   cp .env.example .env.local
   ```

2. **Edit your local configuration:**
   ```bash
   # Edit .env.local with your settings
   notepad .env.local  # Windows
   nano .env.local     # Linux/Mac
   ```

3. **Set required values:**
   ```env
   # Required: Change the JWT secret
   JWT_SECRET=your-super-secure-32-character-secret

   # Required: Redis connection
   REDIS_ADDRESSES=localhost:6379
   REDIS_PASSWORD=your-redis-password

   # Optional: Database URLs
   POSTGRES_URL=postgresql://user:pass@localhost:5432/db
   ```

## 📋 Configuration Categories

### 🔴 Required Configuration

These **MUST** be configured before production use:

```env
# Security (CRITICAL)
JWT_SECRET=your-super-secure-32-character-secret-key
API_KEYS=admin-key:admin,user-key:user,readonly-key:readonly

# Redis Connection
REDIS_ADDRESSES=localhost:6379
REDIS_PASSWORD=your-redis-password
```

### 🟡 Recommended Configuration

```env
# Server Settings
SERVER_PORT=8080
SERVER_METRICS_PORT=9090

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Monitoring
PROMETHEUS_ENABLED=true
TRACING_ENABLED=true
GRAFANA_PASSWORD=secure-grafana-password
```

### 🟢 Optional Configuration

```env
# Database URLs (for future features)
POSTGRES_URL=postgresql://user:pass@host:5432/db
MYSQL_URL=mysql://user:pass@host:3306/db
MONGO_URL=mongodb://host:27017/db

# Advanced Features
ENABLE_BATCH_OPERATIONS=true
ENABLE_DELAYED_MESSAGES=true
ENABLE_MESSAGE_ENCRYPTION=false
```

## 🔧 Configuration Loading

### In Your Go Code

```go
package main

import (
    "github.com/harshaweb/Queue/internal/config"
    "github.com/harshaweb/Queue/pkg/client"
)

func main() {
    // Load environment configuration
    envConfig, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Validate configuration
    if err := envConfig.ValidateConfig(); err != nil {
        log.Printf("Config validation warning: %v", err)
    }

    // Print configuration (secrets masked)
    envConfig.Print()

    // Use in client
    clientConfig := client.DefaultConfig()
    clientConfig.RedisConfig.Addresses = envConfig.Redis.Addresses
    clientConfig.RedisConfig.Password = envConfig.Redis.Password

    client, err := client.NewClient(clientConfig)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Testing Configuration

Run the environment test utility:

```bash
go run examples/env-test/main.go
```

This will:
- ✅ Load your `.env` files
- ✅ Validate configuration
- ✅ Print current settings (with secrets masked)
- ✅ Show warnings for missing required values

## 🗂️ Complete Configuration Reference

### Redis Configuration
```env
REDIS_ADDRESSES=localhost:6379,localhost:6380  # Comma-separated addresses
REDIS_PASSWORD=your-password                    # Redis password
REDIS_DB=0                                      # Redis database number
REDIS_USERNAME=default                          # Redis ACL username
REDIS_CLUSTER_ENABLED=false                     # Enable Redis cluster mode
REDIS_TLS_ENABLED=false                         # Enable TLS connection
REDIS_TLS_CERT_FILE=./cert.pem                 # TLS certificate file
REDIS_TLS_KEY_FILE=./key.pem                   # TLS key file
REDIS_TLS_CA_FILE=./ca.pem                     # TLS CA file
REDIS_MAX_IDLE_CONNS=10                        # Max idle connections
REDIS_MAX_ACTIVE_CONNS=100                     # Max active connections
REDIS_IDLE_TIMEOUT=300s                        # Idle connection timeout
```

### Server Configuration
```env
SERVER_PORT=8080                               # HTTP API port
SERVER_METRICS_PORT=9090                       # Metrics/gRPC port
SERVER_HOST=0.0.0.0                           # Bind host
SERVER_TIMEOUT=30s                             # Request timeout
SERVER_ENABLE_AUTH=true                        # Enable authentication
```

### Security Configuration
```env
JWT_SECRET=your-32-char-secret                 # JWT signing key (REQUIRED)
API_KEYS=key1:role1,key2:role2                # API key mappings
JWT_EXPIRY=24h                                 # JWT token expiry
CORS_ALLOWED_ORIGINS=http://localhost:3000     # CORS origins
TLS_ENABLED=false                              # Enable HTTPS
TLS_CERT_FILE=./server.crt                    # TLS certificate
TLS_KEY_FILE=./server.key                     # TLS private key
```

### Database URLs
```env
# PostgreSQL
POSTGRES_URL=postgresql://user:pass@host:5432/db
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_DATABASE=queue_metadata

# MySQL
MYSQL_URL=mysql://user:pass@host:3306/db
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=password
MYSQL_DATABASE=queue_metadata

# MongoDB
MONGO_URL=mongodb://host:27017/db
MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_DATABASE=queue_system
MONGO_USERNAME=username
MONGO_PASSWORD=password
```

### Monitoring Configuration
```env
# Prometheus Metrics
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=2112
METRICS_NAMESPACE=redis_queue

# Distributed Tracing
TRACING_ENABLED=true
JAEGER_ENDPOINT=http://localhost:14268/api/traces
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831
JAEGER_SAMPLER_PARAM=0.1
OTEL_SERVICE_NAME=redis-queue-system

# Grafana
GRAFANA_URL=http://localhost:3000
GRAFANA_USERNAME=admin
GRAFANA_PASSWORD=secure-password
```

### Queue Configuration
```env
DEFAULT_VISIBILITY_TIMEOUT=30s                 # Message visibility timeout
DEFAULT_MAX_RETRIES=3                          # Max retry attempts
DEFAULT_MESSAGE_RETENTION=7d                   # Message retention period
CLEANUP_INTERVAL=5m                            # Cleanup job interval
MAX_STREAM_LENGTH=10000                        # Redis stream max length
CONSUMER_TIMEOUT=60s                           # Consumer timeout
MAX_MESSAGE_SIZE=1MB                           # Max message size
BATCH_SIZE=100                                 # Batch operation size
```

## 🛡️ Security Best Practices

1. **Never commit `.env` files** - They're git-ignored by default
2. **Use strong JWT secrets** - At least 32 characters, random
3. **Rotate API keys regularly** - Use different keys per environment
4. **Enable TLS in production** - Set `TLS_ENABLED=true`
5. **Use strong database passwords** - Random, unique passwords
6. **Limit CORS origins** - Don't use `*` in production

## 🚦 Environment-Specific Configurations

### Development
```env
ENVIRONMENT=development
DEBUG=true
LOG_LEVEL=debug
REDIS_ADDRESSES=localhost:6379
PROMETHEUS_ENABLED=false
TRACING_ENABLED=false
```

### Staging
```env
ENVIRONMENT=staging
DEBUG=false
LOG_LEVEL=info
REDIS_ADDRESSES=staging-redis:6379
PROMETHEUS_ENABLED=true
TRACING_ENABLED=true
TLS_ENABLED=true
```

### Production
```env
ENVIRONMENT=production
DEBUG=false
LOG_LEVEL=warn
REDIS_ADDRESSES=prod-redis-cluster:6379,prod-redis-cluster:6380
REDIS_CLUSTER_ENABLED=true
REDIS_TLS_ENABLED=true
PROMETHEUS_ENABLED=true
TRACING_ENABLED=true
TLS_ENABLED=true
SERVER_ENABLE_AUTH=true
```

## 🔍 Troubleshooting

### Configuration Not Loading
```bash
# Test configuration loading
go run examples/env-test/main.go

# Check file permissions
ls -la .env.local

# Verify file format (no spaces around =)
cat .env.local | grep "="
```

### Validation Errors
```bash
# Common issues:
# 1. JWT_SECRET too short (needs 32+ chars)
# 2. Invalid port numbers (1-65535)
# 3. Missing REDIS_ADDRESSES
```

### Environment Variables Not Working
```bash
# Check environment variables
printenv | grep REDIS
printenv | grep JWT

# Test manual override
REDIS_ADDRESSES=test:6379 go run examples/env-test/main.go
```

## 📚 Examples

See the `examples/` directory for complete usage examples:
- `examples/demo/` - Basic usage with environment config
- `examples/env-test/` - Configuration testing utility
- `examples/sdk-usage/` - Advanced SDK usage with environment config

## 🔗 Related Documentation

- [Docker Deployment](../deploy/README.md) - Using environment files with Docker
- [Kubernetes Deployment](../deploy/k8s/README.md) - ConfigMaps and Secrets
- [Security Guide](./SECURITY.md) - Security configuration details
- [Monitoring Guide](./MONITORING.md) - Observability configuration