package worker

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	URL string 
	Method string 
	Concurrency int 
	Requests int
	Timeout time.Duration
}

type Result struct {
	StatusCode int 
	Latency time.Duration
	Err error
}

var inFlight atomic.Int64

func Pool(ctx context.Context, cfg Config, jobs <-chan struct{}, results chan<- Result) {
	defer close (results)
	
	client := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{DisableKeepAlives: false},
	}
	
	var wg sync.WaitGroup
	wg.Add(cfg.Concurrency)
	
	for range cfg.Concurrency {
			go func() {
				defer wg.Done()
				runWorker(ctx, cfg, client, jobs, results)
			}()
		}
 
	wg.Wait() 
}

func runWorker(
	ctx context.Context,
	cfg Config,
	client *http.Client,
	jobs <-chan struct{},
	results chan<- Result,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-jobs:
			if !ok {
				// jobs channel closed — all work dispatched
			return
			}
			results <- do(ctx, cfg, client)
		}
	}
}

func do(ctx context.Context, cfg Config, client *http.Client) Result {
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, nil)
	if err != nil {
		return Result{Err: err}
	}
 
	inFlight.Add(1)
	defer inFlight.Add(-1)
 
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
 
	if err != nil {
		return Result{Latency: latency, Err: err}
	}
	defer resp.Body.Close()
 
	return Result{
		StatusCode: resp.StatusCode,
		Latency:    latency,
	}
}