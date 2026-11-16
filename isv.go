package vivawallet

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// ISVClient talks to Viva's ISV Partner APIs on behalf of a merchant. One client serves
// many merchants: the merchant id is a per-call argument, never client state, so a single
// ISVClient is safe to share across goroutines serving different merchants.
//
// It carries both credential sets an ISV partner needs. The reseller id and reseller API
// key authenticate the older /api/* ISV endpoints with basic auth; the client id and
// secret authenticate the /checkout/v2/isv/* endpoints with a bearer token.
//
// Ref: https://developer.viva.com/isv-partner-program/payment-isv-api/
type ISVClient struct {
	Config Config

	// Client performs every request this client makes. Prefer setting it at construction
	// with WithHTTPClient; mutating it afterwards races with in-flight requests.
	Client *http.Client

	tokenCache
}

// NewISV creates a new viva client for the ISV partner apis.
//
// resellerID and resellerAPIKey are the ISV partner's basic-auth credentials; clientID and
// clientSecret are its OAuth client credentials. The merchant the client acts for is passed
// to each call rather than fixed here.
//
// Without options the client gets its own http.Client with a 60s timeout. Use
// WithHTTPClient, WithTimeout, WithBaseURLs or WithScope to change that.
func NewISV(resellerID, resellerAPIKey, clientID, clientSecret string, demo bool, opts ...Option) *ISVClient {
	o := newOptions(opts...)

	cfg := Config{
		Demo:           demo,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		ResellerID:     resellerID,
		ResellerAPIKey: resellerAPIKey,
	}
	o.applyTo(&cfg)

	return &ISVClient{
		Config: cfg,
		Client: o.client(),
	}
}

// isvBasicAuth returns the username and password for a basic-auth call made on behalf of
// merchantID: username "resellerID:merchantID", password "resellerAPIKey", as documented in
// Viva's ISV API spec (https://developer.viva.com/downloads/payment-isv-api.yaml).
func isvBasicAuth(c Config, merchantID string) (username, password string) {
	return c.ResellerID + ":" + merchantID, c.ResellerAPIKey
}

// ISVBasicAuth returns the base64-encoded basic-auth credential an ISV call for merchantID
// carries, mirroring BasicAuth for the single-merchant client.
func ISVBasicAuth(c Config, merchantID string) string {
	username, password := isvBasicAuth(c, merchantID)
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// Authenticate retrieves the access token used by the ISV OAuth endpoints and stores it
// for later use, in the same way OAuthClient.Authenticate does. Like there, calling it is
// optional: bearer-token requests refresh an expired token on their own.
func (c *ISVClient) Authenticate() (*TokenResponse, error) {
	return c.authenticate(context.Background())
}

func (c *ISVClient) authenticate(ctx context.Context) (*TokenResponse, error) {
	resp, err := requestToken(ctx, c.Client, tokenEndpoint(c.Config), c.Config.ClientID, c.Config.ClientSecret, c.Config.scope)
	if err != nil {
		return nil, err
	}

	c.storeToken(resp)

	return resp, nil
}

// GetWithMerchant performs a basic-auth GET on behalf of merchantID.
func (c *ISVClient) GetWithMerchant(merchantID, uri string, v interface{}) error {
	req, err := newRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	return c.performBasicAuthReq(merchantID, req, v)
}

// PostWithMerchant performs a basic-auth POST on behalf of merchantID.
func (c *ISVClient) PostWithMerchant(merchantID, uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("POST", uri, reader)
	if err != nil {
		return err
	}

	return c.performBasicAuthReq(merchantID, req, v)
}

// DeleteWithMerchant performs a basic-auth DELETE on behalf of merchantID.
func (c *ISVClient) DeleteWithMerchant(merchantID, uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("DELETE", uri, reader)
	if err != nil {
		return err
	}

	return c.performBasicAuthReq(merchantID, req, v)
}

// Post performs a bearer-token POST against the ISV OAuth endpoints.
func (c *ISVClient) Post(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("POST", uri, reader)
	if err != nil {
		return err
	}

	return c.performBearerReq(req, v)
}

// Get performs a bearer-token GET against the ISV OAuth endpoints.
func (c *ISVClient) Get(uri string, v interface{}) error {
	req, err := newRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	return c.performBearerReq(req, v)
}

// Patch performs a bearer-token PATCH against the ISV OAuth endpoints.
func (c *ISVClient) Patch(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("PATCH", uri, reader)
	if err != nil {
		return err
	}

	return c.performBearerReq(req, v)
}

// Delete performs a bearer-token DELETE against the ISV OAuth endpoints.
func (c *ISVClient) Delete(uri string, reader *bytes.Reader, v interface{}) error {
	req, err := newRequest("DELETE", uri, reader)
	if err != nil {
		return err
	}

	return c.performBearerReq(req, v)
}

func (c *ISVClient) performBasicAuthReq(merchantID string, req *http.Request, v interface{}) error {
	req.SetBasicAuth(isvBasicAuth(c.Config, merchantID))

	return c.do(c.Client, req, v)
}

// performBearerReq sends req with a bearer token, refreshing the token first if it has
// expired.
func (c *ISVClient) performBearerReq(req *http.Request, v interface{}) error {
	return c.do(bearerClient(c.Client, c.Config, &c.tokenCache, c.authenticate), req, v)
}

func (c *ISVClient) do(hc *http.Client, req *http.Request, v interface{}) error {
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := hc.Do(req)
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
