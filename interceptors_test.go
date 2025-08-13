package rq

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInterceptorTransport(t *testing.T) {
	var requestIntercepted bool
	var responseIntercepted bool

	transport := &InterceptorTransport{
		Base: http.DefaultTransport,
		RequestInterceptor: func(ctx context.Context, r *http.Request) error {
			requestIntercepted = true
			r.Header.Set("X-Intercepted-Request", "true")
			return nil
		},
		ResponseInterceptor: func(ctx context.Context, r *http.Response) error {
			responseIntercepted = true
			r.Header.Set("X-Intercepted-Response", "true")
			return nil
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Intercepted-Request") != "true" {
			t.Error("request interceptor not applied")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: transport}

	resp := Get(srv.URL).Client(client).Do()
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if !requestIntercepted {
		t.Error("request interceptor not called")
	}

	if !responseIntercepted {
		t.Error("response interceptor not called")
	}

	if resp.Header.Get("X-Intercepted-Response") != "true" {
		t.Error("response interceptor not applied")
	}
}

func TestRoundTripperFunc(t *testing.T) {
	var called bool

	roundTripper := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("OK")),
			Header:     make(http.Header),
		}, nil
	})

	client := &http.Client{Transport: roundTripper}

	resp := Get("https://example.com").Client(client).Do()
	if resp.Error() != nil {
		t.Fatal(resp.Error())
	}

	if !called {
		t.Error("RoundTripperFunc not called")
	}

	body, _ := resp.String()
	if body != "OK" {
		t.Errorf("want body OK, got %s", body)
	}
}

func TestDumpTransport(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello"}`))
	}))
	defer srv.Close()

	transport := DumpTransport(nil, logger)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "test-client")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	logOutput := buf.String()

	if !strings.Contains(logOutput, "=== HTTP REQUEST ===") {
		t.Error("want request dump header")
	}
	if !strings.Contains(logOutput, "GET") {
		t.Error("want GET method in request dump")
	}
	if !strings.Contains(logOutput, "User-Agent: test-client") {
		t.Error("want User-Agent header in request dump")
	}

	if !strings.Contains(logOutput, "=== HTTP RESPONSE ===") {
		t.Error("want response dump header")
	}
	if !strings.Contains(logOutput, "200 OK") {
		t.Error("want status in response dump")
	}
	if !strings.Contains(logOutput, "Content-Type: application/json") {
		t.Error("want Content-Type header in response dump")
	}
	if !strings.Contains(logOutput, `{"message":"hello"}`) {
		t.Error("want response body in dump")
	}
}

func TestDumpTransportWithCustomBase(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	customTransport := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		r.Header.Set("X-Custom", "added-by-transport")
		return http.DefaultTransport.RoundTrip(r)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "added-by-transport" {
			t.Error("custom transport header not found")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := DumpTransport(customTransport, logger)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	logOutput := buf.String()

	if !strings.Contains(logOutput, "X-Custom: added-by-transport") {
		t.Error("want custom header in request dump")
	}
}

func TestDumpTransportWithNilLogger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Should not panic with nil logger
	transport := DumpTransport(nil, nil)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestDumpTransportRequestError(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	errorTransport := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, http.ErrServerClosed
	})

	transport := DumpTransport(errorTransport, logger)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := client.Do(req)

	if !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("want ErrServerClosed, got %v", err)
	}

	logOutput := buf.String()
	// Should still dump the request even if transport fails
	if !strings.Contains(logOutput, "=== HTTP REQUEST ===") {
		t.Error("want request dump even on transport error")
	}
	// Should not have response dump
	if strings.Contains(logOutput, "=== HTTP RESPONSE ===") {
		t.Error("should not have response dump on transport error")
	}
}
