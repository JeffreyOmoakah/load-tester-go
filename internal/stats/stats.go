package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/JeffreyOmoakah/load-tester-go.git/internal/worker"
)

type Report struct {
	Total int 
	Errors int 
	StatusCodes map[int]int
	Latencies []time.Duration
}

func Collect(ctx context.Context, results <-chan worker.Result, expected int) Report {
	r := Report{
		StatusCodes: make(map[int]int),
		Latencies:   make([]time.Duration, 0, expected),
	}
 
	for {
		select {
		case res, ok := <-results:
			if !ok {
				// channel closed — all workers done
				sort.Slice(r.Latencies, func(i, j int) bool {
					return r.Latencies[i] < r.Latencies[j]
				})
				return r
			}
			r.Total++
			if res.Err != nil {
				r.Errors++
			} else {
				r.StatusCodes[res.StatusCode]++
				r.Latencies = append(r.Latencies, res.Latency)
			}
		case <-ctx.Done():
			sort.Slice(r.Latencies, func(i, j int) bool {
				return r.Latencies[i] < r.Latencies[j]
			})
			return r
		}
	}
}
 
// Print writes the final human-readable report to stdout.
func Print(r Report, elapsed time.Duration) {
	rps := float64(r.Total) / elapsed.Seconds()
 
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Requests:    %d\n", r.Total)
	fmt.Printf("Duration:    %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("RPS:         %.1f\n", rps)
 
	if len(r.Latencies) > 0 {
		fmt.Println("\nLatency:")
		fmt.Printf("  p50   %s\n", percentile(r.Latencies, 50))
		fmt.Printf("  p75   %s\n", percentile(r.Latencies, 75))
		fmt.Printf("  p90   %s\n", percentile(r.Latencies, 90))
		fmt.Printf("  p95   %s\n", percentile(r.Latencies, 95))
		fmt.Printf("  p99   %s\n", percentile(r.Latencies, 99))
		fmt.Printf("  max   %s\n", r.Latencies[len(r.Latencies)-1])
	}
 
	fmt.Println("\nStatus Codes:")
	for code, count := range r.StatusCodes {
		fmt.Printf("  %d    %d\n", code, count)
	}
 
	if r.Errors > 0 {
		pct := float64(r.Errors) / float64(r.Total) * 100
		fmt.Printf("\nErrors: %d (%.1f%%)\n", r.Errors, pct)
	} else {
		fmt.Println("\nErrors: 0")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
 
// percentile returns the pth percentile from a sorted latency slice.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	return sorted[idx].Round(time.Millisecond)
}
 