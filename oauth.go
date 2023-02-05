package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

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

	resp, httpErr := c.HTTPClient.Do(req)
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
