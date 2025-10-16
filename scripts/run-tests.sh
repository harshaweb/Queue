#!/bin/bash
# Test runner script for the queue system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REDIS_PORT=6379
REDIS_HOST=localhost
TEST_TIMEOUT=10m
INTEGRATION_TIMEOUT=20m
BENCHMARK_TIMEOUT=30m
CHAOS_TIMEOUT=60m

echo -e "${BLUE}🚀 Queue System Test Runner${NC}"
echo "=================================="

# Function to check if Redis is running
check_redis() {
    echo -e "${YELLOW}🔍 Checking Redis connection...${NC}"
    if redis-cli -h $REDIS_HOST -p $REDIS_PORT ping > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis is running${NC}"
    else
        echo -e "${RED}❌ Redis is not running. Please start Redis first.${NC}"
        echo "Start Redis with: redis-server"
        exit 1
    fi
}

# Function to run unit tests
run_unit_tests() {
    echo -e "${YELLOW}🧪 Running unit tests...${NC}"
    go test -v -timeout $TEST_TIMEOUT ./pkg/... ./internal/... 2>&1 | tee test-results-unit.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✅ Unit tests passed${NC}"
    else
        echo -e "${RED}❌ Unit tests failed${NC}"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    echo -e "${YELLOW}🔗 Running integration tests...${NC}"
    go test -v -timeout $INTEGRATION_TIMEOUT ./test/integration/... 2>&1 | tee test-results-integration.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✅ Integration tests passed${NC}"
    else
        echo -e "${RED}❌ Integration tests failed${NC}"
        return 1
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    echo -e "${YELLOW}⚡ Running benchmark tests...${NC}"
    go test -v -timeout $BENCHMARK_TIMEOUT -bench=. -benchmem ./test/benchmark/... 2>&1 | tee test-results-benchmark.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✅ Benchmark tests completed${NC}"
    else
        echo -e "${RED}❌ Benchmark tests failed${NC}"
        return 1
    fi
}

# Function to run chaos tests
run_chaos_tests() {
    echo -e "${YELLOW}🌪️  Running chaos tests...${NC}"
    go test -v -timeout $CHAOS_TIMEOUT ./test/chaos/... 2>&1 | tee test-results-chaos.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✅ Chaos tests passed${NC}"
    else
        echo -e "${RED}❌ Chaos tests failed${NC}"
        return 1
    fi
}

# Function to run coverage analysis
run_coverage() {
    echo -e "${YELLOW}📊 Running coverage analysis...${NC}"
    go test -coverprofile=coverage.out ./pkg/... ./internal/...
    go tool cover -html=coverage.out -o coverage.html
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo -e "${GREEN}📈 Total coverage: $COVERAGE${NC}"
    echo "Coverage report saved to coverage.html"
}

# Function to run race detection
run_race_tests() {
    echo -e "${YELLOW}🏃 Running race condition tests...${NC}"
    go test -race -timeout $TEST_TIMEOUT ./pkg/... ./internal/... 2>&1 | tee test-results-race.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✅ Race tests passed${NC}"
    else
        echo -e "${RED}❌ Race conditions detected${NC}"
        return 1
    fi
}

# Function to lint code
run_lint() {
    echo -e "${YELLOW}🔍 Running code linting...${NC}"
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./... 2>&1 | tee lint-results.log
        if [ ${PIPESTATUS[0]} -eq 0 ]; then
            echo -e "${GREEN}✅ Linting passed${NC}"
        else
            echo -e "${RED}❌ Linting failed${NC}"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠️  golangci-lint not found, skipping linting${NC}"
    fi
}

# Function to clean up test artifacts
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up test artifacts...${NC}"
    rm -f coverage.out
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Function to generate test report
generate_report() {
    echo -e "${YELLOW}📋 Generating test report...${NC}"
    
    cat > test-report.md << EOF
# Queue System Test Report

Generated on: $(date)

## Test Results Summary

$(if [ -f test-results-unit.log ]; then
    echo "### Unit Tests"
    echo '```'
    tail -5 test-results-unit.log
    echo '```'
fi)

$(if [ -f test-results-integration.log ]; then
    echo "### Integration Tests"
    echo '```'
    tail -5 test-results-integration.log
    echo '```'
fi)

$(if [ -f test-results-benchmark.log ]; then
    echo "### Benchmark Results"
    echo '```'
    grep -E "(Benchmark|PASS|FAIL)" test-results-benchmark.log | tail -10
    echo '```'
fi)

$(if [ -f test-results-chaos.log ]; then
    echo "### Chaos Test Results"
    echo '```'
    tail -10 test-results-chaos.log
    echo '```'
fi)

## Coverage Report
$(if [ -f coverage.html ]; then
    echo "Coverage report available at: coverage.html"
fi)

## Files
- Unit test log: test-results-unit.log
- Integration test log: test-results-integration.log
- Benchmark log: test-results-benchmark.log
- Chaos test log: test-results-chaos.log
- Lint results: lint-results.log
- Race test log: test-results-race.log
EOF

    echo -e "${GREEN}📋 Test report generated: test-report.md${NC}"
}

# Main execution
main() {
    local test_type="$1"
    
    case "$test_type" in
        "unit")
            check_redis
            run_unit_tests
            ;;
        "integration")
            check_redis
            run_integration_tests
            ;;
        "benchmark")
            check_redis
            run_benchmark_tests
            ;;
        "chaos")
            check_redis
            run_chaos_tests
            ;;
        "coverage")
            check_redis
            run_coverage
            ;;
        "race")
            check_redis
            run_race_tests
            ;;
        "lint")
            run_lint
            ;;
        "all"|"")
            echo -e "${BLUE}🎯 Running all tests...${NC}"
            check_redis
            
            # Run tests in order
            run_lint && \
            run_unit_tests && \
            run_race_tests && \
            run_integration_tests && \
            run_benchmark_tests && \
            run_chaos_tests && \
            run_coverage
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
            else
                echo -e "${RED}💥 Some tests failed!${NC}"
                exit 1
            fi
            
            generate_report
            ;;
        "clean")
            cleanup
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [test_type]"
            echo ""
            echo "Test types:"
            echo "  unit        - Run unit tests"
            echo "  integration - Run integration tests"
            echo "  benchmark   - Run benchmark tests"
            echo "  chaos       - Run chaos tests"
            echo "  coverage    - Run coverage analysis"
            echo "  race        - Run race condition tests"
            echo "  lint        - Run code linting"
            echo "  all         - Run all tests (default)"
            echo "  clean       - Clean up test artifacts"
            echo "  help        - Show this help"
            echo ""
            echo "Examples:"
            echo "  $0           # Run all tests"
            echo "  $0 unit      # Run only unit tests"
            echo "  $0 benchmark # Run only benchmarks"
            ;;
        *)
            echo -e "${RED}❌ Unknown test type: $test_type${NC}"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"