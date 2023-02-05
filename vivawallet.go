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
}

type token struct {
	value   string
	expires time.Time
}

type OAuthClient struct {
	Config     Config
	HTTPClient *http.Client
	lock       *sync.RWMutex
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
		lock:       &sync.RWMutex{},
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

func (c OAuthClient) setBearerToken(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.AuthToken())
}

func (c OAuthClient) post(uri string, reader *bytes.Reader) ([]byte, error) {
	req, _ := http.NewRequest("POST", uri, reader)
	return c.performReq(req)
}

func (c OAuthClient) get(uri string) ([]byte, error) {
	req, _ := http.NewRequest("GET", uri, nil)
	return c.performReq(req)
}

func (c OAuthClient) performReq(req *http.Request) ([]byte, error) {
	req.Header.Add("Content-Type", "application/json")
	c.setBearerToken(req)

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to parse response %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to make response %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// BasicAuthClient
func (c BasicAuthClient) get(uri string) ([]byte, error) {
	req, _ := http.NewRequest("GET", uri, nil)
	return c.performReq(req)
}

func (c BasicAuthClient) post(uri string, reader *bytes.Reader) ([]byte, error) {
	req, _ := http.NewRequest("POST", uri, reader)
	return c.performReq(req)
}

func (c BasicAuthClient) performReq(req *http.Request) ([]byte, error) {
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(c.Config.MerchantID, c.Config.APIKey)

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to get wallet %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get wallet with status %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
