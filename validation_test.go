package rq_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k64z/rq"
)

func TestStatusCodeValidator(t *testing.T) {
	tests := map[string]struct {
		serverStatus int
		wantStatus   int
		wantErr      bool
	}{
		"valid status": {
			serverStatus: http.StatusOK,
			wantStatus:   http.StatusOK,
			wantErr:      false,
		},
		"invalid status": {
			serverStatus: http.StatusBadRequest,
			wantStatus:   http.StatusOK,
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer ts.Close()

			resp := rq.New().
				URL(ts.URL).
				Validate(rq.Validate.StatusCode(tt.wantStatus)).
				Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestOKValidator(t *testing.T) {
	tests := map[string]struct {
		serverStatus int
		wantErr      bool
	}{
		"200 OK": {
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		"201 Created": {
			serverStatus: http.StatusCreated,
			wantErr:      false,
		},
		"400 Bad Request": {
			serverStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		"500 Internal Server Error": {
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer ts.Close()

			resp := rq.New().URL(ts.URL).Validate(rq.Validate.OK()).Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestHeaderValidator(t *testing.T) {
	tests := map[string]struct {
		serverHeader map[string]string
		wantKey      string
		wantValue    string
		wantErr      bool
	}{
		"header matches": {
			serverHeader: map[string]string{"X-Custom": "value"},
			wantKey:      "X-Custom",
			wantValue:    "value",
			wantErr:      false,
		},
		"header doesn't match": {
			serverHeader: map[string]string{"X-Custom": "wrong"},
			wantKey:      "X-Custom",
			wantValue:    "value",
			wantErr:      true,
		},
		"header missing": {
			serverHeader: map[string]string{},
			wantKey:      "X-Custom",
			wantValue:    "value",
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.serverHeader {
					w.Header().Set(k, v)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			resp := rq.New().
				URL(ts.URL).
				Validate(rq.Validate.Header(tt.wantKey, tt.wantValue)).
				Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestHeaderExistsValidator(t *testing.T) {
	tests := map[string]struct {
		serverHeader map[string]string
		wantKey      string
		wantErr      bool
	}{
		"header exists": {
			serverHeader: map[string]string{"X-Custom": "value"},
			wantKey:      "X-Custom",
			wantErr:      false,
		},
		"header missing": {
			serverHeader: map[string]string{},
			wantKey:      "X-Custom",
			wantErr:      true,
		},
		"header with empty value": {
			serverHeader: map[string]string{"X-Custom": ""},
			wantKey:      "X-Custom",
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.serverHeader {
					w.Header().Set(k, v)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			resp := rq.New().
				URL(ts.URL).
				Validate(rq.Validate.HeaderExists(tt.wantKey)).
				Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestBodyContainsValidator(t *testing.T) {
	tests := map[string]struct {
		body      string
		substring string
		wantErr   bool
	}{
		"contains substring": {
			body:      "hello world",
			substring: "world",
			wantErr:   false,
		},
		"doesn't contain substring": {
			body:      "hello world",
			substring: "foo",
			wantErr:   true,
		},
		"empty body": {
			body:      "",
			substring: "test",
			wantErr:   true,
		},
		"empty substring": {
			body:      "hello",
			substring: "",
			wantErr:   false, // empty string is always found
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			resp := rq.New().
				URL(ts.URL).
				Validate(rq.Validate.BodyContains(tt.substring)).
				Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestBodyMatchesValidator(t *testing.T) {
	tests := map[string]struct {
		body    string
		pattern string
		wantErr bool
	}{
		"matches pattern": {
			body:    "hello123",
			pattern: `hello\d+`,
			wantErr: false,
		},
		"doesn't match pattern": {
			body:    "hello",
			pattern: `\d+`,
			wantErr: true,
		},
		"invalid pattern": {
			body:    "hello",
			pattern: `[`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			resp := rq.New().
				URL(ts.URL).
				Validate(rq.Validate.BodyMatches(tt.pattern)).
				Do()

			if tt.wantErr && resp.Error() == nil {
				t.Error("want validation error, got nil")
			}
			if !tt.wantErr && resp.Error() != nil {
				t.Errorf("want no error, got %v", resp.Error())
			}
		})
	}
}

func TestAllValidator(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer ts.Close()

	t.Run("all validators pass", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.All(
				rq.Validate.OK(),
				rq.Validate.Header("Content-Type", "application/json"),
				rq.Validate.BodyContains("ok"),
			)).
			Do()
		if resp.Error() != nil {
			t.Errorf("want no error, got %v", resp.Error())
		}
	})

	t.Run("one validator fails", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.All(
				rq.Validate.OK(),
				rq.Validate.Header("Content-Type", "text/html"),
				rq.Validate.BodyContains("ok"),
			)).
			Do()
		if resp.Error() == nil {
			t.Error("want validation error, got nil")
		}
	})
}

func TestAnyValidator(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer ts.Close()

	t.Run("at least one validator passes", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.Any(
				rq.Validate.StatusCode(200),
				rq.Validate.Header("Content-Type", "text/html"),
			)).
			Do()
		if resp.Error() != nil {
			t.Errorf("want no error, got %v", resp.Error())
		}
	})

	t.Run("all validators fail", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.Any(
				rq.Validate.StatusCode(404),
				rq.Validate.Header("Content-Type", "text/html"),
				rq.Validate.BodyContains("notfound"),
			)).
			Do()
		if resp.Error() == nil {
			t.Error("want validation error, got nil")
		}

		errMsg := resp.Error().Error()
		if !strings.Contains(errMsg, "all validators failed") {
			t.Errorf("want error message to mention all validators failed, got %q", errMsg)
		}
	})
}

func TestNotValidator(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer ts.Close()

	t.Run("inverts success", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.Not(rq.Validate.StatusCode(404))).
			Do()
		if resp.Error() != nil {
			t.Errorf("want no error, got %v", resp.Error())
		}
	})

	t.Run("inverts failure", func(t *testing.T) {
		resp := rq.New().
			URL(ts.URL).
			Validate(rq.Validate.Not(rq.Validate.OK())).
			Do()
		if resp.Error() == nil {
			t.Error("want validation error, got nil")
		}
	})
}

func TestValidationFailureStopsEarly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{'"message": "error"}`))
	}))
	defer ts.Close()

	var customValidatorCalled bool
	customValidator := func(r *rq.Response) error {
		customValidatorCalled = true
		return nil
	}

	resp := rq.New().
		URL(ts.URL).
		Validate(rq.Validate.OK(), customValidator).
		Do()
	if resp.Error() == nil {
		t.Error("want validation error, got nil")
	}

	if customValidatorCalled {
		t.Error("custom validator should not have been called after earlier validation failure")
	}
}
