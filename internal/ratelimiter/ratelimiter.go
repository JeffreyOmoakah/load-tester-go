package ratelimiter

import "time"

type RateLimiter struct {
	C <-chan time.Time
	stop chan struct{}
}

// this approach is so the limiter doesn't care about the time but strictly enforces the Limit 
// by using a ticker with a fixed interval between tokens
func New(rps int) *RateLimiter {
	interval := time.Second / time.Duration(rps)
	ticker := time.NewTicker(interval)
	
	r1 := &RateLimiter{
		C:    ticker.C,
		stop: make(chan struct{}),
	}
	
	go func () {
		<- r1.stop
		ticker.Stop() // this stops the ticker manually to avoid goroutine leaks
	}()
	
	return r1
}

func (r1 *RateLimiter) Stop() {
	close(r1.stop)
}