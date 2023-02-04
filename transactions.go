package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TransactionResponse struct {
	Email               string    `json:"email"`
	Amount              int       `json:"amount"`
	OrderCode           string    `json:"orderCode"`
	StatusID            string    `json:"statusId"`
	FullName            string    `json:"fullName"`
	InsDate             time.Time `json:"insDate"`
	CardNumber          string    `json:"cardNumber"`
	CurrencyCode        int       `json:"currencyCode"`
	CustomerTrns        string    `json:"customerTrns"`
	MerchantTrns        string    `json:"merchantTrns"`
	TransactionTypeID   int       `json:"transactionTypeId"`
	RecurringSupport    bool      `json:"recurringSupport"`
	TotalInstallments   int       `json:"totalInstallments"`
	CardCountryCode     string    `json:"cardCountryCode"`
	CardIssuingBank     string    `json:"cardIssuingBank"`
	CurrentInstallment  int       `json:"currentInstallment"`
	CardUniqueReference string    `json:"cardUniqueReference"`
	CardTypeID          int       `json:"cardTypeId"`
	DigitalWalletID     int       `json:"digitalWalletId"`
}

func (c Client) GetTransaction(trxID string) (*TransactionResponse, error) {
	uri := getTransactionUri(c.Config, trxID)

	// TODO: use RoundTripper to avoid rewriting this
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	req, _ := http.NewRequest("GET", uri, nil)
	// TODO: use RoundTripper
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AuthToken()))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to fetch transaction %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch transaction with status %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	trx := &TransactionResponse{}
	if jsonErr := json.Unmarshal(body, trx); jsonErr != nil {
		return nil, jsonErr
	}
	return trx, nil
}

func getTransactionUri(c Config, trxID string) string {
	return fmt.Sprintf("%s/checkout/v2/transactions/%s", ApiUri(c), trxID)
}

type CreateCardToken struct {
	TransactionID string `json:"transactionId"`
}

type CardTokenResponse struct {
	Token string `json:"token"`
}

func (c Client) CreateCardToken(payload CreateCardToken) (*CardTokenResponse, error) {
	// TODO: use RoundTripper to avoid rewriting this
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	uri := getCreateCardTokenUri(c.Config)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CreateCardToken %s", err)
	}

	req, _ := http.NewRequest("POST", uri, bytes.NewReader(data))
	// TODO: use RoundTripper
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AuthToken()))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to create card token %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create card token with status %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	cardToken := &CardTokenResponse{}
	if jsonErr := json.Unmarshal(body, cardToken); jsonErr != nil {
		return nil, jsonErr
	}
	return cardToken, nil
}

func getCreateCardTokenUri(c Config) string {
	return fmt.Sprintf("%s/acquiring/v1/cards/tokens", ApiUri(c))
}
