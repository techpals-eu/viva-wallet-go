package vivawallet

import (
	"net/http"
	"time"
)

// options collects the settings applied by the Option functions passed to NewOAuth and
// NewBasicAuth. It is not exported: callers configure a client through Option values only.
type options struct {
	httpClient *http.Client
	timeout    time.Duration

	apiBase  string
	appBase  string
	authBase string

	scope string
}

// Option customises a client at construction time. Options are applied in the order they
// are given, so a later one wins over an earlier one setting the same thing.
type Option func(*options)

// WithHTTPClient makes the client use hc for every request. This is the supported way to
// set a custom transport, connection pool, proxy or a wrapping RoundTripper for metrics,
// tracing or retries.
//
// The client's own timeout is left untouched, so WithTimeout has no effect once an
// http.Client is supplied — set hc.Timeout instead. Passing nil is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(o *options) {
		if hc == nil {
			return
		}
		o.httpClient = hc
	}
}

// WithTimeout overrides the default 60s timeout of the http.Client the SDK creates for
// this client. It is ignored when WithHTTPClient supplies a client of your own. A
// non-positive duration is ignored.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		if d <= 0 {
			return
		}
		o.timeout = d
	}
}

// WithBaseURLs overrides the hosts the SDK talks to, which is what lets a test point the
// SDK at an httptest.Server. Empty strings leave the corresponding host at its default.
//
//	api  — ApiUri:  /checkout/v2 and /acquiring/v1 endpoints
//	app  — AppUri:  /api/* endpoints
//	auth — the OAuth token endpoint; unused by BasicAuthClient
func WithBaseURLs(api, app, auth string) Option {
	return func(o *options) {
		if api != "" {
			o.apiBase = api
		}
		if app != "" {
			o.appBase = app
		}
		if auth != "" {
			o.authBase = auth
		}
	}
}

// WithScope sets the OAuth scope requested from the token endpoint. Viva's ISV and EFT POS
// flows are documented as using scoped tokens; the plain payments flow needs no scope, which
// is the default. An empty string leaves the token request without a scope parameter.
func WithScope(scope string) Option {
	return func(o *options) {
		o.scope = scope
	}
}

// newOptions applies opts on top of the SDK defaults.
func newOptions(opts ...Option) options {
	o := options{timeout: defaultTimeout}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&o)
	}
	return o
}

// applyTo copies the base URL overrides onto c.
func (o options) applyTo(c *Config) {
	c.apiBase = o.apiBase
	c.appBase = o.appBase
	c.authBase = o.authBase
	c.scope = o.scope
}

// client returns the http.Client to use, defaulting to a fresh one per SDK client so two
// clients in the same process never share mutable state.
func (o options) client() *http.Client {
	if o.httpClient != nil {
		return o.httpClient
	}
	return &http.Client{Timeout: o.timeout}
}
