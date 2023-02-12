package vivawallet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// New creates a new viva client for the oauth apis
func NewOAuth(clientID string, clientSecret string, demo bool) *OAuthClient {
	return &OAuthClient{
		Config: Config{
			Demo:         demo,
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		Client:     httpClient,
		tokenValue: &token{},
		lock:       &sync.RWMutex{},
	}
}

func (c OAuthClient) Post(uri string, reader *bytes.Reader, v interface{}) error {
	req, _ := http.NewRequest("POST", uri, reader)
	return c.performReq(req, v)
}

func (c OAuthClient) Get(uri string, v interface{}) error {
	req, _ := http.NewRequest("GET", uri, nil)
	return c.performReq(req, v)
}

func (c OAuthClient) Patch(uri string, reader *bytes.Reader, v interface{}) error {
	req, _ := http.NewRequest("PATCH", uri, nil)
	return c.performReq(req, v)
}

func (c OAuthClient) setBearerToken(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.AuthToken())
}

func (c OAuthClient) performReq(req *http.Request, v interface{}) error {
	req.Header.Add("Content-Type", "application/json")
	c.setBearerToken(req)

	resp, httpErr := c.Client.Do(req)
	if httpErr != nil {
		return fmt.Errorf("failed to perform request %s", httpErr)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr == nil {
		resp.Body.Close()
	}

	if bodyErr != nil {
		return bodyErr
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to perform request with status %d", resp.StatusCode)
	}

	return json.Unmarshal(body, v)
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// Authenticate retrieves the access token to continue making requests to Viva's API. It
// returns the full response of the API and stores the token and expiration time for
// later use.
func (c OAuthClient) Authenticate() (*TokenResponse, error) {
	uri := c.tokenEndpoint()

	grant := []byte("grant_type=client_credentials")
	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(grant))
	req.SetBasicAuth(c.Config.ClientID, c.Config.ClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, httpErr := c.Client.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to perform access token request %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("non successful response")
	}

	defer resp.Body.Close()
	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	response := &TokenResponse{}
	if jsonErr := json.Unmarshal(body, response); jsonErr != nil {
		return nil, jsonErr
	}

	expiry := time.Now().Add(time.Second * time.Duration(response.ExpiresIn))
	c.SetToken(response.AccessToken, expiry)

	return response, nil
}

// AuthToken returns the token value
func (c OAuthClient) AuthToken() string {
	c.lock.RLock()

	t := c.tokenValue.value

	c.lock.RUnlock()
	return t
}

// SetToken sets the token value and the expiration time of the token.
func (c OAuthClient) SetToken(value string, expires time.Time) {
	c.lock.Lock()

	c.tokenValue.value = value
	c.tokenValue.expires = expires

	c.lock.Unlock()
}

// HasAuthExpired returns true if the expiry time of the token has passed and false
// otherwise.
func (c OAuthClient) HasAuthExpired() bool {
	c.lock.RLock()

	expires := c.tokenValue.expires

	c.lock.RUnlock()

	now := time.Now()
	return now.After(expires)
}

func AuthBody(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func BasicAuth(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.MerchantID, c.APIKey)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c OAuthClient) tokenEndpoint() string {
	return fmt.Sprintf("%s/%s", c.authUri(), "/connect/token")
}

func (c OAuthClient) authUri() string {
	if isDemo(c.Config) {
		return "https://demo-accounts.vivapayments.com"
	}
	return "https://accounts.vivapayments.com"
}
