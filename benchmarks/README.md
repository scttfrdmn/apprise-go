# Apprise-Go Benchmarks & Load Testing

Comprehensive performance testing suite for Apprise-Go notification library and API server.

## Overview

This benchmarking suite provides:

- **Core Library Benchmarks** - Go benchmark tests for library performance
- **Load Testing** - High-concurrency API load testing with realistic scenarios
- **Performance Analysis** - Detailed performance metrics and recommendations
- **Automated Reporting** - Comprehensive performance reports and visualizations

## Quick Start

### Run All Benchmarks
```bash
cd benchmarks
./run_benchmarks.sh
```

### Run Specific Test Types
```bash
# Core library benchmarks only
./run_benchmarks.sh -t core

# Load testing only  
./run_benchmarks.sh -t load

# API-specific benchmarks
./run_benchmarks.sh -t api
```

### Custom Load Testing
```bash
cd load-tests
go run . -config stress -output results.json
```

## Benchmark Types

### 1. Core Library Benchmarks (`core_benchmarks_test.go`)

Tests the performance of core Apprise-Go library functions:

- **Basic Notifications** - Single and multi-service notification sending
- **Concurrency Testing** - Performance under various concurrency levels
- **Message Sizes** - Impact of different message sizes on performance
- **Service Creation** - Service instantiation and configuration overhead
- **Memory Allocation** - Memory usage patterns and GC behavior
- **Error Handling** - Performance impact of error processing
- **Realistic Scenarios** - Production-like test conditions

```bash
# Run core benchmarks
go test -bench=. -benchmem ./benchmarks/

# Run specific benchmarks
go test -bench=BenchmarkAppriseNotify -benchmem ./benchmarks/

# Run with different CPU counts
go test -bench=. -benchmem -cpu=1,2,4,8 ./benchmarks/
```

### 2. Load Testing (`load-tests/`)

High-level load testing framework with configurable scenarios:

#### Predefined Configurations

| Config | Concurrency | RPS | Duration | Description |
|--------|-------------|-----|----------|-------------|
| `quick` | 10 | 50 | 30s | Quick smoke test |
| `moderate` | 50 | 200 | 2m | Standard load test |
| `stress` | 100 | 500 | 5m | High-stress test |
| `endurance` | 25 | 100 | 30m | Long-running stability test |
| `library-only` | 20 | 100 | 1m | Library-only testing |

#### Usage Examples

```bash
cd load-tests

# Quick test
go run . -config quick

# Stress test with custom duration
go run . -config stress -duration 10m

# Custom server URL
go run . -config moderate -url http://production:8080

# Custom configuration file
go run . -config-file custom-config.json

# Override specific parameters
go run . -config quick -concurrency 20 -rps 100
```

#### Custom Configuration

Create custom test configurations:

```json
{
  "base_url": "http://localhost:8080",
  "concurrency": 50,
  "requests_per_second": 200,
  "duration": "2m",
  "test_type": "mixed",
  "notification_types": ["info", "success", "warning", "error"],
  "service_urls": ["webhook://httpbin.org/post"],
  "warmup_duration": "10s",
  "cooldown_duration": "5s",
  "enable_metrics": true,
  "metrics_interval": "2s"
}
```

### 3. Performance Metrics

The testing suite collects comprehensive metrics:

#### Response Time Metrics
- Average, median, 95th, and 99th percentile response times
- Min/max response times
- Response time distribution

#### Throughput Metrics  
- Requests per second (actual vs target)
- Success/error rates
- Throughput efficiency

#### Resource Metrics
- Memory usage (current, peak, average)
- Goroutine counts
- CPU utilization
- System resource consumption

#### Error Analysis
- Error rate distribution
- Error type categorization
- Failure pattern analysis

## Automated Benchmark Runner

The `run_benchmarks.sh` script provides comprehensive automated testing:

### Features
- **Multi-type Testing** - Run core, load, and API tests
- **Configurable Parameters** - Override test settings via CLI
- **Detailed Reporting** - Generate comprehensive performance reports
- **Environment Detection** - Automatic system and dependency checks
- **Result Archiving** - Organized result storage with timestamps

### Usage
```bash
./run_benchmarks.sh [OPTIONS]

OPTIONS:
    -t, --type TYPE        Benchmark type: all, core, load, api
    -o, --output DIR       Output directory
    -c, --config FILE      Custom load test config file
    -s, --server URL       API server URL
    -d, --duration TIME    Override test duration
    -p, --parallel N       Parallel benchmark processes
    -r, --repeat N         Number of repetitions
    -q, --quiet           Quiet mode
    -h, --help            Show help
```

### Examples
```bash
# Full benchmark suite
./run_benchmarks.sh

# Quick core benchmarks
./run_benchmarks.sh -t core -d 1m

# Production load testing
./run_benchmarks.sh -t load -s https://api.prod.com -c prod-config.json

# Repeated testing for reliability
./run_benchmarks.sh -r 5 -t core
```

## Performance Targets & SLAs

### Target Performance Metrics

