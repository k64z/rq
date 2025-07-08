package rq

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	wantUser := "testuser"
	wantPass := "testpass"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Basic" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		creds := strings.Split(string(decoded), ":")
		if len(creds) != 2 || creds[0] != wantUser || creds[1] != wantPass {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Get(srv.URL).BasicAuth(wantUser, wantPass).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	resp = Get(srv.URL).BasicAuth("wrong", "creds").Do(ctx)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want status 401, got %d", resp.StatusCode)
	}
}

func TestBearerToken(t *testing.T) {
	wantToken := "test-token-12345"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		want := "Bearer " + wantToken

		if auth != want {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid token"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Get(srv.URL).BearerToken(wantToken).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestCustomAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if auth != "Custom custom-test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Get(srv.URL).Auth("Custom", "custom-test-token").Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestCustomAuthProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Auth") != "custom-value" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Header.Get("X-Request-ID") != "12345" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	customProvider := customAuthProvider{
		headers: map[string]string{
			"X-Custom-Auth": "custom-value",
			"X-Request-ID":  "12345",
		},
	}

	ctx := context.Background()

	resp := Get(srv.URL).WithAuth(customProvider).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

type customAuthProvider struct {
	headers map[string]string
}

func (p customAuthProvider) Apply(r *Request) *Request {
	for k, v := range p.headers {
		r = r.Header(k, v)
	}
	return r
}
