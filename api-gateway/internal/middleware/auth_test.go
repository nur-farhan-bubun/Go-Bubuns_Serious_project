package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
)

// ─── Test Helpers ───────────────────────────────────────────────────────

// testLogger returns a logger that discards output during tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testConfig returns a minimal config with a test HMAC secret.
func testConfig(secret string) *config.Config {
	return &config.Config{
		ClerkJWTKey:    secret,
		ClerkJWTIssuer: "test-issuer",
	}
}

// generateHS256Token creates a signed HS256 JWT with the given claims.
func generateHS256Token(secret string, claims map[string]interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	return s
}

// generateRS256KeyPair generates an RSA key pair and returns the
// PEM-encoded public key and the private key for signing.
func generateRS256KeyPair() (pemPublic string, privateKey *rsa.PrivateKey, err error) {
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", nil, err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})

	return string(pemBytes), privateKey, nil
}

// generateRS256Token creates a signed RS256 JWT with the given claims.
func generateRS256Token(privateKey *rsa.PrivateKey, claims map[string]interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	s, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	return s
}

// echoContext creates an Echo HTTP test context.
func echoContext(method, path string, headers map[string]string, queryParams map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if len(queryParams) > 0 {
		q := url.Values{}
		for k, v := range queryParams {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	return ctx, rec
}

// ─── safeTokenPrefix Tests ──────────────────────────────────────────────

func TestSafeTokenPrefix_LongToken(t *testing.T) {
	result := safeTokenPrefix("abcdefghijklmnop")
	if result != "abcdef..." {
		t.Errorf("got %q, want %q", result, "abcdef...")
	}
}

func TestSafeTokenPrefix_ShortToken(t *testing.T) {
	result := safeTokenPrefix("abc")
	if result != "abc" {
		t.Errorf("got %q, want %q", result, "abc")
	}
}

func TestSafeTokenPrefix_Empty(t *testing.T) {
	result := safeTokenPrefix("")
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

// ─── parseRSAPublicKey Tests ───────────────────────────────────────────

func TestParseRSAPublicKey_Valid(t *testing.T) {
	pemPub, _, err := generateRS256KeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	key, err := parseRSAPublicKey(pemPub)
	if err != nil {
		t.Fatalf("parseRSAPublicKey: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
}

func TestParseRSAPublicKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPublicKey("not a pem block")
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestParseRSAPublicKey_EmptyString(t *testing.T) {
	_, err := parseRSAPublicKey("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseRSAPublicKey_WrongKeyType(t *testing.T) {
	// Encode a private key as "PUBLIC KEY" — should fail parsing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	privASN1 := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: privASN1,
	})

	_, err = parseRSAPublicKey(string(pemBytes))
	if err == nil {
		t.Fatal("expected error for private key data in public key block")
	}
}

// ─── resolveKey Tests ──────────────────────────────────────────────────

func TestResolveKey_HMAC(t *testing.T) {
	cfg := testConfig("my-hmac-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-1"})

	key, err := resolveKey(token, cfg)
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}

	keyBytes, ok := key.([]byte)
	if !ok {
		t.Fatal("expected []byte key for HMAC")
	}
	if string(keyBytes) != "my-hmac-secret" {
		t.Errorf("got %q, want %q", string(keyBytes), "my-hmac-secret")
	}
}

func TestResolveKey_RSA(t *testing.T) {
	pemPub, _, err := generateRS256KeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	cfg := testConfig(pemPub)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "user-1"})

	key, err := resolveKey(token, cfg)
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}

	if _, ok := key.(*rsa.PublicKey); !ok {
		t.Fatal("expected *rsa.PublicKey for RS256")
	}
}

func TestResolveKey_EmptyKey(t *testing.T) {
	cfg := testConfig("")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-1"})

	_, err := resolveKey(token, cfg)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should mention 'not configured', got: %v", err)
	}
}

func TestResolveKey_UnsupportedMethod(t *testing.T) {
	cfg := testConfig("some-secret")
	// ES256 (ECDSA) is not supported
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": "user-1"})

	_, err := resolveKey(token, cfg)
	if err == nil {
		t.Fatal("expected error for unsupported method")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported', got: %v", err)
	}
}

// ─── parseAndValidateToken Tests ────────────────────────────────────────

func TestParseAndValidateToken_HS256_Valid(t *testing.T) {
	cfg := testConfig("test-secret-12345")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret-12345", claims)

	parsed, err := parseAndValidateToken(tokenStr, cfg)
	if err != nil {
		t.Fatalf("parseAndValidateToken: %v", err)
	}
	if parsed["sub"] != "user-42" {
		t.Errorf("sub = %v, want %v", parsed["sub"], "user-42")
	}
}

