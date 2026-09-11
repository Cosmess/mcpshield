package jwks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheRefreshesOnceForConcurrentKeyRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_ = json.NewEncoder(writer).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "key-1", "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	}))
	defer server.Close()
	cache := New(server.URL, time.Minute, time.Second)
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := cache.Key(context.Background(), "key-1"); err != nil {
				t.Errorf("Key() error = %v", err)
			}
		}()
	}
	group.Wait()
	if got := requests.Load(); got != 1 {
		t.Fatalf("JWKS requests = %d, want 1", got)
	}
}

func TestCacheRefreshesAfterTTL(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_ = json.NewEncoder(writer).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "key-1", "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	}))
	defer server.Close()
	cache := New(server.URL, time.Millisecond, time.Second)
	if _, err := cache.Key(context.Background(), "key-1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := cache.Key(context.Background(), "key-1"); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("JWKS requests = %d, want 2", requests.Load())
	}
}
