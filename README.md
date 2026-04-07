# blast

A concurrent HTTP load tester built in Go.

## Usage

```bash
go run ./cmd/blast \
  --url https://httpbin.org/get \
  --c 20 \
  --n 500 \
  --rate 100 \
  --timeout 5s
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | required | Target URL |
| `--c` | 10 | Concurrent workers |
| `--n` | 100 | Total requests |
| `--rate` | 0 (unlimited) | Max requests/sec |
| `--timeout` | 30s | Per-request timeout |
| `--method` | GET | HTTP method |

### Example Output

```
Blast → https://httpbin.org/get
Workers: 20 | Requests: 500 | Timeout: 5s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Requests:    500
Duration:    4.21s
RPS:         118.8

Latency:
  p50   312ms
  p75   489ms
  p90   601ms
  p95   712ms
  p99   891ms
  max   1.2s

Status Codes:
  200    500

Errors: 0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Architecture

```
main
 └─ signal.NotifyContext   ← ctrl-c cancels everything
 └─ dispatch goroutine     ← pushes N tokens into jobs channel
       └─ rate limiter     ← time.Ticker gates throughput
 └─ worker.Pool            ← N goroutines pulling from jobs
       └─ worker[0..N]     ← each makes HTTP requests
            └─ results channel (fan-in)
 └─ stats.Collect          ← drains results, builds report
 └─ stats.Print            ← percentiles, RPS, status codes
```

## Concurrency Patterns

- **Worker pool** — fixed N goroutines pulling from a buffered jobs channel
- **Fan-in** — all workers write to a single results channel
- **Rate limiting** — `time.Ticker` controls dispatch throughput
- **Graceful shutdown** — `signal.NotifyContext` + `select` on `ctx.Done()`
- **Atomic counter** — `sync/atomic` tracks in-flight requests without a mutex
- **WaitGroup** — waits for all workers to drain before closing results
- **Buffered channels** — backpressure between dispatcher and workers