func TestParseAndValidateToken_HS256_WrongIssuer(t *testing.T) {
	cfg := testConfig("test-secret-12345")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "wrong-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret-12345", claims)

	_, err := parseAndValidateToken(tokenStr, cfg)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestParseAndValidateToken_HS256_Expired(t *testing.T) {
	cfg := testConfig("test-secret-12345")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(-time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret-12345", claims)

	_, err := parseAndValidateToken(tokenStr, cfg)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseAndValidateToken_HS256_BadSignature(t *testing.T) {
	cfg := testConfig("correct-secret")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("wrong-secret", claims)

	_, err := parseAndValidateToken(tokenStr, cfg)
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestParseAndValidateToken_HS256_NoSubject(t *testing.T) {
	cfg := testConfig("test-secret-12345")
	claims := jwt.MapClaims{
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret-12345", claims)

	parsed, err := parseAndValidateToken(tokenStr, cfg)
	if err != nil {
		t.Fatalf("parseAndValidateToken: %v", err)
	}
	if parsed["sub"] != nil {
		t.Errorf("expected no sub claim, got %v", parsed["sub"])
	}
}

func TestParseAndValidateToken_InvalidToken(t *testing.T) {
	cfg := testConfig("test-secret-12345")
	_, err := parseAndValidateToken("not-a-jwt-token", cfg)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestParseAndValidateToken_RS256_Valid(t *testing.T) {
	pemPub, privKey, err := generateRS256KeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	cfg := testConfig(pemPub)
	claims := jwt.MapClaims{
		"sub": "user-rsa-99",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateRS256Token(privKey, claims)

	parsed, err := parseAndValidateToken(tokenStr, cfg)
	if err != nil {
		t.Fatalf("parseAndValidateToken: %v", err)
	}
	if parsed["sub"] != "user-rsa-99" {
		t.Errorf("sub = %v, want %v", parsed["sub"], "user-rsa-99")
	}
}

func TestParseAndValidateToken_RS256_BadSignature(t *testing.T) {
	// Generate two key pairs — sign with one private key, verify with the other's public key
	pemPub1, _, err := generateRS256KeyPair()
	if err != nil {
		t.Fatalf("generate key 1: %v", err)
	}
	_, privKey2, err := generateRS256KeyPair()
	if err != nil {
		t.Fatalf("generate key 2: %v", err)
	}

	cfg := testConfig(pemPub1)
	claims := jwt.MapClaims{
		"sub": "user-rsa-99",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	// Sign with privKey2 (doesn't match pemPub1) — should fail
	tokenStr := generateRS256Token(privKey2, claims)

	_, err = parseAndValidateToken(tokenStr, cfg)
	if err == nil {
		t.Fatal("expected error for RS256 token signed with different key")
	}
}

func TestParseAndValidateToken_HS256_NoIssuerRequired(t *testing.T) {
	// When ClerJWIssuer is empty, issuer validation should be skipped.
	cfg := &config.Config{
		ClerkJWTKey:    "test-secret",
		ClerkJWTIssuer: "", // no issuer configured
	}
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "some-other-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret", claims)

	parsed, err := parseAndValidateToken(tokenStr, cfg)
	if err != nil {
		t.Fatalf("parseAndValidateToken: %v", err)
	}
	if parsed["sub"] != "user-42" {
		t.Errorf("sub = %v, want %v", parsed["sub"], "user-42")
	}
}

// ─── AuthMiddleware Tests ──────────────────────────────────────────────

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := testConfig("test-secret")
	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", nil, nil)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_BadFormat(t *testing.T) {
	cfg := testConfig("test-secret")
	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", map[string]string{
		"Authorization": "NotBearer token123",
	}, nil)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := testConfig("test-secret")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret", claims)

	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", map[string]string{
		"Authorization": "Bearer " + tokenStr,
	}, nil)

	var capturedUserID string
	handler := mw(func(c echo.Context) error {
		capturedUserID = c.Request().Header.Get("X-User-ID")
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedUserID != "user-42" {
		t.Errorf("X-User-ID = %q, want %q", capturedUserID, "user-42")
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := testConfig("test-secret")
	claims := jwt.MapClaims{
		"sub": "user-42",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(-time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret", claims)

	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", map[string]string{
		"Authorization": "Bearer " + tokenStr,
	}, nil)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ExtractUserID(t *testing.T) {
	cfg := testConfig("test-secret")
	claims := jwt.MapClaims{
		"sub": "user-99",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret", claims)

	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", map[string]string{
		"Authorization": "Bearer " + tokenStr,
	}, nil)

	handler := mw(func(c echo.Context) error {
		uid, ok := ExtractUserID(c)
		if !ok {
			t.Error("ExtractUserID returned false")
		}
		if uid != "user-99" {
			t.Errorf("ExtractUserID = %q, want %q", uid, "user-99")
		}
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Dev Secret End-to-End Validation Tests ───────────────────────────
// These tests prove that a JWT signed with the same DEV_JWT_SECRET used
// by the frontend's loginSimulated() will be accepted by the gateway's
// auth middleware. This validates the end-to-end dev auth flow.

const devJWTSecret = "dev-jwt-secret-do-not-use-in-production"

func TestParseAndValidateToken_DevSecret_Valid(t *testing.T) {
	cfg := testConfig(devJWTSecret)
	// No issuer configured — matches frontend which doesn't require iss validation
	cfg.ClerkJWTIssuer = ""

	claims := jwt.MapClaims{
		"sub":   "user-sim-001",
		"email": "alice@example.com",
		"name":  "Alice",
		"iat":   float64(time.Now().Unix()),
		"exp":   float64(time.Now().Add(24 * time.Hour).Unix()),
		"iss":   "dev-simulated",
	}
	tokenStr := generateHS256Token(devJWTSecret, claims)

	parsed, err := parseAndValidateToken(tokenStr, cfg)
	if err != nil {
		t.Fatalf("parseAndValidateToken with dev secret: %v", err)
	}
	if parsed["sub"] != "user-sim-001" {
		t.Errorf("sub = %v, want %v", parsed["sub"], "user-sim-001")
	}
	if parsed["email"] != "alice@example.com" {
		t.Errorf("email = %v, want %v", parsed["email"], "alice@example.com")
	}
}

func TestParseAndValidateToken_DevSecret_WrongSecret(t *testing.T) {
	// Token signed with dev secret but verified with wrong secret
	cfg := testConfig("some-other-secret")

	claims := jwt.MapClaims{
		"sub": "user-sim-001",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token(devJWTSecret, claims)

	_, err := parseAndValidateToken(tokenStr, cfg)
	if err == nil {
		t.Fatal("expected error for token signed with different secret")
	}
}

func TestAuthMiddleware_DevSecret_ValidToken(t *testing.T) {
	cfg := testConfig(devJWTSecret)
	cfg.ClerkJWTIssuer = ""

	claims := jwt.MapClaims{
		"sub":   "user-sim-001",
		"email": "alice@example.com",
		"name":  "Alice",
		"iat":   float64(time.Now().Unix()),
		"exp":   float64(time.Now().Add(24 * time.Hour).Unix()),
		"iss":   "dev-simulated",
	}
	tokenStr := generateHS256Token(devJWTSecret, claims)

	mw := AuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/v1/users", map[string]string{
		"Authorization": "Bearer " + tokenStr,
	}, nil)

	var capturedUserID string
	handler := mw(func(c echo.Context) error {
		capturedUserID = c.Request().Header.Get("X-User-ID")
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedUserID != "user-sim-001" {
		t.Errorf("X-User-ID = %q, want %q", capturedUserID, "user-sim-001")
	}
}

func TestWebSocketAuthMiddleware_DevSecret_ValidToken(t *testing.T) {
	cfg := testConfig(devJWTSecret)
	cfg.ClerkJWTIssuer = ""

	claims := jwt.MapClaims{
		"sub":   "user-sim-001",
		"email": "alice@example.com",
		"name":  "Alice",
		"iat":   float64(time.Now().Unix()),
		"exp":   float64(time.Now().Add(24 * time.Hour).Unix()),
		"iss":   "dev-simulated",
	}
	tokenStr := generateHS256Token(devJWTSecret, claims)

	mw := WebSocketAuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/ws", nil, map[string]string{
		"token": tokenStr,
	})

	handler := mw(func(c echo.Context) error {
		actualUserID := c.Request().URL.Query().Get("user_id")
		if actualUserID != "user-sim-001" {
			t.Errorf("user_id = %q, want %q", actualUserID, "user-sim-001")
		}
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── WebSocketAuthMiddleware Tests ─────────────────────────────────────

func TestWebSocketAuthMiddleware_MissingToken(t *testing.T) {
	cfg := testConfig("test-secret")
	mw := WebSocketAuthMiddleware(cfg, testLogger())
	// No token query param
	ctx, rec := echoContext(http.MethodGet, "/ws", nil, nil)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWebSocketAuthMiddleware_ValidToken(t *testing.T) {
	cfg := testConfig("test-secret")
	claims := jwt.MapClaims{
		"sub": "user-ws-007",
		"iss": "test-issuer",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	tokenStr := generateHS256Token("test-secret", claims)

	mw := WebSocketAuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/ws", nil, map[string]string{
		"token":   tokenStr,
		"user_id": "should-be-overridden",
	})

	handler := mw(func(c echo.Context) error {
		// Verify user_id was overridden by the middleware
		// Note: Echo caches query params, so we read from the raw URL directly
		actualUserID := c.Request().URL.Query().Get("user_id")
		if actualUserID != "user-ws-007" {
			t.Errorf("user_id = %q, want %q", actualUserID, "user-ws-007")
		}
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWebSocketAuthMiddleware_InvalidToken(t *testing.T) {
	cfg := testConfig("test-secret")
	mw := WebSocketAuthMiddleware(cfg, testLogger())
	ctx, rec := echoContext(http.MethodGet, "/ws", nil, map[string]string{
		"token": "invalid-token-string",
	})

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
