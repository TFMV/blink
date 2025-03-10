# Blink Performance Metrics

*Report generated on March 10, 2025 at 10:56 AM CDT*

## Memory Profile Analysis

### Memory Allocations (Top Consumers)

```
File: blink.test
Type: alloc_space
Time: 2025-03-10 10:51:26 CDT
Showing nodes accounting for 132.11MB, 98.14% of 134.61MB total

Top Memory Consumers:
1. os.(*File).readdir: 62.06MB (46.11%)
2. os.lstatNolog: 13.50MB (10.03%)
3. syscall.ByteSliceFromString: 10.50MB (7.80%)
4. github.com/fsnotify/fsnotify.(*watches).add: 9.52MB (7.07%)
5. os.newFile: 6.50MB (4.83%)
6. strings.(*Builder).grow: 5.50MB (4.09%)
7. github.com/fsnotify/fsnotify.(*watches).addUserWatch: 4.51MB (3.35%)
```

### Cumulative Memory Operations

- `github.com/TFMV/blink/pkg/blink.(*Watcher).Start`: 125.60MB (93.31%)
- `github.com/TFMV/blink/pkg/blink.(*Watcher).addWatches`: 125.60MB (93.31%)
- `path/filepath.Walk`: 125.60MB (93.31%)
- `github.com/fsnotify/fsnotify.(*kqueue).Add`: 83.07MB (61.71%)

## Benchmark Results

### Watcher Performance

```
BenchmarkWatcher-10
- Operations/sec: 484 ops/sec
- Time/op: 2.07 ms/op
- Memory/op: 85,124 B/op
- Allocations/op: 530 allocs/op
```

### Filter Performance

```
BenchmarkFilterPerformance Results:
1. NoFilters:
   - Operations/sec: 337,099,838
   - Time/op: 3.51 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0

2. IncludePatterns:
   - Operations/sec: 18,537,398
   - Time/op: 65.64 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0

3. ExcludePatterns:
   - Operations/sec: 5,711,019
   - Time/op: 212.4 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0

4. IncludeEvents:
   - Operations/sec: 5,038,678
   - Time/op: 242.2 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0

5. IgnoreEvents:
   - Operations/sec: 4,297,494
   - Time/op: 273.1 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0

6. AllFilters:
   - Operations/sec: 4,412,713
   - Time/op: 276.1 ns/op
   - Memory/op: 0 B/op
   - Allocations/op: 0
```

## Race Condition Analysis

- Race detector tests passed successfully
- No data races detected in core functionality
- All concurrent operations properly synchronized

## Resource Management

- File descriptor management improved
- Proper cleanup of resources in Close() method
- Batch processing implemented for high-volume operations

## Notes

1. Memory usage is stable and consistent across operations
2. Main memory allocations are from expected file system operations
3. No memory leaks detected in core functionality
4. File descriptor management optimized for high-volume operations
5. All tests pass with race detector enabled

## Test Environment

- OS: Darwin 24.4.0
- Architecture: arm64
- CPU: Apple M2 Pro
- Go Version: 1.24
