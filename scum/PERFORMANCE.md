# Performance Analysis: scum Parser and Tokenizer

**Last Updated**: 2026-04-18  
**System**: AMD Ryzen 9 6900HS, Linux, Go benchmark with `-benchmem -benchtime=3s`

---

## External-AST Model Benchmark (Current)

Command:

```bash
go test -run '^$' -bench 'BenchmarkParseOuterAST' -benchmem -benchtime=3s -count=3 ./scum
```

The table uses the average of 3 runs. `Parse` is the return-AST wrapper. `ParseIntoCallerAST`
uses one caller-owned AST reused across iterations. `ParseIntoPooledAST` gets an AST from a
`sync.Pool`, parses into it, clears only ownership-sensitive fields, then returns it to the pool.

| Input | Mode | ns/op | B/op | allocs/op | Speed vs `Parse` | Bytes vs `Parse` |
|-------|------|-------|------|-----------|------------------|------------------|
| Simple | `Parse` | 2,058 | 2,643 | 5 | baseline | baseline |
| Simple | `ParseIntoCallerAST` | 1,667 | 1,361 | 4 | 19.0% faster | 48.5% less |
| Simple | `ParseIntoPooledAST` | 1,681 | 1,362 | 4 | 18.3% faster | 48.5% less |
| LongInput (~1KB) | `Parse` | 25,277 | 39,291 | 5 | baseline | baseline |
| LongInput (~1KB) | `ParseIntoCallerAST` | 22,289 | 18,788 | 4 | 11.8% faster | 52.2% less |
| LongInput (~1KB) | `ParseIntoPooledAST` | 21,712 | 18,806 | 4 | 14.1% faster | 52.1% less |
| DeeplyNested | `Parse` | 1,140 | 1,361 | 5 | baseline | baseline |
| DeeplyNested | `ParseIntoCallerAST` | 952 | 784 | 4 | 16.5% faster | 42.4% less |
| DeeplyNested | `ParseIntoPooledAST` | 967 | 785 | 4 | 15.2% faster | 42.3% less |

### Why External AST Wins

The old return-AST model makes each `Parse` call start with an empty AST value. That is simple,
but it means the parser must allocate fresh `Nodes` and `Attributes` backing arrays for every
parse. Returning the AST by value is not the main cost: the AST header is small. The cost is that
the arenas owned by that AST cannot be reused by the next parse.

The external-AST model changes ownership:

- The caller owns the `AST` and passes `*AST` to `ParseInto`.
- The parser resets the AST metadata for the new input.
- Existing `Nodes` and `Attributes` backing arrays are reused when their capacity is suitable.
- Explicit limits still control memory retention: if `MaxNodes` or `MaxAttributes` is lower than
  the reused backing capacity, `ParseInto` allocates a smaller arena instead of retaining the old
  oversized one.

This removes one allocation per parse in the measured cases and cuts bytes/op by roughly 42-52%.
The remaining allocation pressure mostly comes from tokenization and warning storage, not AST arena
creation.

`ParseIntoCallerAST` is the best model when one worker or loop can keep a local AST and reuse it
serially. It has no pool overhead and makes ownership obvious.

`ParseIntoPooledAST` is useful for request-style workloads where each parse needs a separate AST
object but the application can return ASTs to a pool after rendering/serialization is complete.
Pooling is not magic: it performs about the same as direct caller-owned reuse, sometimes a little
slower from `sync.Pool` overhead, sometimes a little faster from run-to-run noise. Its value is
sharing warmed AST arenas across independent requests.

### Ownership Rules

- Do not reuse or return an AST to a pool while any renderer, serializer, or caller still reads it.
- Reusing an AST invalidates its previous `Input`, `Nodes`, and `Attributes` contents.
- Pooled ASTs can retain large arenas. Use `MaxNodes` / `MaxAttributes`, or drop oversized ASTs
  before returning them to an application-level pool.
- Unit tests should usually use plain AST values. Pooling belongs in benchmarks or explicit pool
  lifecycle tests, otherwise ownership bugs can get hidden.

---

## Older Full Benchmark Snapshot (2026-02-01)

> Historical note: the remaining profile details and recommendations from this
> snapshot predate parser-state pooling and the external-AST `ParseInto` model.
> Treat them as baseline context, not current profiler output.

| Benchmark | ops/sec | ns/op | Allocs | Bytes/op |
|-----------|---------|-------|--------|----------|
| **Tokenize** | ~5.6M | 595 ns | 4 | 1,360 B |
| **Parse** | ~2.1M | 1,712 ns | 10 | 2,696 B |
| **Serialize** | ~6.6M | 555 ns | 5 | 1,728 B |
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
| **Serialize** | 724 | 555 | **+23% faster** | 1,792 | 1,728 | **-4%** |
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

### 1. Serialize optimization applied ✅

Pre-allocating a single backing array for all child nodes reduced:
- Allocations from 9 → 5 (44% reduction)
- Memory from 1,792 B → 1,728 B (4% reduction)
- Time from 638 ns → 555 ns (13% faster)

### 2. Tokenize optimization applied ✅

Token slice pre-allocation (`make([]Token, 0, len(input)/4)`) reduced:
- Allocations from 8 → 4 (50% reduction)
- Memory from 2,320 B → 1,360 B (41% reduction)
- Time from 882 ns → 595 ns (32% faster)

### 3. Memory pressure triggers GC

~22% of CPU spent in GC. Culprits:
- `tokenize.go:79` — Token slice growth
- `parse.go:78` — `nodes := make([]Node, 1, totalExpectedNodes)` — good pre-allocation but still allocates
- `warnings.go:97` — unbounded `w.list = append(...)` in `WarnOverflowNoCap` mode

### 4. AlternatingOpenClose improved ✅

Previously the worst case (287KB, 116μs), now significantly better:
- Memory: 287,050 B → 199,049 B (31% reduction)
- Time: 116,160 ns → 96,963 ns (17% faster)
- Allocations: 21 → 12 (43% reduction)

### 5. Escape handling is expensive (EscapeHell: 119KB)

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

## Advanced Profiling

```bash
# Interactive web UI (best for exploration - opens flame graphs, call graphs)
go tool pprof -http=:8080 cpu.prof

# Line-by-line source timing
go tool pprof -list=Tokenize cpu.prof
go tool pprof -list=Parse cpu.prof

# Escape analysis (what allocates on heap vs stack)
go build -gcflags='-m' ./scum/ 2>&1 | grep -E 'escape|heap'

# Verbose escape analysis (shows reasoning)
go build -gcflags='-m -m' ./scum/ 2>&1 | head -100

# Benchmark comparison (detect regressions)
# First, save baseline:
go test -bench=. -benchmem -count=5 ./scum/ > old.txt
# After changes:
go test -bench=. -benchmem -count=5 ./scum/ > new.txt
# Compare (requires: go install golang.org/x/perf/cmd/benchstat@latest):
benchstat old.txt new.txt
```
