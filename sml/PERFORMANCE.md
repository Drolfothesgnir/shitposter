# Performance Analysis: sml Parser and Renderer

**Last Updated**: 2026-04-18  
**System**: AMD Ryzen 9 6900HS, Linux/amd64, Go `go1.26.1`, benchmarked with `-benchmem -benchtime=3s -count=3`

---

## Benchmark Results Summary (Current)

Numbers below are arithmetic means of three runs.

| Benchmark | ns/op | Allocs/op | Bytes/op |
|-----------|-------|-----------|----------|
| **Munch_Simple** | 4,654 | 23 | 11,784 B |
| **Munch_LongInput** | 157,374 | 153 | 304,730 B |
| **Munch_DeeplyNested** | 24,176 | 108 | 50,184 B |
| **Munch_ChaosIssues** | 83,712 | 284 | 162,615 B |
| **HTML_Render_Simple** | 543.0 | 6 | 768 B |
| **HTML_Render_LongInput** | 34,442 | 79 | 63,648 B |
| **Text_Render_LongInput** | 2,407 | 1 | 2,688 B |
| **TextByteLen_Read_LongInput** | 3.180 | 0 | 0 B |
| **MunchAndHTML_EndToEnd** | 105,827 | 132 | 184,697 B |

## Change Since 2026-04-14 Baseline

| Benchmark | Time | Allocs | Bytes |
|-----------|------|--------|-------|
| **Munch_Simple** | +1.6% | +9.5% | +2.6% |
| **Munch_LongInput** | +10.8% | +73.9% | +7.1% |
| **Munch_DeeplyNested** | -11.0% | +2.9% | +2.3% |
| **Munch_ChaosIssues** | +33.7% | +132.8% | +15.6% |
| **HTML_Render_Simple** | -64.7% | -68.4% | -42.5% |
| **HTML_Render_LongInput** | -65.9% | -89.9% | -35.8% |
| **Text_Render_LongInput** | -8.7% | 0.0% | 0.0% |
| **TextByteLen_Read_LongInput** | -14.6% | 0.0% | 0.0% |
| **MunchAndHTML_EndToEnd** | -15.1% | -70.9% | -4.0% |

### Quick Read

- Moving validation and normalization into `Munch` made render-only HTML much cheaper: long-input render time is down about 66%, and allocation count is down about 90%.
- `Munch` is more expensive for long and malformed inputs because it now owns attribute validation, issue construction and render-tree normalization.
- End-to-end long rendering is still faster overall: about 15% less time and 71% fewer allocations.
- `TextByteLen` is effectively constant-time and allocation-free.

---

## End-to-End Profile Hot Spots (Previous, Not Refreshed)

**Profile target**: `BenchmarkMunchAndHTML_EndToEnd`  
**Profile run**: `-benchtime=5s` on 2026-04-14

These profile tables are from before validation/normalization moved into `Munch`.
They are useful as historical context, but function names such as `handleAttributes`
and `handleNode` refer to the old render pipeline.

### CPU (cum%)

| Function | cum% | Notes |
|----------|------|-------|
| `sml.(*Eater).Munch` | 39.22% | Parse + warning serialization path dominates total work. |
| `scum.Parse` | 28.54% | Core parser pipeline under `Munch`. |
| `sml.Poop.HTML` | 25.98% | HTML rendering stage is a major second contributor. |
| `sml.handleNode` | 25.26% | Recursive node dispatch through rendered tree. |
| `sml.handleTag` | 23.72% | Tag open/close writes plus children traversal. |
| `sml.handleAttributes` | 16.02% | Attribute validation/rewriting is a notable render cost. |
| `scum.Tokenize` | 13.24% | Large parsing sub-cost from tokenization. |
| `scum.AST.Serialize` | 10.16% | AST -> serializable tree conversion cost before render. |

### Memory (alloc_space, flat%)

