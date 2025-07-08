package rq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkSimpleGET(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		resp := Get(srv.URL).Do(ctx)
		if resp.Error() != nil {
			b.Fatal(resp.Error())
		}
	}
}

// TODO: Need more benchmarks

func BenchmarkComparison(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	ctx := context.Background()

	b.Run("rq", func(b *testing.B) {
		for b.Loop() {
			resp := Get(srv.URL).Header("Accept", "application/json").Do(ctx)

			var result map[string]string
			if err := resp.JSON(&result); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("stdlib", func(b *testing.B) {
		client := &http.Client{}

		for b.Loop() {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, http.NoBody)
			if err != nil {
				b.Fatal(err)
			}

			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}

			var result map[string]string
			err = json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

}
