package rq

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "[TEST] ", 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	middleware := LoggingMiddleware(logger)

	resp := Get(srv.URL).Use(middleware).Do(ctx)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "[TEST]") {
		t.Error("want log prefix")
	}
	if !strings.Contains(logOutput, "GET") {
		t.Error("want method in log")
	}
	if !strings.Contains(logOutput, srv.URL) {
		t.Error("want URL in log")
	}
}

func TestUserAgentMiddleware(t *testing.T) {
	wantUA := "Test/1.0"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != wantUA {
			t.Errorf("want User-Agent %q, got %q", wantUA, ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	middleware := UserAgentMiddleware(wantUA)

	resp := Get(srv.URL).Use(middleware).Do(ctx)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	middleware := TimeoutMiddleware(50 * time.Millisecond)

	resp := Get(srv.URL).Use(middleware).Do(ctx)
	if resp.Error() == nil {
		t.Error("want timeout error")
	}
}

func TestHeadersMiddleware(t *testing.T) {
	headers := map[string]string{
		"X-App-Name":    "TestApp",
		"X-App-Version": "1.0.0",
		"X-Request-ID":  "123456",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			if got := r.Header.Get(k); got != v {
				t.Errorf("want header %s=%s, got %s", k, v, got)
			}
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	middleware := HeadersMiddleware(headers)

	resp := Get(srv.URL).Use(middleware).Do(ctx)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}
}

func TestChainMiddleware(t *testing.T) {
	var executionOrder []string

	midlleware1 := func(r *Request) *Request {
		executionOrder = append(executionOrder, "middleware1")
		return r.Header("X-Middleware-1", "true")
	}

	midlleware2 := func(r *Request) *Request {
		executionOrder = append(executionOrder, "middleware2")
		return r.Header("X-Middleware-2", "true")
	}

	midlleware3 := func(r *Request) *Request {
		executionOrder = append(executionOrder, "middleware3")
		return r.Header("X-Middleware-3", "true")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 1; i <= 3; i++ {
			header := fmt.Sprintf("X-Middleware-%d", i)
			if r.Header.Get(header) != "true" {
				t.Errorf("want header %s to be true", header)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	chain := Chain(midlleware1, midlleware2, midlleware3)

	resp := Get(srv.URL).Use(chain).Do(ctx)
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	want := []string{"middleware1", "middleware2", "middleware3"}

	for i, name := range want {
		if executionOrder[i] != name {
			t.Errorf("want middleware %s at pos %d, got %s", name, i, executionOrder[i])
		}
	}
}

func TestUseMethodStarting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Error("want custom header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Use(func(r *Request) *Request {
		return r.Header("X-Custom", "value")
	}).URL(srv.URL).Do(ctx)

	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}
}
