package rq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBasicRequests(t *testing.T) {
	tests := map[string]struct {
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		"GET request": {
			method:   http.MethodGet,
			path:     "/get",
			wantCode: http.StatusOK,
			wantBody: "GET OK",
		},
		"POST request": {
			method:   http.MethodPost,
			path:     "/post",
			wantCode: http.StatusCreated,
			wantBody: "POST OK",
		},
		"PUT request": {
			method:   http.MethodPut,
			path:     "/put",
			wantCode: http.StatusOK,
			wantBody: "PUT OK",
		},
		"DELETE request": {
			method:   http.MethodDelete,
			path:     "/delete",
			wantCode: http.StatusNoContent,
			wantBody: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("want method %s, got %s", tt.method, r.Method)
				}

				if r.URL.Path != tt.path {
					t.Errorf("want path %s, got %s", tt.path, r.URL.Path)
				}

				w.WriteHeader(tt.wantCode)
				if tt.wantBody != "" {
					w.Write([]byte(tt.wantBody))
				}
			}))
			defer srv.Close()

			resp := Method(tt.method).URL(srv.URL + tt.path).Do()

			if resp.StatusCode != tt.wantCode {
				t.Errorf("want status %d, got %d", tt.wantCode, resp.StatusCode)
			}

			if tt.wantBody != "" {
				body, err := resp.String()
				if err != nil {
					t.Fatal(err)
				}

				if body != tt.wantBody {
					t.Errorf("want body %q, got %q", tt.wantBody, body)
				}
			}
		})
	}
}

func TestQueryParameters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(query)
	}))
	defer srv.Close()

	resp := Get(srv.URL).
		QueryParam("page", "1").
		QueryParam("limit", "10").
		QueryParam("sort", "name").
		QueryParams(map[string]string{
			"filter": "active",
			"lang":   "en",
		}).
		Do()

	var result map[string][]string
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"page":   {"1"},
		"limit":  {"10"},
		"sort":   {"name"},
		"filter": {"active"},
		"lang":   {"en"},
	}

	for k, v := range want {
		if got := result[k]; len(got) == 0 || got[0] != v[0] {
			t.Errorf("want query param %s=%s, got %v", k, v[0], got)
		}
	}
}

func TestHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := make(map[string]string)
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(headers)
	}))
	defer srv.Close()

	resp := Get(srv.URL).
		Header("X-Custom-Header", "value1").
		Header("X-Another-Header", "value2").
		Headers(map[string]string{
			"X-Third-Header":  "value3",
			"X-Fourth-Header": "value4",
		}).
		Do()

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"X-Custom-Header":  "value1",
		"X-Another-Header": "value2",
		"X-Third-Header":   "value3",
		"X-Fourth-Header":  "value4",
	}

	for k, v := range want {
		if got := result[k]; got != v {
			t.Errorf("want header %s=%s, got %s", k, v, got)
		}
	}
}

func TestCookies(t *testing.T) {
	t.Run("single cookie", func(t *testing.T) {
		cookie := &http.Cookie{Name: "session", Value: "abc123"}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("session")
			if err != nil || c.Value != "abc123" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := Get(ts.URL).Cookies(cookie).Do()
		if resp.Error() != nil {
			t.Fatal(resp.Error())
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("multiple cookies", func(t *testing.T) {
		cookies := []*http.Cookie{
			{Name: "session", Value: "abc123"},
			{Name: "lang", Value: "en"},
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil || session.Value != "abc123" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			lang, err := r.Cookie("lang")
			if err != nil || lang.Value != "en" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := Get(ts.URL).Cookies(cookies...).Do()
		if resp.Error() != nil {
			t.Fatal(resp.Error())
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("chained cookies", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := r.Cookie("session")
			if err != nil || session.Value != "abc123" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			lang, err := r.Cookie("lang")
			if err != nil || lang.Value != "en" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := Get(ts.URL).
			Cookies(&http.Cookie{Name: "session", Value: "abc123"}).
			Cookies(&http.Cookie{Name: "lang", Value: "en"}).
			Do()
		if resp.Error() != nil {
			t.Fatal(resp.Error())
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("no cookies", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.Cookies()) != 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := Get(ts.URL).Do()
		if resp.Error() != nil {
			t.Fatal(resp.Error())
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
	})
}

func TestJSONBody(t *testing.T) {
	type TestData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("want Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		var data TestData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	testData := TestData{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	resp := Post(srv.URL).BodyJSON(testData).Do()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want status 200, got %d", resp.StatusCode)
	}

	var result TestData
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}

	if result != testData {
		t.Errorf("want %+v, got %+v", testData, result)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp := Get(srv.URL).Timeout(50 * time.Millisecond).Do()
	if resp.Error() == nil {
		t.Error("want timeout error, got nil")
	}

	resp = Get(srv.URL).Timeout(200 * time.Millisecond).Do()
	if resp.Error() != nil {
		t.Errorf("want no error, got %v", resp.Error())
	}
}

func TestErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/404":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer srv.Close()

	resp := Get(srv.URL + "/404").Do()
	if !resp.IsError() {
		t.Error("want IsError() to be true for 404")
	}
	if resp.IsOK() {
		t.Error("want IsOK() to be false for 404")
	}

	err := resp.ExpectOK()
	if err == nil {
		t.Error("want ExpectOK to return error for 404")
	}

	err = resp.ExpectStatus(http.StatusNotFound)
	if err != nil {
		t.Errorf("want ExpectStatus(404) to return nil, got %v", err)
	}

	// TODO: implement AsHTTPError
	// resp = Get(srv.URL+"/500").Do(ctx)
	// httpErr := resp.AsHTTPError()
	// if httpErr == nil {
	// 	t.Errorf("want AsHTTPError to return error for 500")
	// }
}

func TestMustDoContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Run("successful request", func(t *testing.T) {
		ctx := context.Background()
		resp := Get(srv.URL).MustDoContext(ctx)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("panics on context timeout", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustDoContext should have panicked on timeout")
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		Get(srv.URL).MustDoContext(ctx)
	})

	t.Run("panics on invalid URL", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustDoContext should have panicked on invalid URL")
			}
		}()

		ctx := context.Background()
		Get("invalid-url").MustDoContext(ctx)
	})
}

func TestMustDo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer srv.Close()

	t.Run("successful request", func(t *testing.T) {
		resp := Get(srv.URL).MustDo()

		body, err := resp.String()
		if err != nil {
			t.Fatal(err)
		}

		if body != "Success" {
			t.Errorf("want body 'Success', got %s", body)
		}
	})

	t.Run("panics on network error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustDo should have panicked on invalid URL")
			}
		}()

		Get("invalid-url").MustDo()
	})
}
