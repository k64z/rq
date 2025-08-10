package rq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

// Body creates a new request with a body from an io.Reader
func Body(body io.Reader) *Request {
	return New().Body(body)
}

// Body sets the request body from an io.Reader
func (r *Request) Body(body io.Reader) *Request {
	if r.err != nil {
		return r
	}
	r.body = body
	return r
}

// BodyString creates a new request with a string body
func BodyString(body string) *Request {
	return New().BodyString(body)
}

// BodyString sets the request body from a string
func (r *Request) BodyString(body string) *Request {
	if r.err != nil {
		return r
	}
	r.body = strings.NewReader(body)
	return r
}

// BodyBytes creates a new request with a byte slice body
func BodyBytes(body []byte) *Request {
	return New().BodyBytes(body)
}

// BodyBytes sets the request body from bytes
func (r *Request) BodyBytes(body []byte) *Request {
	if r.err != nil {
		return r
	}
	r.body = bytes.NewReader(body)
	return r
}

// BodyJSON creates a new request with a JSON body
func BodyJSON(v any) *Request {
	return New().BodyJSON(v)
}

// BodyJSON sets the request body as JSON
func (r *Request) BodyJSON(v any) *Request {
	if r.err != nil {
		return r
	}

	data, err := json.Marshal(v)
	if err != nil {
		r.err = fmt.Errorf("failed to marshal JSON: %w", err)
		return r
	}

	r.body = bytes.NewReader(data)
	r.headers.Set("Content-Type", "application/json")
	return r
}

// BodyForm creates a new request with form data
func BodyForm(data url.Values) *Request {
	return New().BodyForm(data)
}

// BodyForm sets the request body as form data
func (r *Request) BodyForm(data url.Values) *Request {
	if r.err != nil {
		return r
	}

	r.body = strings.NewReader(data.Encode())
	r.headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// Bytes returns the response body as bytes
func (r *Response) Bytes() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.body, nil
}

// String returns the response body as string
func (r *Response) String() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return string(r.body), nil
}

// JSON decodes the response body as JSON
func (r *Response) JSON(v any) error {
	if r.err != nil {
		return r.err
	}

	if err := json.Unmarshal(r.body, v); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	return nil
}

// MustJSON decodes the response body as JSON, panicking on error
// This is useful for cases where you want fail fast on JSON decode errors
func (r *Response) MustJSON(v any) {
	if err := r.JSON(v); err != nil {
		panic(err)
	}
}

// BodyReader return an io.Reader for the response body
func (r *Response) BodyReader() (io.Reader, error) {
	if r.err != nil {
		return nil, r.err
	}
	return bytes.NewReader(r.body), nil
}

// SaveToFile saves the response body to a file
func (r *Response) SaveToFile(filename string) error {
	if r.err != nil {
		return r.err
	}

	return os.WriteFile(filename, r.body, 0o600)
}
