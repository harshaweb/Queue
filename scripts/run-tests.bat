@echo off
setlocal enabledelayedexpansion

:: Queue System Test Runner for Windows
echo ===============================
echo 🚀 Queue System Test Runner
echo ===============================

:: Configuration
set REDIS_PORT=6379
set REDIS_HOST=localhost
set TEST_TIMEOUT=10m
set INTEGRATION_TIMEOUT=20m
set BENCHMARK_TIMEOUT=30m
set CHAOS_TIMEOUT=60m

:: Check command line arguments
set TEST_TYPE=%1
if "%TEST_TYPE%"=="" set TEST_TYPE=all

goto main

:check_redis
    echo 🔍 Checking Redis connection...
    redis-cli -h %REDIS_HOST% -p %REDIS_PORT% ping >nul 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Redis is running
    ) else (
        echo ❌ Redis is not running. Please start Redis first.
        echo Start Redis with: redis-server
        exit /b 1
    )
goto :eof

:run_unit_tests
    echo 🧪 Running unit tests...
    go test -v -timeout %TEST_TIMEOUT% ./pkg/... ./internal/... > test-results-unit.log 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Unit tests passed
    ) else (
        echo ❌ Unit tests failed
        exit /b 1
    )
goto :eof

:run_integration_tests
    echo 🔗 Running integration tests...
    go test -v -timeout %INTEGRATION_TIMEOUT% ./test/integration/... > test-results-integration.log 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Integration tests passed
    ) else (
        echo ❌ Integration tests failed
        exit /b 1
    )
goto :eof

:run_benchmark_tests
    echo ⚡ Running benchmark tests...
    go test -v -timeout %BENCHMARK_TIMEOUT% -bench=. -benchmem ./test/benchmark/... > test-results-benchmark.log 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Benchmark tests completed
    ) else (
        echo ❌ Benchmark tests failed
        exit /b 1
    )
goto :eof

:run_chaos_tests
    echo 🌪️ Running chaos tests...
    go test -v -timeout %CHAOS_TIMEOUT% ./test/chaos/... > test-results-chaos.log 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Chaos tests passed
    ) else (
        echo ❌ Chaos tests failed
        exit /b 1
    )
goto :eof

:run_coverage
    echo 📊 Running coverage analysis...
    go test -coverprofile=coverage.out ./pkg/... ./internal/...
    go tool cover -html=coverage.out -o coverage.html
    
    :: Extract coverage percentage (simplified for Windows)
    for /f "tokens=3" %%i in ('go tool cover -func=coverage.out ^| findstr "total"') do set COVERAGE=%%i
    echo 📈 Total coverage: !COVERAGE!
    echo Coverage report saved to coverage.html
goto :eof

:run_race_tests
    echo 🏃 Running race condition tests...
    go test -race -timeout %TEST_TIMEOUT% ./pkg/... ./internal/... > test-results-race.log 2>&1
    if !errorlevel! equ 0 (
        echo ✅ Race tests passed
    ) else (
        echo ❌ Race conditions detected
        exit /b 1
    )
goto :eof

:run_lint
    echo 🔍 Running code linting...
    where golangci-lint >nul 2>&1
    if !errorlevel! equ 0 (
        golangci-lint run ./... > lint-results.log 2>&1
        if !errorlevel! equ 0 (
            echo ✅ Linting passed
        ) else (
            echo ❌ Linting failed
            exit /b 1
        )
    ) else (
        echo ⚠️ golangci-lint not found, skipping linting
    )
goto :eof

:cleanup
    echo 🧹 Cleaning up test artifacts...
    if exist coverage.out del coverage.out
    if exist *.log del *.log
    echo ✅ Cleanup completed
goto :eof

