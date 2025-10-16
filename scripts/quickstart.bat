@echo off
REM Redis Queue System Quick Start Script for Windows
REM This script helps you get started quickly with the Redis Queue System

setlocal enabledelayedexpansion

echo ==================================
echo Redis Queue System Quick Start
echo ==================================
echo.

REM Check if Docker is installed
docker --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not installed. Please install Docker first.
    exit /b 1
)

REM Check if Docker Compose is installed
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker Compose is not installed. Please install Docker Compose first.
    exit /b 1
)

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not running. Please start Docker first.
    exit /b 1
)

echo [INFO] All prerequisites are met!

REM Parse command line arguments
set COMMAND=%1
if "%COMMAND%"=="" set COMMAND=help

if "%COMMAND%"=="dev" goto :start_dev
if "%COMMAND%"=="development" goto :start_dev
if "%COMMAND%"=="prod" goto :start_prod
if "%COMMAND%"=="production" goto :start_prod
if "%COMMAND%"=="test" goto :run_tests
if "%COMMAND%"=="tests" goto :run_tests
if "%COMMAND%"=="stop" goto :stop_services
if "%COMMAND%"=="clean" goto :clean_all
if "%COMMAND%"=="logs" goto :show_logs
if "%COMMAND%"=="help" goto :show_help
if "%COMMAND%"=="-h" goto :show_help
if "%COMMAND%"=="--help" goto :show_help

echo [ERROR] Unknown command: %COMMAND%
echo.
goto :show_help

:start_dev
echo [INFO] Starting development environment...

echo [INFO] Building application...
docker-compose build
if errorlevel 1 (
    echo [ERROR] Failed to build application
    exit /b 1
)

echo [INFO] Starting services...
docker-compose up -d
if errorlevel 1 (
    echo [ERROR] Failed to start services
    exit /b 1
)

echo [INFO] Waiting for services to be ready...
timeout /t 10 /nobreak >nul

echo [SUCCESS] Development environment is ready!
echo.
echo Available services:
echo   - Queue API (HTTP):    http://localhost:8080
echo   - Queue API (gRPC):    localhost:9090
echo   - Prometheus:          http://localhost:9091
echo   - Grafana:             http://localhost:3000 (admin/admin)
echo   - Redis Insight:       http://localhost:8001
echo.
echo To view logs: docker-compose logs -f
echo To stop:      docker-compose down
goto :end

:start_prod
echo [INFO] Starting production environment...

REM Check for required environment variables
if "%JWT_SECRET%"=="" (
    echo [ERROR] JWT_SECRET environment variable is required for production
    exit /b 1
)

if "%API_KEYS%"=="" (
    echo [ERROR] API_KEYS environment variable is required for production
    exit /b 1
)

if "%GRAFANA_PASSWORD%"=="" (
    echo [ERROR] GRAFANA_PASSWORD environment variable is required for production
    exit /b 1
)

echo [INFO] Building application...
docker-compose -f docker-compose.yml -f docker-compose.prod.yml build
if errorlevel 1 (
    echo [ERROR] Failed to build application
    exit /b 1
)

echo [INFO] Starting services...
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
if errorlevel 1 (
    echo [ERROR] Failed to start services
    exit /b 1
)

echo [INFO] Waiting for services to be ready...
timeout /t 10 /nobreak >nul

echo [SUCCESS] Production environment is ready!
goto :end

:run_tests
echo [INFO] Running tests...

echo [INFO] Starting test dependencies...
docker-compose up -d redis
timeout /t 5 /nobreak >nul

echo [INFO] Running unit tests...
go test ./... -v
if errorlevel 1 (
    echo [ERROR] Unit tests failed
    exit /b 1
)

echo [INFO] Running integration tests...
go test ./tests/integration/... -v
if errorlevel 1 (
    echo [ERROR] Integration tests failed
    exit /b 1
)

echo [INFO] Running benchmarks...
go test ./... -bench=. -benchmem

echo [SUCCESS] All tests passed!
goto :end

:stop_services
echo [INFO] Stopping services...
docker-compose down
echo [SUCCESS] Services stopped!
goto :end

:clean_all
echo [WARNING] This will remove all containers, volumes, and data!
set /p CONFIRM="Are you sure? (y/N): "
if /i not "%CONFIRM%"=="y" (
    echo [INFO] Cleanup cancelled.
    goto :end
)

echo [INFO] Cleaning up...
docker-compose down -v --remove-orphans
docker system prune -f
echo [SUCCESS] Cleanup complete!
goto :end

:show_logs
if "%2"=="" (
    docker-compose logs -f
) else (
    docker-compose logs -f %2
)
goto :end

:show_help
echo Usage: %0 [COMMAND]
echo.
echo Commands:
echo   dev         Start development environment
echo   prod        Start production environment
echo   test        Run tests
echo   stop        Stop all services
echo   clean       Stop and remove all containers and volumes
echo   logs        Show logs
echo   help        Show this help message
echo.
echo Examples:
echo   %0 dev                    # Start development environment
echo   %0 prod                   # Start production environment
echo   %0 test                   # Run all tests
echo   %0 logs queue-server      # Show logs for specific service
goto :end

:end
endlocal