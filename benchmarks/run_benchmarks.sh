#!/bin/bash

# Comprehensive benchmark runner for Apprise-Go
# This script runs various performance tests and generates detailed reports

set -e

# Configuration
RESULTS_DIR="$(dirname "$0")/results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_DIR="$RESULTS_DIR/$TIMESTAMP"
LOG_FILE="$REPORT_DIR/benchmark.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run comprehensive benchmarks for Apprise-Go

OPTIONS:
    -t, --type TYPE        Benchmark type: all, core, load, api (default: all)
    -o, --output DIR       Output directory (default: results/TIMESTAMP)
    -c, --config FILE      Custom load test config file
    -s, --server URL       API server URL (default: http://localhost:8080)
    -d, --duration TIME    Override test duration (e.g., 1m, 30s)
    -p, --parallel N       Parallel benchmark processes (default: 4)
    -r, --repeat N         Number of benchmark repetitions (default: 3)
    -q, --quiet           Quiet mode - minimal output
    -h, --help            Show this help message

EXAMPLES:
    $0                     # Run all benchmarks with defaults
    $0 -t core            # Run only core library benchmarks  
    $0 -t load -d 2m      # Run load tests for 2 minutes
    $0 -c custom.json     # Use custom load test configuration
    $0 -s http://prod:8080 -t api  # Test against production API

EOF
}

# Default configuration
BENCH_TYPE="all"
OUTPUT_DIR=""
CONFIG_FILE=""
SERVER_URL="http://localhost:8080"
DURATION=""
PARALLEL=4
REPEAT=3
QUIET=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            BENCH_TYPE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -s|--server)
            SERVER_URL="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -p|--parallel)
            PARALLEL="$2"
            shift 2
            ;;
        -r|--repeat)
            REPEAT="$2"
            shift 2
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Set output directory
if [[ -n "$OUTPUT_DIR" ]]; then
    REPORT_DIR="$OUTPUT_DIR"
fi

# Create results directory
mkdir -p "$REPORT_DIR"
touch "$LOG_FILE"

log_info "Starting Apprise-Go benchmarks"
log_info "Configuration:"
log_info "  Type: $BENCH_TYPE"
log_info "  Output: $REPORT_DIR"
log_info "  Server: $SERVER_URL"
log_info "  Parallel: $PARALLEL"
log_info "  Repeat: $REPEAT"

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Check if server is running (for API tests)
    if [[ "$BENCH_TYPE" == "all" || "$BENCH_TYPE" == "api" || "$BENCH_TYPE" == "load" ]]; then
        if ! curl -sf "$SERVER_URL/health" &> /dev/null; then
            log_warning "API server at $SERVER_URL is not responding"
            log_warning "API and load tests will be skipped"
            if [[ "$BENCH_TYPE" == "api" || "$BENCH_TYPE" == "load" ]]; then
                log_error "Cannot run $BENCH_TYPE tests without server"
                exit 1
            fi
        else
            log_success "API server is responding"
        fi
    fi
    
    # Check system resources
    available_memory=$(free -m 2>/dev/null | awk '/^Mem:/{print $7}' || echo "unknown")
    if [[ "$available_memory" != "unknown" && $available_memory -lt 1000 ]]; then
        log_warning "Low available memory: ${available_memory}MB"
    fi
}

# Run core library benchmarks
run_core_benchmarks() {
    log_info "Running core library benchmarks..."
    
    cd "$(dirname "$0")"
    
    local bench_file="$REPORT_DIR/core_benchmarks.txt"
    local json_file="$REPORT_DIR/core_benchmarks.json"
    
    # Run benchmarks multiple times for reliability
    for i in $(seq 1 $REPEAT); do
        log_info "Core benchmarks run $i/$REPEAT"
        
        go test -bench=. -benchmem -count=1 -timeout=30m \
            -benchtime=5s -cpu=$PARALLEL \
            ./benchmarks/ >> "$bench_file" 2>&1
    done
    
    # Process results
    log_info "Processing core benchmark results..."
    process_core_results "$bench_file" "$json_file"
    
    log_success "Core benchmarks completed"
}

