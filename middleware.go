package rq

import (
	"log"
	"time"
)

// Middleware modifies a request before it is sent
type Middleware func(*Request) *Request

// Use creates a new request with middleware applied
func Use(middleware ...Middleware) *Request {
	return New().Use(middleware...)
}

// Use applies middleware to the request
func (r *Request) Use(middleware ...Middleware) *Request {
	for _, m := range middleware {
		r = m(r)
	}
	return r
}

// Chain combines multiple middleware into one
func Chain(middleware ...Middleware) Middleware {
	return func(r *Request) *Request {
		for _, m := range middleware {
			r = m(r)
		}
		return r
	}
}

// LoggingMiddleware logs request details
func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(r *Request) *Request {
		if logger != nil {
			logger.Printf("%s %s", r.method, r.url)
		}
		return r
	}
}

// UserAgentMiddleware sets a custom User-Agent header
func UserAgentMiddleware(userAgent string) Middleware {
	return func(r *Request) *Request {
		return r.Header("User-Agent", userAgent)
	}
}

// TimeoutMiddleware sets a timeout for the request
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(r *Request) *Request {
		return r.Timeout(timeout)
	}
}

// HeadersMiddleware sets a timeout for the request
func HeadersMiddleware(headers map[string]string) Middleware {
	return func(r *Request) *Request {
		return r.Headers(headers)
	}
}
