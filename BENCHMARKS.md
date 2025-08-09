# Apprise Go Performance Benchmarks

This document provides performance benchmarks for the Apprise Go notification library, measured on Apple M4 Pro with Go 1.21+.

## Executive Summary

- **Single notification**: ~880 ns/op with 720 B/op (9 allocations)
- **Concurrent notifications**: Excellent parallel performance with ~97µs/op
- **Service registry**: Very fast service creation at ~25 ns/op
- **Attachment handling**: Efficient with ~46 ns/op for adding attachments
- **Memory footprint**: Low allocation patterns, scales linearly

## Core Notification Performance

### Single Service Notifications

```
BenchmarkApprise_Notify-12    1327072    876.8 ns/op    720 B/op    9 allocs/op
```

**Analysis**: Basic notification sending is highly optimized with sub-microsecond latency and minimal memory allocation.

### Multiple Service Scaling

| Services | Time/op | Memory/op | Allocations/op |
|----------|---------|-----------|----------------|
| 1        | 889 ns  | 720 B     | 9              |
| 5        | 2.8 µs  | 1.9 KB    | 17             |
| 10       | 4.5 µs  | 3.5 KB    | 27             |
| 25       | 9.1 µs  | 8.2 KB    | 57             |
| 50       | 16.8 µs | 15.9 KB   | 107            |

**Analysis**: Performance scales linearly with service count due to concurrent execution. Memory usage is predictable and efficient.

### Network Latency Impact

| Delay | Time/op | Performance Impact |
|-------|---------|-------------------|
| 0ms   | 897 ns  | Baseline          |
| 10ms  | 10.9 ms | +1,215,000%       |
| 50ms  | 50.8 ms | +5,565,000%       |
| 100ms | 100.9 ms| +11,150,000%      |

**Analysis**: As expected, network latency dominates performance. The library adds minimal overhead to network operations.

## Concurrent Performance

```
BenchmarkApprise_ConcurrentNotify-12    12261    97216 ns/op    816 B/op    10 allocs/op
```

**Analysis**: Excellent concurrent performance with goroutine-safe operations and minimal lock contention.

## Attachment Performance

### Adding Attachments

```
BenchmarkAttachmentManager_AddFile-12    25658889    46.62 ns/op    96 B/op    2 allocs/op
```

**Analysis**: Attachment operations are highly optimized with constant-time performance regardless of file size.

### Attachment Size Scaling

| Size  | Time/op | Memory/op | Notes |
|-------|---------|-----------|-------|
| 1KB   | 46.7 ns | 96 B      | Baseline |
| 10KB  | 46.3 ns | 96 B      | No degradation |
| 100KB | 46.5 ns | 96 B      | No degradation |
| 1MB   | 42.0 ns | 96 B      | Slightly faster |

**Analysis**: Attachment metadata operations are O(1) - file size doesn't affect performance during registration.

### Multiple Attachments

| Files | Time/op | Memory/op | Allocations/op |
|-------|---------|-----------|----------------|
| 1     | 88.9 ns | 112 B     | 3              |
| 5     | 495 ns  | 720 B     | 14             |
| 10    | 975 ns  | 1.5 KB    | 25             |
| 25    | 2.3 µs  | 3.4 KB    | 56             |

**Analysis**: Linear scaling for multiple attachments with efficient memory usage.

### Base64 Encoding Performance

| Size  | Time/op  | Memory/op | Throughput |
|-------|----------|-----------|------------|
| 1KB   | 862 ns   | 2.8 KB    | ~1.2 MB/s  |
| 10KB  | 7.9 µs   | 28.7 KB   | ~1.3 MB/s  |
| 100KB | 70 µs    | 279 KB    | ~1.4 MB/s  |
| 1MB   | 833 µs   | 2.8 MB    | ~1.2 MB/s  |

**Analysis**: Base64 encoding performance is consistent across file sizes with ~1.3 MB/s throughput.

### Hash Generation Performance