# Run load tests
run_load_tests() {
    log_info "Running load tests..."
    
    cd "$(dirname "$0")/load-tests"
    
    # Build load test binary
    go build -o load-tester . || {
        log_error "Failed to build load tester"
        return 1
    }
    
    # Define test configurations
    local configs=("quick" "moderate" "stress")
    
    if [[ -n "$CONFIG_FILE" ]]; then
        configs=("custom")
    fi
    
    for config in "${configs[@]}"; do
        log_info "Running load test: $config"
        
        local output_file="$REPORT_DIR/load_test_${config}.json"
        local extra_args=""
        
        if [[ "$config" == "custom" ]]; then
            extra_args="-config-file $CONFIG_FILE"
        else
            extra_args="-config $config"
        fi
        
        if [[ -n "$DURATION" ]]; then
            extra_args="$extra_args -duration $DURATION"
        fi
        
        if [[ -n "$SERVER_URL" ]]; then
            extra_args="$extra_args -url $SERVER_URL"
        fi
        
        # Run load test
        ./load-tester $extra_args -output "$output_file" || {
            log_warning "Load test $config failed"
            continue
        }
        
        log_success "Load test $config completed"
    done
    
    # Clean up
    rm -f load-tester
}

# Run API-specific benchmarks
run_api_benchmarks() {
    log_info "Running API benchmarks..."
    
    local output_file="$REPORT_DIR/api_benchmarks.txt"
    
    # Test various API endpoints
    local endpoints=(
        "/health"
        "/version" 
        "/metrics"
        "/api/v1/notify"
        "/api/v1/services"
    )
    
    for endpoint in "${endpoints[@]}"; do
        log_info "Benchmarking endpoint: $endpoint"
        
        # Use Apache Bench if available, otherwise use curl
        if command -v ab &> /dev/null; then
            ab -n 1000 -c 10 "$SERVER_URL$endpoint" >> "$output_file" 2>&1 || true
        else
            log_warning "Apache Bench not available, using basic curl test"
            for i in {1..100}; do
                curl -s "$SERVER_URL$endpoint" > /dev/null || true
            done
        fi
    done
    
    log_success "API benchmarks completed"
}

# Process core benchmark results into JSON format
process_core_results() {
    local input_file="$1"
    local output_file="$2"
    
    # Extract benchmark results using awk
    awk '
    BEGIN { 
        print "{"
        print "  \"benchmarks\": ["
        first = 1
    }
    /^Benchmark/ && /ns\/op/ {
        if (!first) print ","
        first = 0
        
        # Parse benchmark line
        name = $1
        iterations = $2
        ns_per_op = $3
        
        # Extract additional metrics if present
        bytes_per_op = ""
        allocs_per_op = ""
        mb_per_sec = ""
        
        for (i = 4; i <= NF; i++) {
            if ($i ~ /B\/op$/) bytes_per_op = $(i-1)
            if ($i ~ /allocs\/op$/) allocs_per_op = $(i-1)
            if ($i ~ /MB\/s$/) mb_per_sec = $(i-1)
        }
        
        printf "    {\n"
        printf "      \"name\": \"%s\",\n", name
        printf "      \"iterations\": %s,\n", iterations
        printf "      \"ns_per_op\": %s", ns_per_op
        
        if (bytes_per_op != "") printf ",\n      \"bytes_per_op\": %s", bytes_per_op
        if (allocs_per_op != "") printf ",\n      \"allocs_per_op\": %s", allocs_per_op
        if (mb_per_sec != "") printf ",\n      \"mb_per_sec\": %s", mb_per_sec
        
        printf "\n    }"
    }
    END {
        print ""
        print "  ],"
        printf "  \"timestamp\": \"%s\",\n", strftime("%Y-%m-%dT%H:%M:%SZ")
        printf "  \"version\": \"benchmark-run\"\n"
        print "}"
    }
    ' "$input_file" > "$output_file"
}

