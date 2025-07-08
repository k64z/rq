package rq

import (
	"context"
	"io"
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
	ctx := context.Background()

	resp := Get(srv.URL).Client(client).Do(ctx)
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
	ctx := context.Background()

	resp := Get("https://example.com").Client(client).Do(ctx)
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
