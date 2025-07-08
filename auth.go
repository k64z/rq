package rq

import (
	"encoding/base64"
	"fmt"
)

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	Apply(r *Request) *Request
}

// Auth creates a new request with custom authorization
func Auth(authType, credentials string) *Request {
	return New().Auth(authType, credentials)
}

// Auth sets a custom authorization header
func (r *Request) Auth(authType, credentials string) *Request {
	if r.err != nil {
		return r
	}
	r.headers.Set("Authorization", authType+" "+credentials)
	return r
}

// WithAuth creates a new request with an AuthProvider
func WithAuth(provider AuthProvider) *Request {
	return New().WithAuth(provider)
}

// WithAuth applies an AuthProvider to the request
func (r *Request) WithAuth(provider AuthProvider) *Request {
	if r.err != nil {
		return r
	}
	return provider.Apply(r)
}

// BasicAuth creates a new request with basic authentication
func BasicAuth(username, password string) *Request {
	return New().BasicAuth(username, password)
}

// BasicAuth sets basic authentication
func (r *Request) BasicAuth(username, password string) *Request {
	if r.err != nil {
		return r
	}
	r.headers.Set("Authorization", "Basic "+basicAuth(username, password))
	return r
}

// BearerToken creates a new request with bearer token authentication
func BearerToken(token string) *Request {
	return New().BearerToken(token)
}

// BearerToken sets bearer token authentication
func (r *Request) BearerToken(token string) *Request {
	if r.err != nil {
		return r
	}
	r.headers.Set("Authorization", "Bearer "+token)
	return r
}

// basicAuth creates a basic auth string from username and password
func basicAuth(username, password string) string {
	auth := fmt.Sprintf("%s:%s", username, password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
