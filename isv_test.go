package vivawallet

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
)

// TestISVBasicAuthComposition pins the three-part basic-auth composition. Viva documents the
// ISV credentials as "Reseller ID + Merchant ID + Reseller API Key" without saying how they
// map onto the two basic-auth fields; this test is the single place the chosen mapping is
// asserted. What goes on the wire is the base64 of "username:password", so only the order of
// the three parts is observable; correcting it means changing isvBasicAuth and this test.
func TestISVBasicAuthComposition(t *testing.T) {
	cfg := Config{ResellerID: "reseller-1", ResellerAPIKey: "secret-key"}

	username, password := isvBasicAuth(cfg, "merchant-9")
	if username+":"+password != "reseller-1:merchant-9:secret-key" {
		t.Errorf("credential = %q, want %q", username+":"+password, "reseller-1:merchant-9:secret-key")
	}

	want := base64.StdEncoding.EncodeToString([]byte("reseller-1:merchant-9:secret-key"))
	if got := ISVBasicAuth(cfg, "merchant-9"); got != want {
		t.Errorf("ISVBasicAuth() = %q, want %q", got, want)
	}
}

func TestISVBasicAuthHeaderOnRequest(t *testing.T) {
	var gotHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"OrderCode": 1})
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	var out struct {
		OrderCode int `json:"OrderCode"`
	}
	if err := c.GetWithMerchant("merchant-9", srv.URL+"/api/orders/1", &out); err != nil {
		t.Fatalf("GetWithMerchant() error = %v", err)
	}

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("reseller-id:merchant-9:reseller-api-key"))
	if gotHeader != want {
		t.Errorf("Authorization header = %q, want %q", gotHeader, want)
	}
}

func TestISVBearerHeaderOnRequest(t *testing.T) {
	var sawBearer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/connect/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "isv-token", ExpiresIn: 3600})
		default:
			sawBearer = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]int{"orderCode": 7})
		}
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	if _, err := c.Authenticate(); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if c.AuthToken() != "isv-token" {
		t.Errorf("AuthToken() = %q, want %q", c.AuthToken(), "isv-token")
	}

	var out struct {
		OrderCode int `json:"orderCode"`
	}
	if err := c.Get(srv.URL+"/checkout/v2/isv/orders/7", &out); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if sawBearer != "Bearer isv-token" {
		t.Errorf("Authorization header = %q, want %q", sawBearer, "Bearer isv-token")
	}
}

// TestISVScopeIsSentOnTokenRequest asserts the scope reaches the token endpoint, and that a
// client without one sends no scope parameter at all.
func TestISVScopeIsSentOnTokenRequest(t *testing.T) {
	var gotScope string
	var hadScopeParam bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotScope = r.PostForm.Get("scope")
		_, hadScopeParam = r.PostForm["scope"]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "tok", ExpiresIn: 3600})
	}))
	defer srv.Close()

	scoped := newISVTestClient(t, srv, WithScope("urn:viva:payments:isv:api"))
	if _, err := scoped.Authenticate(); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if gotScope != "urn:viva:payments:isv:api" {
		t.Errorf("scope = %q, want %q", gotScope, "urn:viva:payments:isv:api")
	}

	hadScopeParam = false
	unscoped := newISVTestClient(t, srv)
	if _, err := unscoped.Authenticate(); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if hadScopeParam {
		t.Error("token request carried a scope parameter, want none when no scope is set")
	}
}

// TestISVClientServesManyMerchants is the property that makes one ISV client usable across
// goroutines: the merchant id travels with the call, never on the client, so concurrent
// requests for different merchants each carry their own credentials.
func TestISVClientServesManyMerchants(t *testing.T) {
	var mu sync.Mutex
	var seen []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, decodeBasic(t, r.Header.Get("Authorization")))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	merchants := []string{"merchant-a", "merchant-b", "merchant-c"}

	var wg sync.WaitGroup
	for _, m := range merchants {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(merchantID string) {
				defer wg.Done()
				if err := c.GetWithMerchant(merchantID, srv.URL+"/api/orders/1", nil); err != nil {
					t.Errorf("GetWithMerchant(%q) error = %v", merchantID, err)
				}
			}(m)
		}
	}
	wg.Wait()

	counts := map[string]int{}
	mu.Lock()
	for _, u := range seen {
		counts[u]++
	}
	mu.Unlock()

	var got []string
	for u := range counts {
		got = append(got, u)
		if counts[u] != 10 {
			t.Errorf("credential %q seen %d times, want 10", u, counts[u])
		}
	}
	sort.Strings(got)

	want := []string{
		"reseller-id:merchant-a:reseller-api-key",
		"reseller-id:merchant-b:reseller-api-key",
		"reseller-id:merchant-c:reseller-api-key",
	}
	if len(got) != len(want) {
		t.Fatalf("credentials = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("credentials = %v, want %v", got, want)
			break
		}
	}
}

// decodeBasic returns the decoded "username:password" of a Basic Authorization header. The
// ISV credential contains two colons, so http.Request.BasicAuth, which splits on the first
// one, cannot round-trip it.
func decodeBasic(t *testing.T, header string) string {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		t.Fatalf("decoding %q: %v", header, err)
	}
	return string(raw)
}

func TestISVAuthenticateTokenEndpointFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	if _, err := c.Authenticate(); err == nil {
		t.Fatal("Authenticate() error = nil, want error on non-200 response")
	}
}
