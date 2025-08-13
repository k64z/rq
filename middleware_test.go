package rq

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
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

	middleware := LoggingMiddleware(logger)

	resp := Get(srv.URL).Use(middleware).Do()
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

	middleware := UserAgentMiddleware(wantUA)

	resp := Get(srv.URL).Use(middleware).Do()
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

	middleware := TimeoutMiddleware(50 * time.Millisecond)

	resp := Get(srv.URL).Use(middleware).Do()
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

	middleware := HeadersMiddleware(headers)

	resp := Get(srv.URL).Use(middleware).Do()
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

	chain := Chain(midlleware1, midlleware2, midlleware3)

	resp := Get(srv.URL).Use(chain).Do()
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

func TestDumpMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "[DUMP] ", 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))
	defer srv.Close()

	middleware := DumpMiddleware(logger)

	resp := Get(srv.URL).
		Header("X-Request-Header", "test-header").
		Use(middleware).
		Do()

	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "=== HTTP REQUEST ===") {
		t.Error("want request dump header")
	}
	if !strings.Contains(logOutput, "GET") {
		t.Error("want GET method in request dump")
	}
	if !strings.Contains(logOutput, "X-Request-Header: test-header") {
		t.Error("want request header in dump")
	}

	if !strings.Contains(logOutput, "=== HTTP RESPONSE ===") {
		t.Error("want response dump header")
	}
	if !strings.Contains(logOutput, "X-Response-Header: test-value") {
		t.Error("want response header in dump")
	}
	if !strings.Contains(logOutput, "response body") {
		t.Error("want response body in dump")
	}
}

func TestDumpMiddlewarePreservesClientSettings(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "[DUMP] ", 0)

	cookieServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/set-cookie" {
			http.SetCookie(w, &http.Cookie{Name: "test", Value: "value"})
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/check-cookie" {
			cookie, err := r.Cookie("test")
			if err != nil || cookie.Value != "value" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer cookieServer.Close()

	middleware := DumpMiddleware(logger)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	resp1 := Get(cookieServer.URL + "/set-cookie").
		Client(client).
		Use(middleware).
		Do()

	if resp1.Error() != nil {
		t.Fatal(resp1.Error())
	}

	resp2 := Get(cookieServer.URL + "/check-cookie").
		Client(client).
		Use(middleware).
		Do()

	if resp2.Error() != nil {
		t.Fatal(resp2.Error())
	}

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d - cookie jar not preserved", resp2.StatusCode)
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

	resp := Use(func(r *Request) *Request {
		return r.Header("X-Custom", "value")
	}).URL(srv.URL).Do()

	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}
}
