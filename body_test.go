package rq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBodyString(t *testing.T) {
	wantBody := "Hello!"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if string(body) != wantBody {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Post(srv.URL).BodyString(wantBody).Do(ctx)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestBodyBytes(t *testing.T) {
	wantBody := []byte("Hello!")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if !bytes.Equal(body, wantBody) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Post(srv.URL).BodyBytes(wantBody).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	resp = BodyBytes(wantBody).Method(http.MethodPost).URL(srv.URL).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestBodyForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(r.Form)
	}))
	defer srv.Close()

	ctx := context.Background()

	formData := url.Values{
		"username": {"testuser"},
		"password": {"testpass"},
		"remember": {"true"},
	}

	resp := Post(srv.URL).BodyForm(formData).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want status 200, got %d", resp.StatusCode)
	}

	var result map[string][]string
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}

	for k, v := range formData {
		if got := result[k]; len(got) == 0 || got[0] != v[0] {
			t.Errorf("want form field %s=%s, got %v", k, v[0], got)
		}
	}
}

func TestBodyReader(t *testing.T) {
	wantBody := "Hello!"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if string(body) != wantBody {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()

	reader1 := strings.NewReader(wantBody)
	resp := Post(srv.URL).Body(reader1).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}

	reader2 := strings.NewReader(wantBody)
	resp = Body(reader2).Method(http.MethodPost).URL(srv.URL).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestBodyJSON(t *testing.T) {
	type TestUser struct {
		ID       int            `json:"id"`
		Name     string         `json:"name"`
		Active   bool           `json:"active"`
		Tags     []string       `json:"tags"`
		Metadata map[string]any `json:"metadata"`
	}

	wantUser := TestUser{
		ID:     123,
		Name:   "John Doe",
		Active: true,
		Tags:   []string{"test-tag1", "test-tag2"},
		Metadata: map[string]any{
			"metadataStr": "m",
			"metadataNum": 1,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid content type"))
			return
		}

		var user TestUser
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid JSON"))
			return
		}

		if user.ID != wantUser.ID || user.Name != wantUser.Name {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid JSON"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	ctx := context.Background()

	resp := Post(srv.URL).BodyJSON(wantUser).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		body, _ := resp.String()
		t.Errorf("want status 200, got %d: %s", resp.StatusCode, body)
	}

	var gotUser TestUser
	if err := resp.JSON(&gotUser); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if gotUser.ID != wantUser.ID {
		t.Errorf("want ID %d, got %d", wantUser.ID, gotUser.ID)
	}

	resp = BodyJSON(wantUser).Method(http.MethodPost).URL(srv.URL).Do(ctx)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want status 200, got %d", resp.StatusCode)
	}
}

func TestMustJSON(t *testing.T) {
	type TestUser struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/valid-user":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 123, "name": "John Doe"}`))
		case "/malformed-json":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 123, "name": "John Doe"`)) // Missing closing brace
		case "/type-mismatch":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "not-a-number", "name": "John Doe"}`))
		case "/empty":
			w.WriteHeader(http.StatusOK)
			// Empty response body
		case "/array":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[1, 2, 3, 4, 5]`))
		case "/map":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok", "message": "success"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx := context.Background()

	t.Run("successful decode", func(t *testing.T) {
		resp := Get(srv.URL + "/valid-user").Do(ctx)

		var user TestUser
		resp.MustJSON(&user)

		if user.ID != 123 {
			t.Errorf("want ID 123, got %d", user.ID)
		}
		if user.Name != "John Doe" {
			t.Errorf("want name 'John Doe', got %s", user.Name)
		}
	})

	t.Run("panics on malformed JSON", func(t *testing.T) {
		resp := Get(srv.URL + "/malformed-json").Do(ctx)

		var user TestUser
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustJSON should have panicked on malformed JSON")
			}
		}()

		resp.MustJSON(&user)
	})

	t.Run("panics on response error", func(t *testing.T) {
		resp := &Response{err: errors.New("network error")}

		var user TestUser

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustJSON should have panicked on response error")
			}
		}()

		resp.MustJSON(&user)
	})

	t.Run("panics on type mismatch", func(t *testing.T) {
		resp := Get(srv.URL + "/type-mismatch").Do(ctx)

		var user TestUser

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustJSON should have panicked on type mismatch")
			}
		}()

		resp.MustJSON(&user)
	})

	t.Run("panics on empty response", func(t *testing.T) {
		resp := Get(srv.URL + "/empty").Do(ctx)

		var user TestUser

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustJSON should have panicked on empty response")
			}
		}()

		resp.MustJSON(&user)
	})

	t.Run("works with different types", func(t *testing.T) {
		resp := Get(srv.URL + "/array").Do(ctx)

		var numbers []int
		resp.MustJSON(&numbers)

		want := []int{1, 2, 3, 4, 5}
		if len(numbers) != len(want) {
			t.Errorf("want length %d, got %d", len(want), len(numbers))
		}

		for i, v := range want {
			if numbers[i] != v {
				t.Errorf("want numbers[%d] = %d, got %d", i, v, numbers[i])
			}
		}
	})

	t.Run("works with map types", func(t *testing.T) {
		resp := Get(srv.URL + "/map").Do(ctx)

		var result map[string]string
		resp.MustJSON(&result)

		if result["status"] != "ok" {
			t.Errorf("want status 'ok', got %s", result["status"])
		}
		if result["message"] != "success" {
			t.Errorf("want message 'success', got %s", result["message"])
		}
	})
}
