# Docker Build and Registry Push Guide

This guide explains how to build and push Docker images for the Redis Queue System.

## 🚀 Quick Start

### Prerequisites

1. **Docker Desktop** installed and running
2. **Git** for version information
3. **Registry Access** (GitHub Container Registry recommended)

### Login to Registry

**GitHub Container Registry (Recommended):**
```bash
# Create a Personal Access Token with 'write:packages' scope
# Then login:
docker login ghcr.io -u YOUR_GITHUB_USERNAME
```

**Docker Hub:**
```bash
docker login
```

## 🏗️ Building Images

### Option 1: Automated Build Script (Windows)

```cmd
# Simple local build
scripts\docker-build-simple.bat

# Full build with registry push
scripts\docker-build.bat latest ghcr.io/harshaweb
```

### Option 2: Manual Docker Commands

```bash
# Basic build
docker build -t ghcr.io/harshaweb/queue:latest .

# Build with version info
docker build \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VERSION=$(git describe --tags --always) \
  -t ghcr.io/harshaweb/queue:latest \
  -t ghcr.io/harshaweb/queue:$(git rev-parse --short HEAD) \
  .
```

### Option 3: Docker Compose Build

```bash
# Build and run locally
docker-compose -f docker-compose.build.yml build

# Build and run with monitoring stack
docker-compose -f docker-compose.build.yml --profile monitoring up --build
```

## 📤 Pushing to Registry

### Manual Push

```bash
# Push latest tag
docker push ghcr.io/harshaweb/queue:latest

# Push specific version
docker push ghcr.io/harshaweb/queue:$(git rev-parse --short HEAD)

# Push all tags
docker image ls ghcr.io/harshaweb/queue --format "{{.Repository}}:{{.Tag}}" | xargs -I {} docker push {}
```

### Automated Push (CI/CD)

The repository includes GitHub Actions workflow that automatically:
- ✅ Builds on every push to `main`
- ✅ Creates multi-architecture images (amd64, arm64)
- ✅ Pushes to GitHub Container Registry
- ✅ Tags with branch name, commit SHA, and `latest`

## 🏷️ Image Tags

The build system creates multiple tags:

- `latest` - Latest build from main branch
- `main` - Latest build from main branch  
- `{commit-sha}` - Specific commit build
- `v{version}` - Tagged releases
- `pr-{number}` - Pull request builds

## 📋 Available Images

### Main Application Images

| Image | Description | Size | Platforms |
|-------|-------------|------|-----------|
| `ghcr.io/harshaweb/queue:latest` | Production server | ~15MB | amd64, arm64 |
| `ghcr.io/harshaweb/queue:main` | Latest main branch | ~15MB | amd64, arm64 |

### Development Images

| Image | Description | Use Case |
|-------|-------------|----------|
| `redis-queue-local:latest` | Local development | Testing |

## 🔧 Build Configuration

### Dockerfile Options

1. **`Dockerfile`** - Full production build (Alpine-based)
2. **`Dockerfile.simple`** - Simplified build (Distroless-based)

### Build Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `GIT_COMMIT` | Git commit SHA | `unknown` |
| `BUILD_DATE` | Build timestamp | `unknown` |
| `VERSION` | Version/tag | `dev` |

### Environment Variables

The built images support all environment variables from `.env` file:

```bash
# Run with custom configuration
docker run -p 8080:8080 \
  -e REDIS_ADDRESSES=my-redis:6379 \
  -e JWT_SECRET=my-secure-secret-32-characters \
  -e LOG_LEVEL=debug \
  ghcr.io/harshaweb/queue:latest
```

## 🚀 Running Built Images

### Basic Usage

```bash
# Run server only
docker run -p 8080:8080 ghcr.io/harshaweb/queue:latest

# Run with Redis
docker run -d --name redis redis:7-alpine
docker run -p 8080:8080 \
  --link redis:redis \
  -e REDIS_ADDRESSES=redis:6379 \
  ghcr.io/harshaweb/queue:latest
```

### Production Deployment

```bash
# Using docker-compose
docker-compose -f docker-compose.build.yml up -d

# Using Kubernetes
kubectl apply -f deploy/k8s/
kubectl set image deployment/queue-server \
  queue-server=ghcr.io/harshaweb/queue:latest
```

### Development Testing

```bash
# Run with development settings
docker run -p 8080:8080 \
  -e ENVIRONMENT=development \
  -e DEBUG=true \
  -e LOG_LEVEL=debug \
  -v ${PWD}/.env.local:/app/.env.local \
  ghcr.io/harshaweb/queue:latest
```

## 🔍 Troubleshooting

### Build Issues

**Problem: Network connectivity errors**
```bash
# Check Docker daemon
docker info

# Check network connectivity
docker run --rm alpine:latest ping -c 1 google.com

# Try different base images
# Edit Dockerfile to use different registry mirrors
```

**Problem: Build context too large**
```bash
# Check .dockerignore file
cat .dockerignore

# Manually exclude large directories
echo "node_modules/" >> .dockerignore
echo "*.log" >> .dockerignore
```

### Registry Issues

**Problem: Authentication failed**
```bash
# Re-login to registry
docker login ghcr.io

# Check token permissions
# For GitHub: Ensure token has 'write:packages' scope
```

**Problem: Push denied**
```bash
# Check repository ownership
# Ensure you have push access to harshaweb/Queue

# Try with your own registry
docker tag ghcr.io/harshaweb/queue:latest ghcr.io/YOUR_USERNAME/queue:latest
docker push ghcr.io/YOUR_USERNAME/queue:latest
```

### Runtime Issues

**Problem: Container exits immediately**
```bash
# Check logs
docker logs CONTAINER_ID

# Run interactively for debugging
docker run -it --entrypoint /bin/sh ghcr.io/harshaweb/queue:latest

# Check health
docker run --rm ghcr.io/harshaweb/queue:latest --version
```

## 📊 Image Information

### Security Features

- ✅ **Distroless base** - Minimal attack surface
- ✅ **Non-root user** - Security best practices
- ✅ **Multi-stage build** - Smaller image size
- ✅ **No package manager** - Reduced vulnerabilities
- ✅ **Static compilation** - No runtime dependencies

### Performance Features

- ✅ **Static binaries** - Fast startup
- ✅ **Small size** (~15MB) - Quick downloads
- ✅ **Multi-arch** - Native performance on ARM/AMD64
- ✅ **Layer caching** - Faster rebuilds

## 🤖 Automation

### GitHub Actions

The repository automatically builds and pushes images on:

- ✅ Push to `main` branch
- ✅ Creation of version tags (`v*`)
- ✅ Pull requests (build only)

### Local Automation

```bash
# Set up local build automation
# Add to .git/hooks/pre-push:
#!/bin/bash
scripts/docker-build.sh $(git rev-parse --short HEAD)
```

## 📚 Related Documentation

- [Environment Configuration](./ENVIRONMENT.md) - Runtime configuration
- [Deployment Guide](../deploy/README.md) - Kubernetes deployment
- [Development Guide](./DEVELOPMENT.md) - Local development setup
- [Security Guide](./SECURITY.md) - Security considerations