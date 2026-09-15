package vivawallet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAuthenticateSetsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connect/token" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "client-id" || password != "client-secret" {
			t.Errorf("unexpected basic auth: %q/%q (ok=%v)", username, password, ok)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "tok-1",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		})
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)

	resp, err := c.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if resp.AccessToken != "tok-1" {
		t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "tok-1")
	}
	if c.AuthToken() != "tok-1" {
		t.Errorf("AuthToken() = %q, want %q", c.AuthToken(), "tok-1")
	}
	if c.HasAuthExpired() {
		t.Error("HasAuthExpired() = true, want false right after authenticating")
	}
}

func TestAuthenticateTokenEndpointFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)

	if _, err := c.Authenticate(); err == nil {
		t.Fatal("Authenticate() error = nil, want error on non-200 response")
	}
}

// TestExpiredTokenIsRefreshedOnRequest is the regression test for the auth-refresh bug: an
// expired token must be silently refreshed and the request retried with the new bearer token,
// rather than the call returning an error on a successful refresh.
func TestExpiredTokenIsRefreshedOnRequest(t *testing.T) {
	var tokenRequests int
	var sawBearer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/connect/token":
			tokenRequests++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TokenResponse{
				AccessToken: "fresh-token",
				ExpiresIn:   3600,
			})
		case "/checkout/v2/transactions/trx-1":
			sawBearer = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GetTransactionResponse{OrderCode: 42})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)
	c.SetToken("stale-token", time.Now().Add(-time.Hour))

	trx, err := c.GetTransaction("trx-1")
	if err != nil {
		t.Fatalf("GetTransaction() error = %v, want nil after a successful token refresh", err)
	}
	if trx.OrderCode != 42 {
		t.Errorf("OrderCode = %d, want 42", trx.OrderCode)
	}
	if tokenRequests != 1 {
		t.Errorf("token endpoint hit %d times, want 1", tokenRequests)
	}
	if sawBearer != "Bearer fresh-token" {
		t.Errorf("Authorization header = %q, want %q", sawBearer, "Bearer fresh-token")
	}
}

func TestAuthTokenConcurrency(t *testing.T) {
	c := &OAuthClient{}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.SetToken("tok", time.Now().Add(time.Duration(i)*time.Second))
		}(i)
		go func() {
			defer wg.Done()
			_ = c.AuthToken()
			_ = c.HasAuthExpired()
		}()
	}
	wg.Wait()
}
