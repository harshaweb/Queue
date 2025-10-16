# Redis Queue System

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Redis](https://img.shields.io/badge/Redis-7.0+-red.svg)](https://redis.io)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.25+-blue.svg)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A **production-ready, high-performance message queueing system** built on Redis Streams with Go, designed for massive scale and high availability in Kubernetes environments.

## 🚀 Features

### Core Queue Operations
- ✅ **Basic Operations**: Enqueue, dequeue, acknowledge, negative acknowledge
- ✅ **Advanced Operations**: Skip, move, requeue, dead letter queue
- ✅ **Batch Operations**: High-throughput batch enqueue/dequeue
- ✅ **Scheduled Messages**: Delay and schedule messages for future delivery
- ✅ **Message Priorities**: Priority-based message processing
- ✅ **Retry Logic**: Configurable exponential/linear/fixed backoff strategies

### High Availability & Scaling
- ✅ **Redis Streams**: Built on Redis Streams with consumer groups
- ✅ **Horizontal Scaling**: Auto-scaling with Kubernetes HPA
- ✅ **High Availability**: Multi-replica deployments with leader election
- ✅ **Health Checks**: Comprehensive liveness and readiness probes
- ✅ **Circuit Breakers**: Resilience patterns for fault tolerance

### APIs & SDKs
- ✅ **Go SDK**: Full-featured client library with middleware support
- ✅ **REST API**: HTTP/JSON API for cross-language integration
- ✅ **gRPC API**: High-performance streaming for heavy workloads
- ✅ **CLI Tools**: Command-line interface for administration

### Enterprise Features
- ✅ **Authentication**: JWT and API key-based authentication
- ✅ **Authorization**: Role-based access control (RBAC)
- ✅ **Rate Limiting**: Configurable rate limiting and throttling
- ✅ **Observability**: Prometheus metrics, OpenTelemetry tracing, Grafana dashboards
- ✅ **Security**: TLS support, audit logging, secure defaults

## 📦 Installation

### Using Go Module (Recommended)

```bash
go mod init your-app
go get github.com/queue-system/redis-queue/pkg/client
```

### From Source

```bash
git clone https://github.com/queue-system/redis-queue.git
cd redis-queue
go mod download
```

## 🏃 Quick Start

### Using Quick Start Scripts

For the fastest setup, use our quick start scripts:

**Linux/macOS:**
```bash
# Make script executable
chmod +x scripts/quickstart.sh

# Start development environment
./scripts/quickstart.sh dev

# Start production environment (requires environment variables)
export JWT_SECRET="your-secret-key"
export API_KEYS="admin-key:admin,user-key:user"
export GRAFANA_PASSWORD="secure-password"
./scripts/quickstart.sh prod

# Run tests
./scripts/quickstart.sh test
```

**Windows:**
```cmd
# Start development environment
scripts\quickstart.bat dev

# Start production environment (requires environment variables)
set JWT_SECRET=your-secret-key
set API_KEYS=admin-key:admin,user-key:user
set GRAFANA_PASSWORD=secure-password
scripts\quickstart.bat prod

# Run tests
scripts\quickstart.bat test
```

### Manual Setup

#### 1. Prerequisites

**Redis Server** (Required)
```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:latest

# Using Redis directly
redis-server
```

**Go 1.21+** (Required)
```bash
go version  # Should be 1.21 or higher
```

### 2. Basic Usage with Go SDK

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/queue-system/redis-queue/pkg/client"
)

func main() {
    // Create client with default configuration
    config := client.DefaultConfig()
    config.RedisConfig.Addresses = []string{"localhost:6379"}
    
    queueClient, err := client.NewClient(config)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer queueClient.Close()

    ctx := context.Background()

    // Create a queue
    queueConfig := &client.QueueConfig{
        Name:              "my-queue",
        VisibilityTimeout: 30 * time.Second,
        MaxRetries:        3,
    }
    
    err = queueClient.CreateQueue(ctx, queueConfig)
    if err != nil {
        log.Fatal("Failed to create queue:", err)
    }

    // Send a message
    payload := map[string]interface{}{
        "user_id": 12345,
        "action":  "send_email",
        "email":   "user@example.com",
    }

    result, err := queueClient.Enqueue(ctx, "my-queue", payload)
    if err != nil {
        log.Fatal("Failed to enqueue message:", err)
    }
    
    fmt.Printf("Message enqueued with ID: %s\n", result.MessageID)

    // Consume messages
    handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
        fmt.Printf("Processing message: %+v\n", payload)
        
        // Simulate work
        time.Sleep(100 * time.Millisecond)
        
        // Acknowledge successful processing
        return client.Ack()
    }

    // Start consuming (this blocks)
    err = queueClient.Consume(ctx, "my-queue", handler,
        client.WithBatchSize(10),
        client.WithVisibilityTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal("Failed to consume messages:", err)
    }
}
```

### 3. Advanced Usage Examples

#### Scheduled Messages
```go
// Schedule a message for 1 hour from now
result, err := queueClient.Enqueue(ctx, "my-queue", payload,
    client.WithScheduleAt(time.Now().Add(1*time.Hour)),
)

// Delay a message by 5 minutes
result, err := queueClient.Enqueue(ctx, "my-queue", payload,
    client.WithDelay(5*time.Minute),
)
```

#### Priority Messages
```go
// High priority message
result, err := queueClient.Enqueue(ctx, "my-queue", payload,
    client.WithPriority(10),
)
```

#### Batch Operations
```go
// Batch enqueue multiple messages
payloads := []interface{}{
    map[string]interface{}{"id": 1, "task": "task1"},
    map[string]interface{}{"id": 2, "task": "task2"},
    map[string]interface{}{"id": 3, "task": "task3"},
}

