package rq

import (
	"context"
	"net/http"
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
