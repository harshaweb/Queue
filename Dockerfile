# Multi-stage build for Redis Queue System
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the queue server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o queue-server ./cmd/queue-server

# Build the CLI tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o queue-cli ./cmd/queue-cli

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/queue-server .
COPY --from=builder /app/queue-cli .

# Create directories
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090 2112

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["./queue-server"]

# Labels
LABEL maintainer="Queue System Team"
LABEL version="1.0.0"
LABEL description="Production-ready Redis-backed queueing system"