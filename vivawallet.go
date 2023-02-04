package vivawallet

import (
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
}

type token struct {
	value   string
	expires time.Time
}

type OAuthClient struct {
	Config     Config
	HTTPClient *http.Client
	lock       sync.RWMutex
	tokenValue *token
}

type BasicAuthClient struct {
	Config     Config
	HTTPClient *http.Client
}

// defaultHTTPTimeout is the default timeout on the http.Client used by the library.
const defaultTimeout = 60 * time.Second

var httpClient = &http.Client{
	Timeout: defaultTimeout,
}

// New creates a new viva client for the oauth apis
func NewOAuth(clientID string, clientSecret string, demo bool) *OAuthClient {
	return &OAuthClient{
		Config: Config{
			Demo:         demo,
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		HTTPClient: httpClient,
		tokenValue: &token{},
	}
}

// New creates a new viva client for the basic auth apis
func NewBasicAuth(merchantID string, apiKey string, demo bool) *BasicAuthClient {
	return &BasicAuthClient{
		Config: Config{
			Demo:       demo,
			MerchantID: merchantID,
			APIKey:     apiKey,
		},
		HTTPClient: httpClient,
	}
}

// ApiUri returns the uri of the production or the demo api.
func ApiUri(c Config) string {
	if isDemo(c) {
		return "https://demo-api.vivapayments.com"
	}
	return "https://api.vivapayments.com"
}

func AppUri(c Config) string {
	if isDemo(c) {
		return "https://demo.vivapayments.com"
	}
	return "https://www.vivapayments.com"
}

func isDemo(c Config) bool {
	return c.Demo
}