:generate_report
    echo 📋 Generating test report...
    
    echo # Queue System Test Report > test-report.md
    echo. >> test-report.md
    echo Generated on: %date% %time% >> test-report.md
    echo. >> test-report.md
    echo ## Test Results Summary >> test-report.md
    echo. >> test-report.md
    
    if exist test-results-unit.log (
        echo ### Unit Tests >> test-report.md
        echo ``` >> test-report.md
        powershell "Get-Content test-results-unit.log | Select-Object -Last 5" >> test-report.md
        echo ``` >> test-report.md
        echo. >> test-report.md
    )
    
    if exist test-results-integration.log (
        echo ### Integration Tests >> test-report.md
        echo ``` >> test-report.md
        powershell "Get-Content test-results-integration.log | Select-Object -Last 5" >> test-report.md
        echo ``` >> test-report.md
        echo. >> test-report.md
    )
    
    if exist test-results-benchmark.log (
        echo ### Benchmark Results >> test-report.md
        echo ``` >> test-report.md
        findstr /R "Benchmark.*PASS.*FAIL" test-results-benchmark.log >> test-report.md
        echo ``` >> test-report.md
        echo. >> test-report.md
    )
    
    if exist coverage.html (
        echo ## Coverage Report >> test-report.md
        echo Coverage report available at: coverage.html >> test-report.md
        echo. >> test-report.md
    )
    
    echo ## Files >> test-report.md
    echo - Unit test log: test-results-unit.log >> test-report.md
    echo - Integration test log: test-results-integration.log >> test-report.md
    echo - Benchmark log: test-results-benchmark.log >> test-report.md
    echo - Chaos test log: test-results-chaos.log >> test-report.md
    echo - Lint results: lint-results.log >> test-report.md
    echo - Race test log: test-results-race.log >> test-report.md
    
    echo 📋 Test report generated: test-report.md
goto :eof

:show_help
    echo Usage: %0 [test_type]
    echo.
    echo Test types:
    echo   unit        - Run unit tests
    echo   integration - Run integration tests
    echo   benchmark   - Run benchmark tests
    echo   chaos       - Run chaos tests
    echo   coverage    - Run coverage analysis
    echo   race        - Run race condition tests
    echo   lint        - Run code linting
    echo   all         - Run all tests (default)
    echo   clean       - Clean up test artifacts
    echo   help        - Show this help
    echo.
    echo Examples:
    echo   %0           # Run all tests
    echo   %0 unit      # Run only unit tests
    echo   %0 benchmark # Run only benchmarks
goto :eof

:main
    if "%TEST_TYPE%"=="unit" (
        call :check_redis
        call :run_unit_tests
    ) else if "%TEST_TYPE%"=="integration" (
        call :check_redis
        call :run_integration_tests
    ) else if "%TEST_TYPE%"=="benchmark" (
        call :check_redis
        call :run_benchmark_tests
    ) else if "%TEST_TYPE%"=="chaos" (
        call :check_redis
        call :run_chaos_tests
    ) else if "%TEST_TYPE%"=="coverage" (
        call :check_redis
        call :run_coverage
    ) else if "%TEST_TYPE%"=="race" (
        call :check_redis
        call :run_race_tests
    ) else if "%TEST_TYPE%"=="lint" (
        call :run_lint
    ) else if "%TEST_TYPE%"=="all" (
        echo 🎯 Running all tests...
        call :check_redis
        call :run_lint
        if !errorlevel! neq 0 goto :error
        call :run_unit_tests
        if !errorlevel! neq 0 goto :error
        call :run_race_tests
        if !errorlevel! neq 0 goto :error
        call :run_integration_tests
        if !errorlevel! neq 0 goto :error
        call :run_benchmark_tests
        if !errorlevel! neq 0 goto :error
        call :run_chaos_tests
        if !errorlevel! neq 0 goto :error
        call :run_coverage
        
        echo 🎉 All tests completed successfully!
        call :generate_report
    ) else if "%TEST_TYPE%"=="clean" (
        call :cleanup
    ) else if "%TEST_TYPE%"=="help" (
        call :show_help
    ) else if "%TEST_TYPE%"=="-h" (
        call :show_help
    ) else if "%TEST_TYPE%"=="--help" (
        call :show_help
    ) else (
        echo ❌ Unknown test type: %TEST_TYPE%
        echo Use '%0 help' for usage information
        exit /b 1
    )
goto :eof

:error
    echo 💥 Some tests failed!
    exit /b 1