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

	return NewOAuth("client-id", "client-secret", true,
		WithHTTPClient(srv.Client()),
		WithBaseURLs(srv.URL, srv.URL, srv.URL),
	)
}

// newBasicAuthTestClient returns a BasicAuthClient wired to srv instead of Viva's real hosts.
func newBasicAuthTestClient(t *testing.T, srv *httptest.Server) *BasicAuthClient {
	t.Helper()

	return NewBasicAuth("merchant-id", "api-key", true,
		WithHTTPClient(srv.Client()),
		WithBaseURLs(srv.URL, srv.URL, ""),
	)
}
