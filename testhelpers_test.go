package vivawallet

import (
	"net/http/httptest"
	"testing"
	"time"
)

// futureExpiry returns a token expiry far enough in the future that HasAuthExpired is false.
func futureExpiry() time.Time {
	return time.Now().Add(time.Hour)
}

// newOAuthTestClient returns an OAuthClient wired to srv instead of Viva's real hosts.
func newOAuthTestClient(t *testing.T, srv *httptest.Server) *OAuthClient {
	t.Helper()

	c := NewOAuth("client-id", "client-secret", true)
	c.Client = srv.Client()
	c.Config.apiBase = srv.URL
	c.Config.appBase = srv.URL
	c.Config.authBase = srv.URL

	return c
}

// newBasicAuthTestClient returns a BasicAuthClient wired to srv instead of Viva's real hosts.
func newBasicAuthTestClient(t *testing.T, srv *httptest.Server) *BasicAuthClient {
	t.Helper()

	c := NewBasicAuth("merchant-id", "api-key", true)
	c.Client = srv.Client()
	c.Config.apiBase = srv.URL
	c.Config.appBase = srv.URL

	return c
}
