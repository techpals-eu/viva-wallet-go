package vivawallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Transaction struct {
}

func (c Client) GetTransaction(trxID string) (*Transaction, error) {
	uri := getTransactionUri(c.Config, trxID)

	// TODO: use RoundTripper to avoid rewriting this
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		fmt.Errorf("authentication error %s", authErr)
		return nil, authErr
	}

	req, _ := http.NewRequest("GET", uri, nil)
	// TODO: use RoundTripper
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AuthToken()))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		fmt.Errorf("failed to fetch transaction", httpErr)
		return nil, httpErr
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("failed to fetch transaction with status %d", resp.StatusCode))
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	trx := &Transaction{}
	if jsonErr := json.Unmarshal(body, trx); jsonErr != nil {
		return nil, jsonErr
	}
	return trx, nil
}

func getTransactionUri(c Config, trxID string) string {
	return fmt.Sprintf("%s/%s/%s", ApiUri(c), "checkout/v2/transactions", trxID)
}

