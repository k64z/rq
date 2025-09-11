package rq

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

// RequestInterceptor allows inspection/modification of http.Request
type RequestInterceptor func(context.Context, *http.Request) error

// ResponseInterceptor allows inspection/modification if http.Response
type ResponseInterceptor func(context.Context, *http.Response) error

// RoundTripperFunc is an adapter to allow functions to be used as RoundTrippers
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface
func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// InterceptorTransport wraps an http.RoundTripper with interceptors
type InterceptorTransport struct {
	Base                http.RoundTripper
	RequestInterceptor  RequestInterceptor
	ResponseInterceptor ResponseInterceptor
}

// RoundTrip implements the RoundTripper interface with interceptor support
func (t *InterceptorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.RequestInterceptor != nil {
		if err := t.RequestInterceptor(req.Context(), req); err != nil {
			return nil, err
		}
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if t.ResponseInterceptor != nil {
		if err := t.ResponseInterceptor(req.Context(), resp); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
	}

	return resp, nil
}

// DumpTransport creates a transport that dumps requests and responses
func DumpTransport(base http.RoundTripper, logger *log.Logger) *InterceptorTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if logger == nil {
		logger = log.New(os.Stdout, "[HTTP] ", log.LstdFlags)
	}

	dumpWrapper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Preserve the original body by reading it into memory
		var bodyBytes []byte
		var err error

		if req.Body != nil {
			bodyBytes, err = io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("read request body: %w", err)
			}
			req.Body.Close()

			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Making the actual request may modify headers and consume body
		resp, err := base.RoundTrip(req)

		// Restore the body again for dumping the modified request
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Dump the request regardless of success or failure
		dump, dumpErr := httputil.DumpRequestOut(req, true)
		if dumpErr != nil {
			logger.Printf("Failed to dump request: %v", dumpErr)
		} else {
			logger.Printf("=== HTTP REQUEST ===\n%s\n=====================", string(dump))
		}

		return resp, err
	})

	return &InterceptorTransport{
		Base: dumpWrapper,
		ResponseInterceptor: func(ctx context.Context, resp *http.Response) error {
			dump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				logger.Printf("Failed to dump response: %v", err)
				return nil
			}

			logger.Printf("=== HTTP RESPONSE ===\n%s\n======================", string(dump))
			return nil
		},
	}
}