# Generate comprehensive report
generate_report() {
    log_info "Generating comprehensive report..."
    
    local report_file="$REPORT_DIR/benchmark_report.md"
    
    cat > "$report_file" << EOF
# Apprise-Go Benchmark Report

**Generated:** $(date)  
**Duration:** $DURATION  
**Server:** $SERVER_URL  
**Type:** $BENCH_TYPE

## Test Environment

- **OS:** $(uname -s -r)
- **Go Version:** $(go version)
- **CPU:** $(nproc) cores
- **Memory:** $(free -h 2>/dev/null | awk '/^Mem:/{print $2}' || echo "unknown")

## Summary

EOF

    # Add core benchmarks summary if available
    if [[ -f "$REPORT_DIR/core_benchmarks.json" ]]; then
        cat >> "$report_file" << EOF
### Core Library Benchmarks

$(jq -r '.benchmarks[] | "- **\(.name)**: \(.ns_per_op) ns/op (\(.iterations) iterations)"' "$REPORT_DIR/core_benchmarks.json" | head -10)

EOF
    fi

    # Add load test summaries if available
    for load_result in "$REPORT_DIR"/load_test_*.json; do
        if [[ -f "$load_result" ]]; then
            local config_name=$(basename "$load_result" .json | sed 's/load_test_//')
            cat >> "$report_file" << EOF
### Load Test: $config_name

$(jq -r '"- **Total Requests**: \(.total_requests)
- **Success Rate**: \((1 - .error_rate) * 100 | round)%  
- **Average Response Time**: \(.average_response_time)
- **95th Percentile**: \(.p95_response_time)
- **Requests/Second**: \(.requests_per_second | round)"' "$load_result")

EOF
        fi
    done

    cat >> "$report_file" << EOF
## Detailed Results

- Core benchmarks: [core_benchmarks.txt](core_benchmarks.txt)
- Load test results: load_test_*.json
- API benchmarks: [api_benchmarks.txt](api_benchmarks.txt)

## Performance Analysis

### Key Metrics

1. **Notification Throughput**: Measured in notifications/second
2. **Response Time**: 95th percentile response times under load
3. **Resource Efficiency**: Memory and CPU usage patterns
4. **Error Rates**: Reliability under various load conditions

### Recommendations

Based on the benchmark results:

1. **Optimal Concurrency**: Best performance observed at X concurrent connections
2. **Scaling Limits**: System shows degradation beyond Y requests/second  
3. **Memory Usage**: Peak memory usage of Z MB under maximum load
4. **Bottlenecks**: Identified performance bottlenecks in [component]

## Files Generated

$(ls -la "$REPORT_DIR"/ | tail -n +2)

---
*Report generated by Apprise-Go benchmark suite*
EOF

    log_success "Report generated: $report_file"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    # Remove any temporary files or processes
}

# Main execution
main() {
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Check prerequisites
    check_prerequisites
    
    # Run benchmarks based on type
    case "$BENCH_TYPE" in
        "all")
            run_core_benchmarks
            run_load_tests
            run_api_benchmarks
            ;;
        "core")
            run_core_benchmarks
            ;;
        "load")
            run_load_tests
            ;;
        "api")
            run_api_benchmarks
            ;;
        *)
            log_error "Unknown benchmark type: $BENCH_TYPE"
            exit 1
            ;;
    esac
    
    # Generate comprehensive report
    generate_report
    
    log_success "All benchmarks completed successfully!"
    log_info "Results available in: $REPORT_DIR"
    
    # Show quick summary
    if [[ "$QUIET" == "false" ]]; then
        echo
        echo "=== Quick Summary ==="
        if [[ -f "$REPORT_DIR/benchmark_report.md" ]]; then
            head -30 "$REPORT_DIR/benchmark_report.md"
        fi
    fi
}

# Run main function
main "$@"