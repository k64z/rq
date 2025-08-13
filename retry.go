package rq

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	Jitter      bool
	RetryIf     func(*Response) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
		RetryIf:     defaultRetryIf,
	}
}

// defaultRetryIf retries on 5xx errors and network errors
func defaultRetryIf(resp *Response) bool {
	if resp.err != nil {
		return true
	}
	return resp.StatusCode >= 500 || resp.StatusCode == 429
}

// DoWithRetry executes the request with retry logic
func (r *Request) DoWithRetry(ctx context.Context, config *RetryConfig) *Response {
	if config == nil {
		config = DefaultRetryConfig()
	}

	if r.err != nil {
		return &Response{err: r.err}
	}

	// Read body into memory so we can retry
	var bodyBytes []byte
	if r.body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.body)
		if err != nil {
			return &Response{err: err}
		}
	}

	var resp *Response
	delay := config.Delay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if bodyBytes != nil {
			r.body = bytes.NewReader(bodyBytes)
		}

		resp = r.DoContext(ctx)

		if !config.RetryIf(resp) {
			return resp
		}

		if attempt == config.MaxAttempts-1 {
			break
		}

		if config.Jitter {
			delay = addJitter(delay)
		}

		select {
		case <-ctx.Done():
			resp.err = ctx.Err()
			return resp
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return resp
}

// addJitter adds random jitter to the delay
func addJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.3)
	return delay + jitter
}

// ExponentialBackoff returns a backoff function with exponential delay
func ExponentialBackoff(base time.Duration, multiplier float64, maxDelay time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		delay := base * time.Duration(math.Pow(multiplier, float64(attempt)))
		if delay > maxDelay {
			return maxDelay
		}
		return delay
	}
}

// LinearBackoff returns a backoff function with linear delay
func LinearBackoff(base, increment, maxDelay time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		delay := base + (increment * time.Duration(attempt))
		if delay > maxDelay {
			return maxDelay
		}
		return delay
	}
}

// ConstantBackoff returns a backoff function with constant delay
func ConstantBackoff(delay time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return delay
	}
}
