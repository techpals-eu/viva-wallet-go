package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// New creates a new viva client for the basic auth apis
func NewBasicAuth(merchantID string, apiKey string, demo bool) *BasicAuthClient {
	return &BasicAuthClient{
		Config: Config{
			Demo:       demo,
			MerchantID: merchantID,
			APIKey:     apiKey,
		},
		Client: httpClient,
	}
}

func (c BasicAuthClient) Get(uri string, v interface{}) error {
	req := newRequest("GET", uri, nil)
	body, reqErr := c.performReq(req)
	if reqErr != nil {
		return reqErr
	}

	if len(body) == 0 {
		return nil
	}

	return json.Unmarshal(body, v)
}

func (c BasicAuthClient) Post(uri string, reader *bytes.Reader, v interface{}) error {
	req := newRequest("POST", uri, reader)
	body, reqErr := c.performReq(req)
	if reqErr != nil {
		return reqErr
	}

	if len(body) == 0 {
		return nil
	}

	return json.Unmarshal(body, v)
}

func (c BasicAuthClient) Patch(uri string, reader *bytes.Reader) error {
	req := newRequest("PATCH", uri, reader)
	_, reqErr := c.performReq(req)
	if reqErr != nil {
		return reqErr
	}

	return nil
}

func (c BasicAuthClient) Delete(uri string, reader *bytes.Reader, v interface{}) error {
	req := newRequest("DELETE", uri, reader)
	body, reqErr := c.performReq(req)
	if reqErr != nil {
		return reqErr
	}

	if len(body) == 0 {
		return nil
	}

	return json.Unmarshal(body, v)
}

func (c BasicAuthClient) performReq(req *http.Request) ([]byte, error) {
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(c.Config.MerchantID, c.Config.APIKey)

	resp, httpErr := c.Client.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to perform request: %w", httpErr)
	}

	body, bodyErr := readBody(resp)
	if bodyErr != nil {
		return nil, bodyErr
	}

	if !isSuccess(resp.StatusCode) {
		return nil, newAPIError(resp.StatusCode, body)
	}

	return body, nil
}
