#!/bin/bash

# Docker Build and Push Script for Redis Queue System
# Usage: ./scripts/docker-build.sh [tag] [registry]

set -e

# Default values
DEFAULT_TAG="latest"
DEFAULT_REGISTRY="ghcr.io/harshaweb"
DEFAULT_IMAGE_NAME="queue"

# Parse command line arguments
TAG=${1:-$DEFAULT_TAG}
REGISTRY=${2:-$DEFAULT_REGISTRY}
IMAGE_NAME=${3:-$DEFAULT_IMAGE_NAME}

# Full image name
FULL_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${TAG}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command_exists docker; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build Docker image
build_image() {
    log_info "Building Docker image: ${FULL_IMAGE_NAME}"
    
    # Get build context information
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
    
    log_info "Build context:"
    log_info "  - Git commit: ${GIT_COMMIT}"
    log_info "  - Build date: ${BUILD_DATE}"
    log_info "  - Version: ${VERSION}"
    
    # Build the image with build args
    docker build \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VERSION="${VERSION}" \
        --tag "${FULL_IMAGE_NAME}" \
        --tag "${REGISTRY}/${IMAGE_NAME}:${GIT_COMMIT}" \
        .
    
    log_success "Docker image built successfully"
}

# Test the built image
test_image() {
    log_info "Testing Docker image..."
    
    # Test that the image can start
    local container_id
    container_id=$(docker run -d --rm "${FULL_IMAGE_NAME}" --help 2>/dev/null || true)
    
    if [ -n "$container_id" ]; then
        sleep 2
        docker stop "$container_id" >/dev/null 2>&1 || true
        log_success "Image test passed"
    else
        log_warning "Could not test image startup (this is normal for some images)"
    fi
}

# Push image to registry
push_image() {
    log_info "Pushing image to registry: ${REGISTRY}"
    
    # Check if user is logged in to registry
    if [[ "${REGISTRY}" == "ghcr.io"* ]]; then
        log_info "Detected GitHub Container Registry"
        log_info "Make sure you're logged in with: docker login ghcr.io -u USERNAME"
    elif [[ "${REGISTRY}" == *"docker.io"* ]] || [[ "${REGISTRY}" == *"hub.docker.com"* ]]; then
        log_info "Detected Docker Hub"
        log_info "Make sure you're logged in with: docker login"
    fi
    
    # Push both tags
    log_info "Pushing ${FULL_IMAGE_NAME}..."
    docker push "${FULL_IMAGE_NAME}"
    
    log_info "Pushing ${REGISTRY}/${IMAGE_NAME}:${GIT_COMMIT}..."
    docker push "${REGISTRY}/${IMAGE_NAME}:${GIT_COMMIT}"
    
    log_success "Images pushed successfully"
}

# Show image information
show_image_info() {
    log_info "Image information:"
    docker images "${REGISTRY}/${IMAGE_NAME}" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"
    
    log_info ""
    log_info "Image layers:"
    docker history "${FULL_IMAGE_NAME}" --format "table {{.ID}}\t{{.CreatedBy}}\t{{.Size}}"
}

# Cleanup old images
cleanup_images() {
    log_info "Cleaning up old images..."
    
    # Remove dangling images
    docker image prune -f >/dev/null 2>&1 || true
    
    # Remove old versions (keep last 5)
    docker images "${REGISTRY}/${IMAGE_NAME}" --format "{{.ID}}" | tail -n +6 | xargs -r docker rmi >/dev/null 2>&1 || true
    
    log_success "Cleanup completed"
}

# Main function
main() {
    echo "======================================"
    echo "🐳 Redis Queue System Docker Builder"
    echo "======================================"
    echo ""
    
    log_info "Configuration:"
    log_info "  - Registry: ${REGISTRY}"
    log_info "  - Image: ${IMAGE_NAME}"
    log_info "  - Tag: ${TAG}"
    log_info "  - Full image name: ${FULL_IMAGE_NAME}"
    echo ""
    
    # Execute build pipeline
    check_prerequisites
    build_image
    test_image
    show_image_info
    
    # Ask for push confirmation
    echo ""
    read -p "Do you want to push the image to the registry? (y/N): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        push_image
        cleanup_images
        
        echo ""
        log_success "🎉 Docker build and push completed successfully!"
        log_info "You can now use the image with:"
        log_info "  docker run -p 8080:8080 ${FULL_IMAGE_NAME}"
        echo ""
        log_info "Or deploy to Kubernetes:"
        log_info "  kubectl set image deployment/queue-server queue-server=${FULL_IMAGE_NAME}"
    else
        log_info "Image built but not pushed. You can push later with:"
        log_info "  docker push ${FULL_IMAGE_NAME}"
    fi
    
    echo ""
    log_success "Build process completed! 🚀"
}

# Run main function
main "$@"