| Function | alloc% | Notes |
|----------|--------|-------|
| `scum.Tokenize` | 29.62% | Largest byte allocator in end-to-end flow. |
| `scum.newParserState` | 23.42% | Parser-state setup is heavy for total allocated bytes. |
| `scum.AST.Serialize` | 21.57% | Tree serialization contributes large allocation volume. |
| `strings.(*Builder).WriteString` | 13.32% | Output string assembly is a major allocation source. |
| `sml.Poop.HTML` | 3.33% flat / 22.07% cum | Render pipeline drives many downstream allocations. |
| `scum.NewWarnings` | 3.16% | Warning collector setup is small but visible. |

### Memory (alloc_objects, flat%)

| Function | alloc objects % | Notes |
|----------|------------------|-------|
| `strings.(*Builder).WriteString` | 46.10% | Most allocation count is small string writes. |
| `sml.handleAttributes` | 23.84% | Per-attribute string building/validation churn. |
| `scum.AST.Serialize` | 10.28% | Object creation during serialization. |
| `net/url.parse` | 7.70% | `href` parsing contributes many small objects. |
| `strings.(*byteStringReplacer).Replace` | 7.02% | HTML escaping contributes object churn. |

### Key Findings

- End-to-end time splits mostly between `Munch` (parse side) and `HTML` (render side), so optimization should target both halves.
- Allocation pressure is dominated by parser internals (`Tokenize`, `newParserState`, `AST.Serialize`) plus render-time string building.
- Attribute-heavy inputs amplified attribute validation, `net/url.parse`, and HTML escaping costs in the old render path.

---

## Reproducing Benchmarks

```bash
# Run all sml benchmarks with memory stats
go test -run=^$ -bench=. -benchmem -benchtime=3s -count=3 ./sml/

# Faster local sanity pass
go test -run=^$ -bench=. -benchmem ./sml/
```

## CPU Profiling

```bash
# Install pprof once if missing:
go install github.com/google/pprof@latest

# Profile parser path
go test -run=^$ -bench=BenchmarkMunch -benchtime=5s -cpuprofile=sml_cpu.prof ./sml/
pprof -top -cum sml_cpu.prof

# Profile end-to-end parse + HTML render
go test -run=^$ -bench=BenchmarkMunchAndHTML_EndToEnd -benchtime=5s -cpuprofile=sml_e2e_cpu.prof ./sml/
pprof -top -cum sml_e2e_cpu.prof
```

## Memory Profiling

```bash
# Parse-heavy allocations
go test -run=^$ -bench=BenchmarkMunch -benchtime=5s -memprofile=sml_mem.prof ./sml/
pprof -sample_index=alloc_space -top -cum sml_mem.prof

# End-to-end allocations
go test -run=^$ -bench=BenchmarkMunchAndHTML_EndToEnd -benchtime=5s -memprofile=sml_e2e_mem.prof ./sml/
pprof -sample_index=alloc_space -top -cum sml_e2e_mem.prof
pprof -sample_index=alloc_objects -top sml_e2e_mem.prof
```

## Advanced Profiling

```bash
# Interactive web UI
# (open http://localhost:8080 while command is running)
pprof -http=:8080 sml_cpu.prof

# Source-level breakdowns (examples)
pprof -list=Munch sml_cpu.prof
pprof -list=normalizeNode sml_cpu.prof
pprof -list=normalizeLink sml_cpu.prof
pprof -list=renderNode sml_e2e_cpu.prof

# Escape analysis
go build -gcflags='-m' ./sml/ 2>&1 | grep -E 'escape|heap'
go build -gcflags='-m -m' ./sml/ 2>&1 | head -100
```

## Regression Tracking

```bash
# Baseline run
go test -run=^$ -bench=. -benchmem -count=5 ./sml/ > old.txt

# After changes
go test -run=^$ -bench=. -benchmem -count=5 ./sml/ > new.txt

# Compare (requires: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat old.txt new.txt
```
