package sentinel

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
	"github.com/golang-jwt/jwt/v5"
)

const signingKeyRefetchFloor = time.Minute

var signingKeys = struct {
	sync.RWMutex
	keys         map[string]*rsa.PublicKey
	lastFetch    time.Time
	lastError    error
	refreshMutex sync.Mutex
}{}

func InitializeSigningKeys() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return refreshSigningKeys(ctx, true)
}

func ValidateToken(token string) (map[string]interface{}, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (any, error) {
			kid, _ := token.Header["kid"].(string)
			return signingKey(kid)
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(config.SentinelClientID),
	)
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("token is invalid")
	}
	return map[string]interface{}(claims), nil
}

func signingKey(kid string) (*rsa.PublicKey, error) {
	if kid == "" {
		return nil, errors.New("token key id is missing")
	}
	if key := cachedSigningKey(kid); key != nil {
		return key, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := refreshSigningKeys(ctx, false); err != nil {
		return nil, err
	}
	if key := cachedSigningKey(kid); key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("no signing key matches kid %q", kid)
}

func cachedSigningKey(kid string) *rsa.PublicKey {
	signingKeys.RLock()
	defer signingKeys.RUnlock()
	return signingKeys.keys[kid]
}

func refreshSigningKeys(ctx context.Context, force bool) error {
	signingKeys.refreshMutex.Lock()
	defer signingKeys.refreshMutex.Unlock()

	signingKeys.RLock()
	lastFetch := signingKeys.lastFetch
	lastError := signingKeys.lastError
	signingKeys.RUnlock()
	if !force && !lastFetch.IsZero() && time.Since(lastFetch) < signingKeyRefetchFloor {
		return lastError
	}

	keys, err := fetchSigningKeys(ctx)
	signingKeys.Lock()
	signingKeys.lastFetch = time.Now()
	signingKeys.lastError = err
	if err == nil {
		signingKeys.keys = keys
	}
	signingKeys.Unlock()
	return err
}

func fetchSigningKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	if strings.TrimSpace(config.SentinelURL) == "" {
		return nil, errors.New("SENTINEL_URL is not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(config.SentinelURL, "/")+"/api/core/keys", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch Sentinel JWKS: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Sentinel JWKS: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch Sentinel JWKS: HTTP %d", response.StatusCode)
	}
	var document struct {
		Keys []struct {
			KeyType   string `json:"kty"`
			Use       string `json:"use"`
			Algorithm string `json:"alg"`
			ID        string `json:"kid"`
			Modulus   string `json:"n"`
			Exponent  string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("decode Sentinel JWKS: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, encoded := range document.Keys {
		if encoded.KeyType != "RSA" || encoded.Algorithm != "RS256" || encoded.Use != "sig" || encoded.ID == "" {
			continue
		}
		key, err := decodeRSAKey(encoded.Modulus, encoded.Exponent)
		if err != nil {
			return nil, fmt.Errorf("decode Sentinel signing key %q: %w", encoded.ID, err)
		}
		keys[encoded.ID] = key
	}
	if len(keys) == 0 {
		return nil, errors.New("Sentinel JWKS contains no RS256 signing keys")
	}
	return keys, nil
}

func decodeRSAKey(modulus string, exponent string) (*rsa.PublicKey, error) {
	n, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil || len(n) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	e, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(e) == 0 || len(e) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}
	exponentValue := new(big.Int).SetBytes(e)
	if !exponentValue.IsInt64() || exponentValue.Int64() < 2 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponentValue.Int64())}, nil
}
