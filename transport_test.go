package vivawallet

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// tokenServer serves the token endpoint and GET /checkout/v2/transactions/{id}, counting
// token requests and recording any bearer header the token request carried.
type tokenServer struct {
	*httptest.Server

	tokenRequests int32

	mu               sync.Mutex
	tokenAuthHeaders []string
	apiAuthHeaders   []string
}

func newTokenServer(t *testing.T) *tokenServer {
	t.Helper()

	ts := &tokenServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		defer ts.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/connect/token":
			atomic.AddInt32(&ts.tokenRequests, 1)
			ts.tokenAuthHeaders = append(ts.tokenAuthHeaders, r.Header.Get("Authorization"))
			// Slow the refresh down so concurrent callers pile up behind it.
			time.Sleep(20 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "fresh-token", ExpiresIn: 3600})
		case "/checkout/v2/transactions/trx-1":
			ts.apiAuthHeaders = append(ts.apiAuthHeaders, r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(GetTransactionResponse{OrderCode: 42})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(ts.Close)

	return ts
}

// TestTokenRequestCarriesNoBearer asserts the refresh goes out on the plain client: basic
// auth with the client credentials, never the bearer token the transport attaches.
func TestTokenRequestCarriesNoBearer(t *testing.T) {
	ts := newTokenServer(t)
	c := newOAuthTestClient(t, ts.Server)
	c.SetToken("stale-token", time.Now().Add(-time.Hour))

	if _, err := c.GetTransaction("trx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}

	if len(ts.tokenAuthHeaders) != 1 {
		t.Fatalf("token endpoint hit %d times, want 1", len(ts.tokenAuthHeaders))
	}
	if got := ts.tokenAuthHeaders[0]; got != "Basic "+AuthBody(c.Config) {
		t.Errorf("token request Authorization = %q, want client-credentials basic auth", got)
	}
}

// TestConcurrentRefreshIsSingleFlight is the property refreshLock exists for: many requests
// arriving on one expired token must share a single token request.
func TestConcurrentRefreshIsSingleFlight(t *testing.T) {
	ts := newTokenServer(t)
	c := newOAuthTestClient(t, ts.Server)
	c.SetToken("stale-token", time.Now().Add(-time.Hour))

	const calls = 50
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.GetTransaction("trx-1"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("GetTransaction() error = %v", err)
	}
	if n := atomic.LoadInt32(&ts.tokenRequests); n != 1 {
		t.Errorf("token endpoint hit %d times, want 1", n)
	}
	for _, got := range ts.apiAuthHeaders {
		if got != "Bearer fresh-token" {
			t.Fatalf("API request Authorization = %q, want %q", got, "Bearer fresh-token")
		}
	}
}

// TestTokenAboutToExpireIsRefreshed covers the refresh skew: a token with seconds left is
// refreshed rather than sent and allowed to lapse in flight.
func TestTokenAboutToExpireIsRefreshed(t *testing.T) {
	ts := newTokenServer(t)
	c := newOAuthTestClient(t, ts.Server)
	c.SetToken("nearly-stale-token", time.Now().Add(10*time.Second))

	if _, err := c.GetTransaction("trx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}

	if n := atomic.LoadInt32(&ts.tokenRequests); n != 1 {
		t.Errorf("token endpoint hit %d times, want 1", n)
	}
	if got := ts.apiAuthHeaders[0]; got != "Bearer fresh-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer fresh-token")
	}
}

func TestValidTokenIsNotRefreshed(t *testing.T) {
	ts := newTokenServer(t)
	c := newOAuthTestClient(t, ts.Server)
	c.SetToken("cached-token", futureExpiry())

	if _, err := c.GetTransaction("trx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}

	if n := atomic.LoadInt32(&ts.tokenRequests); n != 0 {
		t.Errorf("token endpoint hit %d times, want 0", n)
	}
	if got := ts.apiAuthHeaders[0]; got != "Bearer cached-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer cached-token")
	}
}

// TestRefreshFailureSkipsRequest asserts a failed refresh surfaces as an error carrying the
// token endpoint's status, and the API request is never sent with a stale token.
func TestRefreshFailureSkipsRequest(t *testing.T) {
	var apiHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		atomic.AddInt32(&apiHits, 1)
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)
	c.SetToken("stale-token", time.Now().Add(-time.Hour))

	_, err := c.GetTransaction("trx-1")
	if err == nil {
		t.Fatal("GetTransaction() error = nil, want refresh failure")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("error = %v, want an *APIError with status 401", err)
	}
	if n := atomic.LoadInt32(&apiHits); n != 0 {
		t.Errorf("API hit %d times after a failed refresh, want 0", n)
	}
}

// TestRoundTripDoesNotMutateRequest asserts the transport sets the bearer on a clone.
func TestRoundTripDoesNotMutateRequest(t *testing.T) {
	ts := newTokenServer(t)
	c := newOAuthTestClient(t, ts.Server)
	c.SetToken("cached-token", futureExpiry())

	req, err := http.NewRequest("GET", ts.URL+"/checkout/v2/transactions/trx-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := bearerClient(c.Client, c.Config, &c.tokenCache, c.authenticate).Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	resp.Body.Close()

	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("caller's request was mutated: Authorization = %q", got)
	}
}

// TestBearerNotSentOffVivaHosts asserts a redirect to a host other than the configured Viva
// hosts does not carry the token.
func TestBearerNotSentOffVivaHosts(t *testing.T) {
	var sawAuth string
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GetTransactionResponse{})
	}))
	defer elsewhere.Close()

	viva := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/landing", http.StatusFound)
	}))
	defer viva.Close()

	c := newOAuthTestClient(t, viva)
	c.SetToken("secret-token", futureExpiry())

	if _, err := c.GetTransaction("trx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}
	if sawAuth != "" {
		t.Errorf("redirect target received Authorization = %q, want none", sawAuth)
	}
}

// TestISVBearerRefreshesAndBasicAuthDoesNot asserts the ISV client refreshes on its bearer
// calls, while its basic-auth calls neither trigger a refresh nor carry a bearer token.
func TestISVBearerRefreshesAndBasicAuthDoesNot(t *testing.T) {
	var tokenRequests int32
	var mu sync.Mutex
	seen := map[string]string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/connect/token" {
			atomic.AddInt32(&tokenRequests, 1)
			_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "isv-token", ExpiresIn: 3600})
			return
		}
		mu.Lock()
		seen[r.URL.Path] = r.Header.Get("Authorization")
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	var out map[string]interface{}
	if err := c.GetWithMerchant("merchant-1", srv.URL+"/api/orders/1", &out); err != nil {
		t.Fatalf("GetWithMerchant() error = %v", err)
	}
	if n := atomic.LoadInt32(&tokenRequests); n != 0 {
		t.Errorf("basic-auth call hit the token endpoint %d times, want 0", n)
	}
	if got := seen["/api/orders/1"]; got != "Basic "+ISVBasicAuth(c.Config, "merchant-1") {
		t.Errorf("basic-auth call Authorization = %q, want ISV basic auth", got)
	}

	if err := c.Get(srv.URL+"/checkout/v2/isv/orders/1", &out); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if n := atomic.LoadInt32(&tokenRequests); n != 1 {
		t.Errorf("token endpoint hit %d times, want 1", n)
	}
	if got := seen["/checkout/v2/isv/orders/1"]; got != "Bearer isv-token" {
		t.Errorf("bearer call Authorization = %q, want %q", got, "Bearer isv-token")
	}
}
