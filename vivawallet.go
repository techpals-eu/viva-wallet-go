package vivawallet

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	Demo         bool
	ClientID     string
	ClientSecret string
	MerchantID   string
	APIKey       string

	// ResellerID and ResellerAPIKey are the ISV partner's credentials, used by ISVClient
	// only. The merchant id an ISV acts for is a per-call argument, not part of Config.
	ResellerID     string
	ResellerAPIKey string

	// scope is the OAuth scope requested from the token endpoint. Empty means the token
	// request carries no scope, which is what the plain OAuth flow uses. ISV and ECR flows
	// need different scopes, so this is client state rather than a package constant.
	scope string

	// apiBase, appBase and authBase override the corresponding host when set, so tests can
	// point the SDK at an httptest.Server instead of Viva's real hosts. Left zero-valued in
	// normal use.
	apiBase  string
	appBase  string
	authBase string
}

type token struct {
	value   string
	expires time.Time
}

// tokenCache holds a client-credentials access token and guards it against concurrent
// access. It is embedded by every client that authenticates with a bearer token, so the
// token accessors are written once rather than per client.
type tokenCache struct {
	lock       sync.RWMutex
	tokenValue token

	// refreshLock is held across a token refresh so concurrent requests on an expired
	// token trigger one token request, not one each.
	refreshLock sync.Mutex
}

// AuthToken returns the token value.
func (t *tokenCache) AuthToken() string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tokenValue.value
}

// SetToken sets the token value and the expiration time of the token.
func (t *tokenCache) SetToken(value string, expires time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.tokenValue.value = value
	t.tokenValue.expires = expires
}

// HasAuthExpired returns true if the token has expired or expires within the next 30
// seconds, and false otherwise.
func (t *tokenCache) HasAuthExpired() bool {
	t.lock.RLock()
	expires := t.tokenValue.expires
	t.lock.RUnlock()

	return time.Now().Add(tokenRefreshSkew).After(expires)
}

// storeToken caches the token from a token response.
func (t *tokenCache) storeToken(resp *TokenResponse) {
	t.SetToken(resp.AccessToken, time.Now().Add(time.Second*time.Duration(resp.ExpiresIn)))
}

type OAuthClient struct {
	Config Config

	// Client performs every request this client makes. Prefer setting it at construction
	// with WithHTTPClient; mutating it afterwards races with in-flight requests.
	Client *http.Client

	tokenCache
}

type BasicAuthClient struct {
	Config Config

	// Client performs every request this client makes. Prefer setting it at construction
	// with WithHTTPClient; mutating it afterwards races with in-flight requests.
	Client *http.Client
}

// defaultTimeout is the timeout on the http.Client a client gets when neither
// WithHTTPClient nor WithTimeout is passed to its constructor.
const defaultTimeout = 60 * time.Second

type Client interface {
	Get(uri string, v interface{}) error
	Post(uri string, reader *bytes.Reader, v interface{}) error
	Patch(uri string, reader *bytes.Reader, v interface{}) error
}

// ApiUri returns the uri of the production or the demo api.
func ApiUri(c Config) string {
	if c.apiBase != "" {
		return c.apiBase
	}
	if isDemo(c) {
		return "https://demo-api.vivapayments.com"
	}
	return "https://api.vivapayments.com"
}

func AppUri(c Config) string {
	if c.appBase != "" {
		return c.appBase
	}
	if isDemo(c) {
		return "https://demo.vivapayments.com"
	}
	return "https://www.vivapayments.com"
}

func isDemo(c Config) bool {
	return c.Demo
}

func newRequest(method string, uri string, reader *bytes.Reader) (*http.Request, error) {
	var body io.Reader
	if reader != nil {
		body = reader
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build %s request for %s: %w", method, uri, err)
	}

	return req, nil
}
