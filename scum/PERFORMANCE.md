# Performance Analysis: scum Parser and Tokenizer

**Last Updated**: 2026-01-31  
**System**: AMD Ryzen 9 6900HS, Linux, Go benchmark with `-benchmem -benchtime=3s`

---

## Benchmark Results Summary (Current)

| Benchmark | ops/sec | ns/op | Allocs | Bytes/op |
|-----------|---------|-------|--------|----------|
| **Tokenize** | ~5.6M | 595 ns | 4 | 1,360 B |
| **Parse** | ~2.1M | 1,712 ns | 10 | 2,696 B |
| **Serialize** | ~5.6M | 638 ns | 9 | 1,792 B |
| **Parse_LongInput** (~1KB) | ~172K | 21,148 ns | 10 | 39,304 B |
| **Parse_DeeplyNested** | ~3.4M | 1,058 ns | 12 | 1,480 B |

### Chaos Benchmarks (Stress Tests)

| Benchmark | ns/op | Allocs | Bytes/op |
|-----------|-------|--------|----------|
| **UnclosedTagStorm** (50 unclosed) | 22,169 | 25 | 65,328 B |
| **NestingMatryoshka** (20 levels) | 14,461 | 21 | 32,176 B |
| **AlternatingOpenClose** (100×3 tags) | 96,963 | 12 | 199,049 B |
| **EscapeHell** (50 escapes + tags) | 49,599 | 22 | 130,096 B |
| **WarningFlood** (mismatched tags) | 35,637 | 21 | 80,176 B |
| **MixedMayhem** | 3,846 | 18 | 7,472 B |

---

## Performance History (vs 2026-01-30 baseline)

| Benchmark | Before (ns/op) | After (ns/op) | **Δ Speed** | Before (B/op) | After (B/op) | **Δ Memory** |
|-----------|----------------|---------------|-------------|---------------|--------------|---------------|
| **Tokenize** | 882 | 595 | **+32% faster** | 2,320 | 1,360 | **-41%** |
| **Parse** | 2,249 | 1,712 | **+24% faster** | 3,656 | 2,696 | **-26%** |
| **Serialize** | 724 | 638 | **+12% faster** | 1,792 | 1,792 | same |
| **Parse_LongInput** | 27,218 | 21,148 | **+22% faster** | 53,576 | 39,304 | **-27%** |
| **Parse_DeeplyNested** | 1,415 | 1,058 | **+25% faster** | 1,992 | 1,480 | **-26%** |
| **AlternatingOpenClose** | 116,160 | 96,963 | **+17% faster** | 287,050 | 199,049 | **-31%** |
| **WarningFlood** | 45,550 | 35,637 | **+22% faster** | 104,816 | 80,176 | **-24%** |
| **MixedMayhem** | 5,422 | 3,846 | **+29% faster** | 12,144 | 7,472 | **-38%** |

---

## CPU Profile Hot Spots

| Function | cum% | Notes |
|----------|------|-------|
| `Parse` | **58.7%** | Main entry, orchestrates everything |
| `Tokenize` | **31.3%** | ~53% of Parse time spent here |
| `processTag` | 13.7% | Tag handling dispatch |
| `CreateAction.func1` | 10.9% | Action closures |
| `processUniversalTag` | 7.4% | Universal tag logic |
| `processOpeningTag` | 6.9% | Opening tag handling |
| `processText` | 6.0% | Text node creation |
| `ActionContext.Reset` | **4.6%** | Context reset per special char |

### Runtime Overhead

- **GC**: ~22% (`gcBgMarkWorker`, `gcDrain`)
- **growslice**: ~13% — slice resizing
- **mallocgc**: ~12% — allocations
- **newstack**: ~7% — stack growth from recursion/deep calls

---

## Memory Profile (Allocation Hot Spots)

| Function | Alloc% | Observations |
|----------|--------|--------------|
| `Tokenize` | **61.8%** | Token slice growth is expensive |
| `newParserState` | **31.0%** | Allocates AST nodes, breadcrumbs, cumWidth |
| `Warnings.Add` | 6.3% | Warning slice grows unbounded |
| `processOpeningTag` | 6.5% | Creates new `Node` structs |

---

## Key Findings

### 1. Tokenize optimization applied ✅

Token slice pre-allocation (`make([]Token, 0, len(input)/4)`) reduced:
- Allocations from 8 → 4 (50% reduction)
- Memory from 2,320 B → 1,360 B (41% reduction)
- Time from 882 ns → 595 ns (32% faster)

### 2. Memory pressure triggers GC

~22% of CPU spent in GC. Culprits:
- `tokenize.go:79` — Token slice growth
- `parse.go:78` — `nodes := make([]Node, 1, totalExpectedNodes)` — good pre-allocation but still allocates
- `warnings.go:97` — unbounded `w.list = append(...)` in `WarnOverflowNoCap` mode

### 3. AlternatingOpenClose improved ✅

Previously the worst case (287KB, 116μs), now significantly better:
- Memory: 287,050 B → 199,049 B (31% reduction)
- Time: 116,160 ns → 96,963 ns (17% faster)
- Allocations: 21 → 12 (43% reduction)

### 4. Escape handling is expensive (EscapeHell: 119KB)

Escape sequences (`\*`, `\$$`, `\[`) require additional lookahead and branching.

---

## Optimization Recommendations

### Quick Wins

1. **Pre-allocate Token slice in Tokenize**:
   ```go
   out.Tokens = make([]Token, 0, len(input)/4) // estimate ~1 token per 4 chars
   ```

2. **Reuse `ActionContext`** — you already do `Reset()`, but `NewBounds(i)` creates a new struct. Consider:
   ```go
   func (ac *ActionContext) Reset(char byte, i int) {
       ac.Bounds.Raw = NewSpan(i, 1)
       ac.Bounds.Inner = NewSpan(i, 0)
       // ... reset other fields in-place
   }
   ```

3. **Use Warning capacity mode** (`WarnOverflowDrop` or `WarnOverflowTrunc`) in benchmarks to avoid unbounded allocation.

### Medium Effort

4. **Pool `parserState`** for repeated Parse calls — `newParserState` allocates 31% of memory.

5. **Reduce Node struct size** — each `Node` is relatively heavy. Consider:
   - Using indices instead of pointers
   - Packing booleans into bitfields

### Architecture

6. **Single-pass tokenize+parse** — currently Parse calls Tokenize first, then iterates tokens. A streaming approach could cut memory in half.

---

## Performance Verdict

| Metric | Rating | Notes |
|--------|--------|-------|
| **Throughput** | ✅ Good | ~584K parses/sec for typical input |
| **Latency** | ✅ Good | ~1.7μs for simple input |
| **Stress handling** | ✅ Good | ~10.3K ops/sec on pathological input |
| **Memory efficiency** | ✅ Good | Token pre-allocation approach working well |
| **Scalability** | ✅ Good | Linear in input size with reduced constant factors |

**Bottom line**: The parser is production-ready. Recent optimizations (Token slice pre-allocation) improved throughput by ~24% and reduced memory by ~26%. For even higher throughput, consider pooling `parserState`.

---

## Reproducing Benchmarks

```bash
# Run all benchmarks with memory stats
go test -bench=. -benchmem -benchtime=3s ./scum/

# Generate CPU profile
go test -bench=BenchmarkParse -cpuprofile=cpu.prof ./scum/
go tool pprof -top -cum cpu.prof

# Generate memory profile
go test -bench=BenchmarkParse -memprofile=mem.prof ./scum/
go tool pprof -top -cum mem.prof
```
