package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type TransactionResponse struct {
	Email               string    `json:"email"`
	Amount              float64   `json:"amount"`
	OrderCode           int       `json:"orderCode"`
	StatusID            string    `json:"statusId"`
	FullName            string    `json:"fullName"`
	InsDate             time.Time `json:"insDate"`
	CardNumber          string    `json:"cardNumber"`
	CurrencyCode        string    `json:"currencyCode"`
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

// GetTransaction fetches a transaction given an ID.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions/paths/~1checkout~1v2~1transactions~1{transactionId}/get
func (c OAuthClient) GetTransaction(trxID string) (*TransactionResponse, error) {
	// TODO: use RoundTripper to avoid rewriting this
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	uri := getTransactionUri(c.Config, trxID)

	trx := &TransactionResponse{}
	reqErr := c.Get(uri, &trx)
	if reqErr != nil {
		return nil, reqErr
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

// CreateCardToken creates card tokens based on a transactionID.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions/paths/~1acquiring~1v1~1cards~1tokens/post
//
// > This feature is available only upon request. Please contact your sales representative or use our Live Chat to request this feature.
//
func (c OAuthClient) CreateCardToken(payload CreateCardToken) (*CardTokenResponse, error) {
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

	cardToken := &CardTokenResponse{}
	reqErr := c.Post(uri, bytes.NewReader(data), &cardToken)
	if reqErr != nil {
		return nil, reqErr
	}

	return cardToken, nil
}

func getCreateCardTokenUri(c Config) string {
	return fmt.Sprintf("%s/acquiring/v1/cards/tokens", ApiUri(c))
}
