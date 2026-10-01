package vivawallet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// tokenRefreshSkew is how long before its real expiry a token is treated as expired, so a
// request never races the server's clock with a token that is about to lapse.
const tokenRefreshSkew = 30 * time.Second

// authTransport attaches a bearer token to every request it carries, refreshing the token
// first when it has expired. It is how every bearer-token endpoint gets authenticated, so no
// endpoint method has to remember to check expiry itself.
//
// The token request must never go through this transport: it would recurse. Clients send it
// on their plain http.Client, and only API calls use the client bearerClient returns.
type authTransport struct {
	base         http.RoundTripper
	cache        *tokenCache
	authenticate func(ctx context.Context) (*TokenResponse, error)

	// hosts are the hosts that may receive the bearer token. A redirect to anywhere else is
	// sent without it, so the token cannot leak off Viva's hosts.
	hosts map[string]bool
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.hosts[req.URL.Host] {
		return t.base.RoundTrip(req)
	}

	if err := t.cache.refreshIfExpired(req.Context(), t.authenticate); err != nil {
		// A RoundTripper must close the request body even when it fails.
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, fmt.Errorf("viva: refresh access token: %w", err)
	}

	// RoundTrip must not modify the request it is given.
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.cache.AuthToken())

	return t.base.RoundTrip(r)
}

// refreshIfExpired fetches a new token when the cached one has expired. Concurrent callers
// on an expired token share one refresh: the first takes refreshLock and fetches, the rest
// wait for it and then find the token fresh on the second check.
func (t *tokenCache) refreshIfExpired(ctx context.Context, authenticate func(context.Context) (*TokenResponse, error)) error {
	if !t.HasAuthExpired() {
		return nil
	}

	t.refreshLock.Lock()
	defer t.refreshLock.Unlock()

	if !t.HasAuthExpired() {
		return nil
	}

	_, err := authenticate(ctx)
	return err
}

// bearerClient returns an http.Client that behaves like hc but authenticates every request
// to the configured Viva hosts with the token in cache. It is built per call rather than
// stored, so changes to the client's exported Client field are always honoured.
func bearerClient(hc *http.Client, c Config, cache *tokenCache, authenticate func(context.Context) (*TokenResponse, error)) *http.Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}

	base := hc.Transport
	if base == nil {
		base = http.DefaultTransport
	}

	return &http.Client{
		Transport: &authTransport{
			base:         base,
			cache:        cache,
			authenticate: authenticate,
			hosts:        hostsOf(ApiUri(c), AppUri(c)),
		},
		CheckRedirect: hc.CheckRedirect,
		Jar:           hc.Jar,
		Timeout:       hc.Timeout,
	}
}

func hostsOf(uris ...string) map[string]bool {
	hosts := make(map[string]bool, len(uris))
	for _, uri := range uris {
		if u, err := url.Parse(uri); err == nil {
			hosts[u.Host] = true
		}
	}
	return hosts
}
