package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JeffreyOmoakah/load-tester-go.git/internal/ratelimiter"
	"github.com/JeffreyOmoakah/load-tester-go.git/internal/stats"
	"github.com/JeffreyOmoakah/load-tester-go.git/internal/worker"
)

func main() {
	url := flag.String("url", "", "Target URL to load test (required)")
		concurrency := flag.Int("c", 10, "Number of concurrent workers")
		requests := flag.Int("n", 100, "Total number of requests to send")
		rate := flag.Int("rate", 0, "Max requests per second (0 = unlimited)")
		timeout := flag.Duration("timeout", 30*time.Second, "Per-request timeout (e.g. 5s, 500ms)")
		method := flag.String("method", "GET", "HTTP method")
		flag.Parse()
 
		if *url == "" {
			fmt.Fprintln(os.Stderr, "error: --url is required")
			flag.Usage()
			os.Exit(1)
		}
 
		// context that cancels on ctrl-c or SIGTERM
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
 
		cfg := worker.Config{
			URL:         *url,
			Method:      *method,
			Concurrency: *concurrency,
			Requests:    *requests,
			Timeout:     *timeout,
		}
 
		// buffered so the dispatcher never blocks on a slow worker
		jobs := make(chan struct{}, *concurrency)
		results := make(chan worker.Result, *concurrency*2)
 
		var rl *ratelimiter.RateLimiter
		if *rate > 0 {
			rl = ratelimiter.New(*rate)
		}
 
		fmt.Printf("\nBlast → %s\n", *url)
		fmt.Printf("Workers: %d | Requests: %d | Timeout: %s\n\n", *concurrency, *requests, *timeout)
 
		start := time.Now()
 
		// start the worker pool
		go worker.Pool(ctx, cfg, jobs, results)
 
		// dispatch jobs (respects rate limit + ctx cancellation)
		go dispatch(ctx, *requests, rl, jobs)
 
		// aggregate results as they come in
		report := stats.Collect(ctx, results, *requests)
 
		elapsed := time.Since(start)
		stats.Print(report, elapsed)

}

func dispatch(ctx context.Context, n int, rl *ratelimiter.RateLimiter, jobs chan<- struct{}) {
	defer close(jobs)
 
	for i := 0; i < n; i++ {
		// wait for rate limiter token if rate limiting is enabled
		if rl != nil {
			select {
			case <-ctx.Done():
				return
			case <-rl.C:
			}
		}
 
		select {
		case <-ctx.Done():
			return
		case jobs <- struct{}{}:
		}
	}
}
 