| Metric | Target | Threshold |
|--------|---------|-----------|
| Success Rate | ≥99% | ≥95% |
| 95th Percentile Response Time | ≤200ms | ≤500ms |
| Average Response Time | ≤50ms | ≤100ms |
| Throughput Efficiency | ≥95% | ≥80% |
| Error Rate | ≤1% | ≤5% |

### Resource Limits
- **Memory Usage**: <1GB under normal load
- **Goroutines**: <1000 active goroutines
- **CPU Usage**: <80% on production hardware

## Interpreting Results

### Core Benchmark Results
```
BenchmarkAppriseNotify-8         	    5000	    245689 ns/op	    1024 B/op	      15 allocs/op
```
- `5000`: Number of iterations
- `245689 ns/op`: Nanoseconds per operation
- `1024 B/op`: Bytes allocated per operation
- `15 allocs/op`: Number of allocations per operation

### Load Test Results
```json
{
  "total_requests": 6000,
  "successful_requests": 5940,
  "error_rate": 0.01,
  "average_response_time": "45ms",
  "p95_response_time": "120ms",
  "requests_per_second": 198.5
}
```

### Performance Rating System
- **A (90-100 points)**: Excellent performance
- **B (80-89 points)**: Good performance  
- **C (70-79 points)**: Fair performance
- **D (60-69 points)**: Poor performance
- **F (<60 points)**: Failing performance

Rating is based on:
- Error rate (40 points max)
- Response times (40 points max)
- Throughput efficiency (20 points max)

## Performance Analysis & Optimization

### Common Performance Issues

1. **High Response Times**
   - Network latency to external services
   - Inefficient serialization/deserialization
   - Blocking I/O operations

2. **Memory Usage**
   - Large message payloads
   - Goroutine leaks
   - Inefficient data structures

3. **Low Throughput**
   - Insufficient concurrency
   - Rate limiting by external services
   - CPU or I/O bottlenecks

### Optimization Recommendations

#### Code Optimizations
```go
// Use connection pooling
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

// Optimize JSON marshaling
var jsonBuffer bytes.Buffer
encoder := json.NewEncoder(&jsonBuffer)
encoder.Encode(data)
```

#### Configuration Tuning
```bash
# Increase file descriptor limits
ulimit -n 65536

# Tune Go GC
export GOGC=100
export GOMEMLIMIT=1GiB
```

#### Infrastructure Optimizations
- Use HTTP/2 for external service connections
- Implement circuit breakers for unreliable services
- Add caching for frequent notification patterns
- Use async processing for high-throughput scenarios

## Continuous Integration

### GitHub Actions Integration
```yaml
name: Performance Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  benchmarks:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    - name: Run Benchmarks
      run: |
        cd benchmarks
        ./run_benchmarks.sh -t all -q
    - name: Upload Results
      uses: actions/upload-artifact@v3
      with:
        name: benchmark-results
        path: benchmarks/results/
```

### Performance Regression Detection
```bash
# Compare with baseline
./run_benchmarks.sh -t core > current_results.txt
benchcmp baseline_results.txt current_results.txt
```

## Troubleshooting

### Common Issues

**Low throughput during load testing:**
```bash
# Check system limits
ulimit -n
cat /proc/sys/net/core/somaxconn

# Monitor system resources
htop
iostat -x 1
```

**High memory usage:**
```bash
# Check for goroutine leaks
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# Profile memory usage
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Inconsistent results:**
- Run multiple iterations: `./run_benchmarks.sh -r 5`
- Check system load during tests
- Ensure stable network conditions
- Verify external services are responsive

### Debug Mode
```bash
# Enable debug logging
export DEBUG=1
./run_benchmarks.sh -t load

# Verbose load testing
go run . -config quick -verbose
```

## Advanced Usage

### Custom Metrics Collection
```go
// Add custom metrics to load tester
type CustomMetrics struct {
    DatabaseConnections int
    CacheHitRate       float64
    QueueDepth         int
}

func (lt *LoadTester) CollectCustomMetrics() *CustomMetrics {
    // Implementation
}
```

### Integration Testing
```bash
# Test with real services (be careful with rate limits)
go run . -config integration -url https://api.production.com
```

### Distributed Load Testing
```bash
# Run from multiple machines
./run_benchmarks.sh -t load -s http://target:8080 &
ssh remote1 "cd /path/to/benchmarks && ./run_benchmarks.sh -t load -s http://target:8080" &
ssh remote2 "cd /path/to/benchmarks && ./run_benchmarks.sh -t load -s http://target:8080" &
wait
```

## Contributing

When adding new benchmarks:

1. Follow existing naming conventions: `BenchmarkFeatureName`
2. Include memory allocation reporting: `b.ReportAllocs()`
3. Use `b.ResetTimer()` after setup
4. Test with multiple CPU counts: `b.SetParallelism(n)`
5. Add realistic test scenarios
6. Document expected performance characteristics

## Results Archive

Benchmark results are stored in `benchmarks/results/TIMESTAMP/`:
- `benchmark_report.md` - Comprehensive analysis report
- `core_benchmarks.txt` - Raw Go benchmark output
- `core_benchmarks.json` - Processed core benchmark data
- `load_test_*.json` - Load test results for each configuration
- `benchmark.log` - Detailed execution log

Results can be compared across runs to detect performance regressions and validate optimizations.