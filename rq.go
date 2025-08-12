package rq

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Request represents an HTTP request configuration
type Request struct {
	client      *http.Client
	method      string
	url         string
	headers     http.Header
	queryParams url.Values
	body        io.Reader
	timeout     time.Duration
	err         error
}

// Response wraps http.Response with additional convenience methods
type Response struct {
	*http.Response
	body []byte
	err  error
}

// New creates a new HTTP request with default settings
func New() *Request {
	return &Request{
		client:      defaultClient,
		method:      http.MethodGet,
		headers:     make(http.Header),
		queryParams: make(url.Values),
	}
}

// Get creates a new GET request
func Get(urlStr string) *Request {
	return New().Method(http.MethodGet).URL(urlStr)
}

// Post creates a new POST request
func Post(urlStr string) *Request {
	return New().Method(http.MethodPost).URL(urlStr)
}

// Put creates a new PUT request
func Put(urlStr string) *Request {
	return New().Method(http.MethodPut).URL(urlStr)
}

// Delete creates a new DELETE request
func Delete(urlStr string) *Request {
	return New().Method(http.MethodDelete).URL(urlStr)
}

// Patch creates a new PATCH request
func Patch(urlStr string) *Request {
	return New().Method(http.MethodPatch).URL(urlStr)
}

// Head creates a new HEAD request
func Head(urlStr string) *Request {
	return New().Method(http.MethodHead).URL(urlStr)
}

// Method creates a new request with the specified HTTP method
func Method(method string) *Request {
	return New().Method(method)
}

// Method sets the HTTP method
func (r *Request) Method(method string) *Request {
	if r.err != nil {
		return r
	}
	r.method = method
	return r
}

// URL creates a new request with the specified URL
func URL(urlStr string) *Request {
	return New().URL(urlStr)
}

// URL sets the request URL
func (r *Request) URL(urlStr string) *Request {
	if r.err != nil {
		return r
	}
	r.url = urlStr
	return r
}

// Client creates a new request with a custom HTTP client
func Client(client *http.Client) *Request {
	return New().Client(client)
}

// Client sets a custom HTTP client
func (r *Request) Client(client *http.Client) *Request {
	if r.err != nil {
		return r
	}
	r.client = client
	return r
}

// Timeout creates a new request with a timeout
func Timeout(timeout time.Duration) *Request {
	return New().Timeout(timeout)
}

// Timeout sets the request timeout
func (r *Request) Timeout(timeout time.Duration) *Request {
	if r.err != nil {
		return r
	}
	r.timeout = timeout
	return r
}

// Header creates a new request with a header
func Header(key, value string) *Request {
	return New().Header(key, value)
}

// Header adds a header to the request
func (r *Request) Header(key, value string) *Request {
	if r.err != nil {
		return r
	}
	r.headers.Add(key, value)
	return r
}

// Headers creates a new request with multiple headers
func Headers(headers map[string]string) *Request {
	return New().Headers(headers)
}

// Headers sets multiple headers at once
func (r *Request) Headers(headers map[string]string) *Request {
	if r.err != nil {
		return r
	}
	for k, v := range headers {
		r.headers.Set(k, v)
	}
	return r
}

// QueryParam creates a new request with a query parameter
func QueryParam(key, value string) *Request {
	return New().QueryParam(key, value)
}

// QueryParam adds a query parameter
func (r *Request) QueryParam(key, value string) *Request {
	if r.err != nil {
		return r
	}
	r.queryParams.Add(key, value)
	return r
}

// QueryParams sets multiple query parameters
func (r *Request) QueryParams(params map[string]string) *Request {
	if r.err != nil {
		return r
	}
	for k, v := range params {
		r.queryParams.Set(k, v)
	}
	return r
}

// QueryParams creates a new request with multiple query parameters
func QueryParams(params map[string]string) *Request {
	return New().QueryParams(params)
}

// DoContext executes the request and returns a Response
func (r *Request) DoContext(ctx context.Context) *Response {
	if r.err != nil {
		return &Response{err: r.err}
	}

	u, err := url.Parse(r.url)
	if err != nil {
		return &Response{err: fmt.Errorf("invalid URL: %q: %w", r.url, err)}
	}

	if len(r.queryParams) > 0 {
		u.RawQuery = r.queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, r.method, u.String(), r.body)
	if err != nil {
		return &Response{err: fmt.Errorf("failed to create request: %w", err)}
	}

	req.Header = r.headers.Clone()

	client := r.client

	if r.timeout > 0 {
		client = &http.Client{
			Timeout:   r.timeout,
			Transport: r.client.Transport,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &Response{err: fmt.Errorf("request failed: %w", err)}
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return &Response{
			Response: resp,
			err:      fmt.Errorf("failed to read body: %w", err),
		}
	}

	return &Response{
		Response: resp,
		body:     body,
	}
}

// Do executes the request with background context and returns a Response
func (r *Request) Do() *Response {
	return r.DoContext(context.Background())
}

// Error returns any error that occurred
func (r *Response) Error() error {
	return r.err
}

// IsOK returns true if status code is 2xx
func (r *Response) IsOK() bool {
	if r.err != nil || r.Response == nil {
		return false
	}
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError returns true if status code is 4xx or 5xx
func (r *Response) IsError() bool { // TODO: shouldn't it be HasError instead?
	if r.err != nil || r.Response == nil {
		return true
	}
	return r.StatusCode >= 400
}

// ExpectStatus returns an error if the status code doesn't match
func (r *Response) ExpectStatus(status int) error {
	if r.err != nil {
		return r.err
	}

	if r.StatusCode != status {
		return fmt.Errorf("expected status %d, got %d", status, r.StatusCode)
	}

	return nil
}

// ExpectOK return an error if the status is not 2xx
func (r *Response) ExpectOK() error {
	if r.err != nil {
		return r.err
	}

	if !r.IsOK() {
		return fmt.Errorf("expected 2xx status, got %d", r.StatusCode)
	}

	return nil
}
