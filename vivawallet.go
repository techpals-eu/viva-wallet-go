package vivawallet

import (
	"bytes"
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
	Client     *http.Client
	lock       *sync.RWMutex
	tokenValue *token
}

type BasicAuthClient struct {
	Config Config
	Client *http.Client
}

// defaultHTTPTimeout is the default timeout on the http.Client used by the library.
const defaultTimeout = 60 * time.Second

var (
	httpClient *http.Client
)

func init() {
	httpClient = &http.Client{
		Timeout: defaultTimeout,
	}
}

type Client interface {
	Get(uri string, v interface{}) error
	Post(uri string, reader *bytes.Reader, v interface{}) error
	Patch(uri string, reader *bytes.Reader, v interface{}) error
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

func newRequest(method string, uri string, reader *bytes.Reader) *http.Request {
	var req *http.Request
	if reader != nil {
		req, _ = http.NewRequest(method, uri, reader)
	} else {
		req, _ = http.NewRequest(method, uri, nil)
	}
	return req
}
