package rq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOnServerError(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt := atomic.AddInt32(&attempts, 1)
		if cnt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer srv.Close()

	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
		RetryIf:     defaultRetryIf,
	}

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryOnRateLimit(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt := atomic.AddInt32(&attempts, 1)
		if cnt < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Retry-After", "1")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	config := DefaultRetryConfig()
	config.Delay = 10 * time.Millisecond

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("want 2 attempts, got %d", attempts)
	}
}

func TestRetryMaxAttempts(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
		RetryIf:     defaultRetryIf,
	}

	resp := Get(srv.URL).DoWithRetry(ctx, config)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("want status 500, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryCustomConditions(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt := atomic.AddInt32(&attempts, 1)
		if cnt < 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	config := &RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
		RetryIf: func(r *Response) bool {
			return r.StatusCode == http.StatusBadRequest
		},
	}

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryWithJitter(t *testing.T) {
	var attempts int32
	var delays []time.Duration
	lastTime := time.Now()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		if atomic.LoadInt32(&attempts) > 0 {
			delays = append(delays, now.Sub(lastTime))
			lastTime = now
		}

		cnt := atomic.AddInt32(&attempts, 1)
		if cnt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts: 3,
		Delay:       50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      true,
		RetryIf:     defaultRetryIf,
	}

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if len(delays) != 2 {
		t.Fatalf("want 2 delays, got %d", len(delays))
	}
}

func TestRetryContextCancellation(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	config := &RetryConfig{
		MaxAttempts: 5,
		Delay:       100 * time.Millisecond,
		MaxDelay:    time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		RetryIf:     defaultRetryIf,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	resp := Get(srv.URL).DoWithRetry(ctx, config)

	if resp.Error() == nil {
		t.Error("want context cancellation error")
	}

	attemptCnt := atomic.LoadInt32(&attempts)
	if attemptCnt > 2 {
		t.Errorf("want at most 2 attempts before cancellation, got %d", attemptCnt)
	}
}

func TestBackoffFunctions(t *testing.T) {
	t.Run("ExponentialBackoff", func(t *testing.T) {
		backoff := ExponentialBackoff(100*time.Millisecond, 2.0, 1*time.Second)

		tests := []struct {
			attempt int
			want    time.Duration
		}{
			{0, 100 * time.Millisecond},
			{1, 200 * time.Millisecond},
			{2, 400 * time.Millisecond},
			{3, 800 * time.Millisecond},
			{4, 1 * time.Second}, // Should stay at max
			{5, 1 * time.Second},
		}

		for _, tt := range tests {
			delay := backoff(tt.attempt)
			if delay != tt.want {
				t.Errorf("attempt %d: want %d, got %d", tt.attempt, tt.want, delay)
			}
		}
	})

	t.Run("LinearBackoff", func(t *testing.T) {
		backoff := LinearBackoff(100*time.Millisecond, 50*time.Millisecond, 500*time.Millisecond)

		tests := []struct {
			attempt int
			want    time.Duration
		}{
			{0, 100 * time.Millisecond},
			{1, 150 * time.Millisecond},
			{2, 200 * time.Millisecond},
			{3, 250 * time.Millisecond},
			{8, 500 * time.Millisecond}, // Cap at max
		}

		for _, tt := range tests {
			delay := backoff(tt.attempt)
			if delay != tt.want {
				t.Errorf("attempt %d: want %d, got %d", tt.attempt, tt.want, delay)
			}
		}
	})

	t.Run("ConstantBackoff", func(t *testing.T) {
		backoff := ConstantBackoff(200 * time.Millisecond)

		for i := 0; i < 5; i++ {
			delay := backoff(i)
			if delay != 200*time.Millisecond {
				t.Errorf("attempt %d: want 200ms, got %d", i, delay)
			}
		}
	})
}

func TestRetryNetworkError(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt := atomic.AddInt32(&attempts, 1)
		if cnt < 3 {
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer srv.Close()

	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
		RetryIf:     defaultRetryIf,
	}

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryNoRetryOnSuccess(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer srv.Close()

	ctx := context.Background()
	config := DefaultRetryConfig()

	resp := Get(srv.URL).DoWithRetry(ctx, config)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("want 1 attempt, got %d", attempts)
	}
}
