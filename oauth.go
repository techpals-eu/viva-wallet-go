package vivawallet

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// NewOAuth creates a new viva client for the oauth apis.
//
// Without options the client gets its own http.Client with a 60s timeout. Use
// WithHTTPClient, WithTimeout or WithBaseURLs to change that.
func NewOAuth(clientID string, clientSecret string, demo bool, opts ...Option) *OAuthClient {
	o := newOptions(opts...)

	cfg := Config{
		Demo:         demo,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	o.applyTo(&cfg)

	return &OAuthClient{
		Config: cfg,
		Client: o.client(),
	}
}

func (c *OAuthClient) Post(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("POST", uri, reader)
	if err != nil {
		return err
	}

	return c.performReq(req, v)
}

func (c *OAuthClient) Get(uri string, v interface{}) error {
	req, err := newRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	return c.performReq(req, v)
}

func (c *OAuthClient) Patch(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("PATCH", uri, reader)
	if err != nil {
		return err
	}

	return c.performReq(req, v)
}

func (c *OAuthClient) Delete(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("DELETE", uri, reader)
	if err != nil {
		return err
	}

	return c.performReq(req, v)
}

// performReq sends req with a bearer token, refreshing the token first if it has expired.
func (c *OAuthClient) performReq(req *http.Request, v interface{}) error {
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := bearerClient(c.Client, c.Config, &c.tokenCache, c.authenticate).Do(req)
	if httpErr != nil {
		return fmt.Errorf("failed to perform request: %w", httpErr)
	}

	body, bodyErr := readBody(resp)
	if bodyErr != nil {
		return bodyErr
	}

	if !isSuccess(resp.StatusCode) {
		return newAPIError(resp.StatusCode, body)
	}

	if len(body) == 0 {
		return nil
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
//
// Calling it is optional: every request refreshes an expired token on its own. Call it to
// fail fast on bad credentials, or to read the token response.
func (c *OAuthClient) Authenticate() (*TokenResponse, error) {
	return c.authenticate(context.Background())
}

// authenticate fetches and stores a token on the plain http.Client, never the bearer one,
// so a refresh cannot recurse into itself.
func (c *OAuthClient) authenticate(ctx context.Context) (*TokenResponse, error) {
	resp, err := requestToken(ctx, c.Client, c.tokenEndpoint(), c.Config.ClientID, c.Config.ClientSecret, c.Config.scope)
	if err != nil {
		return nil, err
	}

	c.storeToken(resp)

	return resp, nil
}

// requestToken performs the client-credentials token request shared by every bearer-token
// client. scope is omitted from the request body when empty, which is what the non-ISV
// OAuth flow expects.
func requestToken(ctx context.Context, hc *http.Client, uri, clientID, clientSecret, scope string) (*TokenResponse, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	if scope != "" {
		form.Set("scope", scope)
	}

	req, reqErr := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBufferString(form.Encode()))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to build access token request: %w", reqErr)
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, httpErr := hc.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to perform access token request: %w", httpErr)
	}

	body, bodyErr := readBody(resp)
	if bodyErr != nil {
		return nil, bodyErr
	}

	if !isSuccess(resp.StatusCode) {
		return nil, newAPIError(resp.StatusCode, body)
	}

	response := &TokenResponse{}
	if jsonErr := json.Unmarshal(body, response); jsonErr != nil {
		return nil, jsonErr
	}

	return response, nil
}

func AuthBody(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func BasicAuth(c Config) string {
	auth := fmt.Sprintf("%s:%s", c.MerchantID, c.APIKey)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c *OAuthClient) tokenEndpoint() string {
	return tokenEndpoint(c.Config)
}

// tokenEndpoint returns the OAuth token endpoint for the configured environment.
func tokenEndpoint(c Config) string {
	return authUri(c) + "/connect/token"
}

func (c *OAuthClient) authUri() string {
	return authUri(c.Config)
}

func authUri(c Config) string {
	if c.authBase != "" {
		return c.authBase
	}
	if isDemo(c) {
		return "https://demo-accounts.vivapayments.com"
	}
	return "https://accounts.vivapayments.com"
}
