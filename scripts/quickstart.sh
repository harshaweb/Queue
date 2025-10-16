#!/bin/bash

# Redis Queue System Quick Start Script
# This script helps you get started quickly with the Redis Queue System

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for service
wait_for_service() {
    local service_name=$1
    local url=$2
    local max_attempts=$3
    local attempt=1
    
    print_status "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$url" >/dev/null 2>&1; then
            print_success "$service_name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start within expected time"
    return 1
}

print_header() {
    echo "=================================="
    echo "Redis Queue System Quick Start"
    echo "=================================="
    echo
}

check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command_exists docker-compose; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    print_success "All prerequisites are met!"
}

start_development() {
    print_status "Starting development environment..."
    
    # Build the application
    print_status "Building application..."
    docker-compose build
    
    # Start services
    print_status "Starting services..."
    docker-compose up -d
    
    # Wait for services to be ready
    wait_for_service "Redis" "http://localhost:6379" 30 || exit 1
    wait_for_service "Queue Server" "http://localhost:8080/health" 60 || exit 1
    wait_for_service "Prometheus" "http://localhost:9091" 30 || exit 1
    wait_for_service "Grafana" "http://localhost:3000" 30 || exit 1
    
    print_success "Development environment is ready!"
    echo
    echo "Available services:"
    echo "  - Queue API (HTTP):    http://localhost:8080"
    echo "  - Queue API (gRPC):    localhost:9090"
    echo "  - Prometheus:          http://localhost:9091"
    echo "  - Grafana:             http://localhost:3000 (admin/admin)"
    echo "  - Redis Insight:       http://localhost:8001"
    echo
    echo "To view logs: docker-compose logs -f"
    echo "To stop:      docker-compose down"
}

start_production() {
    print_status "Starting production environment..."
    
    # Check for required environment variables
    if [ -z "$JWT_SECRET" ]; then
        print_error "JWT_SECRET environment variable is required for production"
        exit 1
    fi
    
    if [ -z "$API_KEYS" ]; then
        print_error "API_KEYS environment variable is required for production"
        exit 1
    fi
    
    if [ -z "$GRAFANA_PASSWORD" ]; then
        print_error "GRAFANA_PASSWORD environment variable is required for production"
        exit 1
    fi
    
    # Build the application
    print_status "Building application..."
    docker-compose -f docker-compose.yml -f docker-compose.prod.yml build
    
    # Start services
    print_status "Starting services..."
    docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
    
    # Wait for services to be ready
    wait_for_service "Redis" "http://localhost:6379" 30 || exit 1
    wait_for_service "Queue Server" "http://localhost:8080/health" 60 || exit 1
    wait_for_service "Prometheus" "http://localhost:9091" 30 || exit 1
    wait_for_service "Grafana" "http://localhost:3000" 30 || exit 1
    
    print_success "Production environment is ready!"
}

run_tests() {
    print_status "Running tests..."
    
    # Start test dependencies
    docker-compose up -d redis
    wait_for_service "Redis" "http://localhost:6379" 30 || exit 1
    
    # Run tests
    print_status "Running unit tests..."
    go test ./... -v
    
    print_status "Running integration tests..."
    go test ./tests/integration/... -v
    
    print_status "Running benchmarks..."
    go test ./... -bench=. -benchmem
    
    print_success "All tests passed!"
}

show_help() {
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  dev         Start development environment"
    echo "  prod        Start production environment"
    echo "  test        Run tests"
    echo "  stop        Stop all services"
    echo "  clean       Stop and remove all containers and volumes"
    echo "  logs        Show logs"
    echo "  help        Show this help message"
    echo
    echo "Examples:"
    echo "  $0 dev                    # Start development environment"
    echo "  $0 prod                   # Start production environment"
    echo "  $0 test                   # Run all tests"
    echo "  $0 logs queue-server      # Show logs for specific service"
}

stop_services() {
    print_status "Stopping services..."
    docker-compose down
    print_success "Services stopped!"
}

clean_all() {
    print_warning "This will remove all containers, volumes, and data!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleaning up..."
        docker-compose down -v --remove-orphans
        docker system prune -f
        print_success "Cleanup complete!"
    else
        print_status "Cleanup cancelled."
    fi
}

show_logs() {
    local service=$1
    if [ -n "$service" ]; then
        docker-compose logs -f "$service"
    else
        docker-compose logs -f
    fi
}

# Main script logic
main() {
    print_header
    
    case "${1:-help}" in
        "dev"|"development")
            check_prerequisites
            start_development
            ;;
        "prod"|"production")
            check_prerequisites
            start_production
            ;;
        "test"|"tests")
            check_prerequisites
            run_tests
            ;;
        "stop")
            stop_services
            ;;
        "clean")
            clean_all
            ;;
        "logs")
            show_logs "$2"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"