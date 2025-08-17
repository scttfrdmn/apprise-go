#!/bin/bash

# Apprise-Go Performance Benchmarking Suite
# Comprehensive performance analysis and reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
BENCHMARK_OUTPUT_DIR="benchmarks/results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_FILE="${BENCHMARK_OUTPUT_DIR}/benchmark_${TIMESTAMP}.txt"
CPU_PROFILE_FILE="${BENCHMARK_OUTPUT_DIR}/cpu_${TIMESTAMP}.prof"
MEM_PROFILE_FILE="${BENCHMARK_OUTPUT_DIR}/mem_${TIMESTAMP}.prof"

# Ensure output directory exists
mkdir -p "$BENCHMARK_OUTPUT_DIR"

echo -e "${BLUE}===========================================${NC}"
echo -e "${BLUE}    Apprise-Go Performance Benchmark Suite${NC}"
echo -e "${BLUE}===========================================${NC}"
echo ""

# System information
echo -e "${CYAN}System Information:${NC}"
echo "Date: $(date)"
echo "Go Version: $(go version)"
echo "OS: $(uname -s -r)"
echo "CPU: $(sysctl -n machdep.cpu.brand_string 2>/dev/null || cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2 | xargs)"
echo "CPU Cores: $(sysctl -n hw.ncpu 2>/dev/null || nproc)"
echo "Memory: $(sysctl -n hw.memsize 2>/dev/null | awk '{print int($1/1024/1024/1024) "GB"}' || free -h | awk '/^Mem:/ {print $2}')"
echo ""

# Function to run benchmark with options
run_benchmark() {
    local name=$1
    local pattern=$2
    local options=$3
    
    echo -e "${YELLOW}Running: $name${NC}"
    echo "go test -bench=$pattern $options -benchmem -v"
    
    if ! go test -bench="$pattern" $options -benchmem -v | tee -a "$RESULTS_FILE"; then
        echo -e "${RED}Benchmark failed: $name${NC}"
        return 1
    fi
    
    echo ""
    return 0
}

# Function to run with profiling
run_with_profiling() {
    local name=$1
    local pattern=$2
    
    echo -e "${PURPLE}Running with CPU profiling: $name${NC}"
    if ! go test -bench="$pattern" -benchmem -cpuprofile="${CPU_PROFILE_FILE}" -v | tee -a "$RESULTS_FILE"; then
        echo -e "${RED}CPU profiling benchmark failed: $name${NC}"
        return 1
    fi
    
    echo -e "${PURPLE}Running with memory profiling: $name${NC}"
    if ! go test -bench="$pattern" -benchmem -memprofile="${MEM_PROFILE_FILE}" -v | tee -a "$RESULTS_FILE"; then
        echo -e "${RED}Memory profiling benchmark failed: $name${NC}"
        return 1
    fi
    
    return 0
}

# Start benchmarking
echo -e "${GREEN}Starting benchmark suite...${NC}"
echo ""

# Save system info to results file
{
    echo "========================================"
    echo "Apprise-Go Performance Benchmark Results"
    echo "========================================"
    echo "Date: $(date)"
    echo "Go Version: $(go version)"
    echo "OS: $(uname -s -r)"
    echo "CPU: $(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Unknown")"
    echo "CPU Cores: $(sysctl -n hw.ncpu 2>/dev/null || nproc)"
    echo "Memory: $(sysctl -n hw.memsize 2>/dev/null | awk '{print int($1/1024/1024/1024) "GB"}' || echo "Unknown")"
    echo "========================================"
    echo ""
} > "$RESULTS_FILE"

# Core performance benchmarks
echo -e "${GREEN}1. Core Performance Benchmarks${NC}"
run_benchmark "Single Service Notification" "BenchmarkSingleServiceNotification" ""
run_benchmark "Multiple Service Notification" "BenchmarkMultipleServiceNotification" ""
run_benchmark "Service Creation" "BenchmarkServiceCreation" ""

# Concurrency benchmarks
echo -e "${GREEN}2. Concurrency Benchmarks${NC}"
run_benchmark "Concurrent Notifications" "BenchmarkConcurrentNotifications" ""
run_benchmark "High Concurrency" "BenchmarkHighConcurrency" ""
run_benchmark "Goroutine Usage" "BenchmarkGoroutineUsage" ""

# Memory and attachment benchmarks
echo -e "${GREEN}3. Memory and Resource Benchmarks${NC}"
run_benchmark "Memory Usage" "BenchmarkMemoryUsage" ""
run_benchmark "Attachment Handling" "BenchmarkAttachmentHandling" ""
run_benchmark "Large Payload" "BenchmarkLargePayload" ""

# Service type variations
echo -e "${GREEN}4. Service Type Benchmarks${NC}"
run_benchmark "Service Type Variations" "BenchmarkServiceTypeVariations" ""

# Error handling and edge cases
echo -e "${GREEN}5. Error Handling Benchmarks${NC}"
run_benchmark "Error Handling" "BenchmarkErrorHandling" ""
run_benchmark "Timeout Handling" "BenchmarkTimeoutHandling" ""

# Stress testing (only if not in short mode)
if [ "$1" != "-short" ]; then
    echo -e "${GREEN}6. Stress Testing${NC}"
    run_benchmark "Stress Test" "BenchmarkStressTest" "-timeout 5m"
