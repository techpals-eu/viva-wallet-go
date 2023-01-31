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
}

type token struct {
	value   string
	expires time.Time
}

type Client struct {
	Config     Config
	HTTPClient *http.Client
	lock       *sync.RWMutex
	tokenValue token
}

// defaultHTTPTimeout is the default timeout on the http.Client used by the library.
const defaultTimeout = 60 * time.Second

var httpClient = &http.Client{
	Timeout: defaultTimeout,
}

// New creates a new viva client
func New(clientID string, clientSecret string, demo bool) *Client {
	return &Client{
		Config: Config{
			Demo:         demo,
			ClientID:     clientID,
			ClientSecret: clientSecret,
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

func isDemo(c Config) bool {
	return c.Demo
}