result, err := queueClient.BatchEnqueue(ctx, "my-queue", payloads)
```

#### Error Handling and Retries
```go
handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
    data := payload.(map[string]interface{})
    
    // Simulate processing that might fail
    if shouldRetry := processMessage(data); shouldRetry {
        // Requeue with exponential backoff
        delay := time.Duration(msgCtx.RetryCount*msgCtx.RetryCount) * time.Second
        return client.Requeue(&delay)
    }
    
    // Move to dead letter queue if max retries exceeded
    if msgCtx.RetryCount >= 3 {
        return client.DeadLetter()
    }
    
    return client.Ack()
}
```

### 4. Using the Server & CLI

#### Start the Queue Server
```bash
# Build and run the server
go build -o queue-server ./cmd/queue-server
./queue-server

# Or with custom config
./queue-server --config config.yaml --port 8080
```

#### Use CLI Tools
```bash
# Build CLI
go build -o queue-cli ./cmd/queue-cli

# Create a queue
./queue-cli queue create my-queue --max-retries 3 --visibility-timeout 30s

# Send a message
./queue-cli message send my-queue '{"task":"hello","data":"world"}'

# Get queue statistics
./queue-cli queue stats my-queue

# List all queues
./queue-cli queue list

# Monitor queue activity
./queue-cli monitor my-queue
```

#### REST API Usage
```bash
# Create queue via REST API
curl -X POST http://localhost:8080/api/v1/queues \
  -H "Content-Type: application/json" \
  -d '{"name":"api-queue","visibility_timeout":"30s","max_retries":3}'

# Send message via REST API
curl -X POST http://localhost:8080/api/v1/queues/api-queue/messages \
  -H "Content-Type: application/json" \
  -d '{"payload":{"task":"process","id":123},"priority":5}'

# Get queue stats
curl http://localhost:8080/api/v1/queues/api-queue/stats
```

## 🐳 Docker Deployment

### Using Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'
services:
  redis:
    image: redis:latest
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    
  queue-server:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"  # gRPC
    depends_on:
      - redis
    environment:
      - REDIS_ADDRESSES=redis:6379
      - HTTP_PORT=8080
      - GRPC_PORT=9090
    volumes:
      - ./config.yaml:/app/config.yaml

volumes:
  redis_data:
```

Run with:
```bash
docker-compose up -d
```

## ☸️ Kubernetes Deployment

### Quick Deployment
```bash
# Apply Kubernetes manifests
kubectl apply -f deploy/k8s/

# Check deployment status
kubectl get pods -l app=queue-system

# Port forward for local access
kubectl port-forward svc/queue-server 8080:80
```

### Using Helm (Recommended)
```bash
# Install with Helm
helm install queue-system ./deploy/helm/queue-system \
  --set redis.persistence.enabled=true \
  --set server.replicas=3 \
  --set server.autoscaling.enabled=true

# Upgrade deployment
helm upgrade queue-system ./deploy/helm/queue-system \
  --set server.image.tag=v1.1.0

# Check status
helm status queue-system
```

## 📊 Monitoring & Observability

### Prometheus Metrics
The system exposes comprehensive metrics:
- Queue length and processing rates
- Message latency and throughput
- Error rates and retry counts
- Consumer and producer statistics
- System resource usage

```bash
# Access metrics endpoint
curl http://localhost:8080/metrics
```

### Grafana Dashboards
Pre-built dashboards are available in `deploy/monitoring/grafana/`:
- Queue System Overview
- Queue Performance
- Error Analysis
- Consumer Health

## 🧪 Testing

### Running Tests
```bash
# Run all tests
./scripts/run-tests.sh

# Run specific test types
./scripts/run-tests.sh unit
./scripts/run-tests.sh integration
./scripts/run-tests.sh benchmark
./scripts/run-tests.sh chaos

# On Windows
.\scripts\run-tests.bat all
```

### Performance Benchmarks
Expected performance characteristics:
- **Throughput**: 10,000+ messages/second
- **Latency**: <10ms for basic operations  
- **Concurrency**: 100+ concurrent consumers
- **Memory**: <500MB for typical workloads

## 🚀 Production Deployment

### Recommended Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Queue Server  │    │   Redis Cluster │
│    (HAProxy)    │────│   (3 replicas)  │────│   (3 nodes)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                       ┌─────────────────┐
                       │   Monitoring    │
                       │ (Prometheus +   │
                       │   Grafana)      │
                       └─────────────────┘
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Setup
```bash
# Clone repository
git clone https://github.com/queue-system/redis-queue.git
cd redis-queue

# Install dependencies
go mod download

# Run Redis for development
docker run -d --name redis-dev -p 6379:6379 redis:latest

# Run tests
make test

# Build project
make build
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: Full docs at [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/queue-system/redis-queue/issues)
- **Discussions**: [GitHub Discussions](https://github.com/queue-system/redis-queue/discussions)
- **Discord**: [Join our Discord](https://discord.gg/queue-system)

## 🏆 Acknowledgments

- Built on the excellent [Redis](https://redis.io) and [Go](https://golang.org) ecosystems
- Inspired by AWS SQS, RabbitMQ, and Apache Kafka
- Thanks to all [contributors](CONTRIBUTORS.md)

---

**Built with ❤️ for high-scale, reliable message processing**