fi

# Profiling runs
echo -e "${GREEN}7. Profiling Runs${NC}"
run_with_profiling "Core Benchmarks with Profiling" "BenchmarkSingleService"

# Performance comparison over time
echo -e "${GREEN}8. Performance Comparison${NC}"
if [ -f "${BENCHMARK_OUTPUT_DIR}/baseline.txt" ]; then
    echo -e "${CYAN}Comparing with baseline performance...${NC}"
    
    # Extract key metrics and compare
    current_single=$(grep "BenchmarkSingleServiceNotification" "$RESULTS_FILE" | awk '{print $3}' | head -1)
    baseline_single=$(grep "BenchmarkSingleServiceNotification" "${BENCHMARK_OUTPUT_DIR}/baseline.txt" | awk '{print $3}' | head -1)
    
    if [ -n "$current_single" ] && [ -n "$baseline_single" ]; then
        echo "Single Service Performance:"
        echo "  Current:  $current_single ns/op"
        echo "  Baseline: $baseline_single ns/op"
        
        # Calculate percentage change
        if command -v bc >/dev/null 2>&1; then
            change=$(echo "scale=2; (($current_single - $baseline_single) / $baseline_single) * 100" | bc)
            if (( $(echo "$change > 0" | bc -l) )); then
                echo -e "  Change:   ${RED}+${change}% (slower)${NC}"
            else
                echo -e "  Change:   ${GREEN}${change}% (faster)${NC}"
            fi
        fi
    fi
else
    echo -e "${YELLOW}No baseline found. Saving current results as baseline.${NC}"
    cp "$RESULTS_FILE" "${BENCHMARK_OUTPUT_DIR}/baseline.txt"
fi

# Generate performance report
echo ""
echo -e "${GREEN}9. Generating Performance Report${NC}"

REPORT_FILE="${BENCHMARK_OUTPUT_DIR}/report_${TIMESTAMP}.md"

cat > "$REPORT_FILE" << EOF
# Apprise-Go Performance Report

**Generated:** $(date)  
**Go Version:** $(go version)  
**System:** $(uname -s -r)

## Summary

This report contains comprehensive performance benchmarks for the Apprise-Go notification library.

## Benchmark Results

\`\`\`
$(cat "$RESULTS_FILE")
\`\`\`

## Key Metrics

### Single Service Performance
$(grep "BenchmarkSingleServiceNotification" "$RESULTS_FILE" | head -1)

### Multi-Service Performance  
$(grep "BenchmarkMultipleServiceNotification" "$RESULTS_FILE" | head -1)

### Concurrency Performance
$(grep "BenchmarkConcurrentNotifications" "$RESULTS_FILE" | head -1)

### Memory Efficiency
$(grep "BenchmarkMemoryUsage" "$RESULTS_FILE" | head -1)

## Files Generated

- **Results:** $RESULTS_FILE
- **CPU Profile:** $CPU_PROFILE_FILE
- **Memory Profile:** $MEM_PROFILE_FILE
- **Report:** $REPORT_FILE

## Analysis Commands

View CPU profile:
\`\`\`bash
go tool pprof $CPU_PROFILE_FILE
\`\`\`

View memory profile:
\`\`\`bash
go tool pprof $MEM_PROFILE_FILE
\`\`\`

## Performance Recommendations

$(if grep -q "FAIL" "$RESULTS_FILE"; then echo "⚠️  Some benchmarks failed - check results for details"; fi)
$(if [ -s "$CPU_PROFILE_FILE" ]; then echo "📊 CPU profiling data available for detailed analysis"; fi)
$(if [ -s "$MEM_PROFILE_FILE" ]; then echo "📊 Memory profiling data available for detailed analysis"; fi)

EOF

echo ""
echo -e "${GREEN}Benchmark suite completed!${NC}"
echo ""
echo -e "${CYAN}Results saved to:${NC}"
echo "  📄 Results: $RESULTS_FILE"
echo "  📊 CPU Profile: $CPU_PROFILE_FILE"
echo "  📊 Memory Profile: $MEM_PROFILE_FILE"
echo "  📋 Report: $REPORT_FILE"
echo ""

# Summary statistics
if command -v wc >/dev/null 2>&1; then
    total_benchmarks=$(grep -c "^Benchmark" "$RESULTS_FILE" || echo "0")
    failed_benchmarks=$(grep -c "FAIL" "$RESULTS_FILE" || echo "0")
    
    echo -e "${CYAN}Summary:${NC}"
    echo "  Total benchmarks run: $total_benchmarks"
    echo "  Failed benchmarks: $failed_benchmarks"
    
    if [ "$failed_benchmarks" -eq 0 ]; then
        echo -e "  Status: ${GREEN}✅ All benchmarks passed${NC}"
    else
        echo -e "  Status: ${YELLOW}⚠️  Some benchmarks failed${NC}"
    fi
fi

echo ""
echo -e "${BLUE}To analyze profiles, run:${NC}"
echo "  go tool pprof $CPU_PROFILE_FILE"
echo "  go tool pprof $MEM_PROFILE_FILE"
echo ""
echo -e "${GREEN}Benchmark suite complete! 🚀${NC}"