package viva_wallet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	Demo         bool
	ClientID     string
	ClientSecret string
}

type Client struct {
	config     Config
	HTTPClient *http.Client
}

type IClient interface {
	New(clientID string, clientSecret string) *Client
}

// defaultHTTPTimeout is the default timeout on the http.Client used by the library.
const defaultTimeout = 60 * time.Second

var httpClient = &http.Client{
	Timeout: defaultTimeout,
}

// New creates a new client
func New(clientID string, clientSecret string) *Client {
	return &Client{
		config: Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		HTTPClient: httpClient,
	}
}

// Authentication flow
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func (c Client) Authenticate() (*TokenResponse, error) {
	uri := c.tokenEndpoint()

	auth := authBody(c.config)
	grant := []byte("grant_type=client_credentials")
	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(grant))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Errorf("failed to perform access token request", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("non successful response")
	}

	body, err2 := io.ReadAll(resp.Body)
	if err2 != nil {
		return nil, err2
	}

	responseObject := &TokenResponse{}
	if err3 := json.Unmarshal(body, responseObject); err3 != nil {
		return nil, err3
	}

	return responseObject, nil
}

func authBody(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c Client) tokenEndpoint() string {
	return fmt.Sprintf("%s/%s", c.authUri(), "/connect/token")
}

func (c Client) authUri() string {
	if isDemo(c.config) {
		return "https://demo-accounts.vivapayments.com"
	}
	return "https://accounts.vivapayments.com"
}

func ApiUri(c Config) string {
	if isDemo(c) {
		return "https://demo-api.vivapayments.com"
	}
	return "https://api.vivapayments.com"
}

func isDemo(c Config) bool {
	return c.Demo
}
