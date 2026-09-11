package jwks

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type Cache struct {
	url       string
	ttl       time.Duration
	client    *http.Client
	mu        sync.Mutex
	refreshMu sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

type document struct {
	Keys []key `json:"keys"`
}
type key struct {
	KID string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func New(url string, ttl, timeout time.Duration) *Cache {
	return &Cache{url: url, ttl: ttl, client: &http.Client{Timeout: timeout}}
}

func (cache *Cache) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	cache.mu.Lock()
	if time.Now().Before(cache.expiresAt) {
		key := cache.keys[kid]
		cache.mu.Unlock()
		if key == nil {
			return nil, fmt.Errorf("unknown signing key")
		}
		return key, nil
	}
	cache.mu.Unlock()
	if err := cache.refresh(ctx); err != nil {
		return nil, err
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	key := cache.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("unknown signing key")
	}
	return key, nil
}

func (cache *Cache) refresh(ctx context.Context) error {
	cache.refreshMu.Lock()
	defer cache.refreshMu.Unlock()
	cache.mu.Lock()
	if time.Now().Before(cache.expiresAt) {
		cache.mu.Unlock()
		return nil
	}
	cache.mu.Unlock()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, cache.url, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	response, err := cache.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: status %d", response.StatusCode)
	}
	var payload document
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode JWKS")
	}
	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, item := range payload.Keys {
		if item.Kty != "RSA" || item.KID == "" || item.N == "" || item.E == "" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil || len(eBytes) == 0 {
			continue
		}
		e := 0
		for _, value := range eBytes {
			e = e<<8 | int(value)
		}
		keys[item.KID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: e}
	}
	if len(keys) == 0 {
		return fmt.Errorf("JWKS contains no supported RSA keys")
	}
	cache.mu.Lock()
	cache.keys = keys
	cache.expiresAt = time.Now().Add(cache.ttl)
	cache.mu.Unlock()
	return nil
}