| Size  | Time/op  | Memory/op | Throughput |
|-------|----------|-----------|------------|
| 1KB   | 1.4 µs   | 72 B      | ~730 MB/s  |
| 10KB  | 12.6 µs  | 72 B      | ~810 MB/s  |
| 100KB | 126 µs   | 72 B      | ~810 MB/s  |
| 1MB   | 1.26 ms  | 74 B      | ~810 MB/s  |

**Analysis**: MD5 hashing maintains consistent high throughput (~810 MB/s) with minimal memory overhead.

## Service Registry Performance

```
BenchmarkServiceRegistry_Create-12    48901498    25.49 ns/op    48 B/op    1 allocs/op
```

**Analysis**: Service creation is extremely fast with map-based registry lookup.

## URL Parsing Performance

| Service  | Time/op | Memory/op | Allocations/op |
|----------|---------|-----------|----------------|
| Discord  | 30.2 ns | 48 B      | 1              |
| Telegram | 73.7 ns | 161 B     | 2              |
| Slack    | 207 ns  | 224 B     | 6              |
| Email    | 219 ns  | 240 B     | 7              |
| Webhook  | 319 ns  | 480 B     | 6              |

**Analysis**: URL parsing performance varies by service complexity. Discord has the simplest parsing, while Webhook services require more processing.

## Memory Allocation Patterns

### Notification with Attachments
```
BenchmarkApprise_NotifyWithAttachments-12    1203832    1002 ns/op    880 B/op    11 allocs/op
```

### Memory Allocation per Operation
```
BenchmarkApprise_MemoryAllocation-12    495824    2577 ns/op    2584 B/op    21 allocs/op
```

**Analysis**: Memory allocations are minimal and predictable. The library creates new instances efficiently without excessive heap pressure.

## Timeout Handling Performance

```
BenchmarkApprise_Timeout-12    10    100868371 ns/op    816 B/op    10 allocs/op
```

**Analysis**: Timeout handling adds minimal overhead. The 100ms delay is due to the mock service delay, not timeout processing overhead.

## Performance Recommendations

### For High-Throughput Applications

1. **Batch notifications** when possible to amortize service setup costs
2. **Use appropriate timeouts** - default 30s is good for most use cases
3. **Leverage concurrency** - the library handles multiple services concurrently
4. **Pre-register services** rather than parsing URLs repeatedly

### For Low-Latency Applications

1. **Minimize service count** for fastest single notifications
2. **Use connection pooling** for HTTP-based services (webhook, email)
3. **Consider caching** parsed service configurations
4. **Pre-validate attachments** to avoid runtime errors

### For Memory-Constrained Environments

1. **Clear attachments** after each notification batch
2. **Limit attachment sizes** using AttachmentManager.SetMaxSize()
3. **Avoid keeping large Apprise instances** in memory
4. **Use streaming** for large attachment processing

## Benchmark Environment

- **CPU**: Apple M4 Pro
- **OS**: macOS (Darwin 24.6.0)
- **Go Version**: 1.21+
- **Architecture**: arm64
- **Concurrency**: 12 CPU cores utilized

## Running Benchmarks

To run these benchmarks yourself:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./apprise

# Run specific benchmark
go test -bench=BenchmarkApprise_Notify -benchmem ./apprise

# Run with longer duration for more stable results
go test -bench=. -benchtime=10s -benchmem ./apprise

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./apprise

# Generate memory profile  
go test -bench=. -memprofile=mem.prof ./apprise
```

## Profiling Integration

The benchmarks are designed to work with Go's built-in profiling tools:

```bash
# CPU profiling
go test -bench=BenchmarkApprise_Notify -cpuprofile=cpu.prof ./apprise
go tool pprof cpu.prof

# Memory profiling
go test -bench=BenchmarkAttachment_Base64 -memprofile=mem.prof ./apprise
go tool pprof mem.prof

# Trace analysis
go test -bench=BenchmarkApprise_ConcurrentNotify -trace=trace.out ./apprise
go tool trace trace.out
```

---

*Benchmarks last updated: January 2025*