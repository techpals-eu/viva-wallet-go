package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type GetTransactionResponse struct {
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
func (c OAuthClient) GetTransaction(trxID string) (*GetTransactionResponse, error) {
	// TODO: use RoundTripper to avoid rewriting this
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	uri := getTransactionUri(c.Config, trxID)

	trx := &GetTransactionResponse{}
	reqErr := c.Get(uri, &trx)
	if reqErr != nil {
		return nil, reqErr
	}

	return trx, nil
}

func getTransactionUri(c Config, trxID string) string {
	return fmt.Sprintf("%s/checkout/v2/transactions/%s", ApiUri(c), trxID)
}

type CreateTransaction struct {
	Amount       int64  `json:"amount"`
	Installments int    `json:"installments,omitempty"`
	CustomerTrnx string `json:"customerTrns,omitempty"`
	MerchantTrns string `json:"merchantTrns,omitempty"`
	SourceCode   string `json:"sourceCode,omitempty"`
	TipAmount    int    `json:"tipAmount,omitempty"`
}

type TransactionResponse struct {
	Emv                      string   `json:"Emv,omitempty"`
	Amount                   float64   `json:"Amount"`
	StatusID                 string    `json:"StatusId,omitempty"`
	CurrencyCode             string    `json:"CurrencyCode,omitempty"`
	TransactionID            string    `json:"TransactionId,omitempty"`
	ReferenceNumber          int       `json:"ReferenceNumber,omitempty"`
	AuthorizationID          string    `json:"AuthorizationId,omitempty"`
	RetrievalReferenceNumber string    `json:"RetrievalReferenceNumber,omitempty"`
	ThreeDSecureStatusID     int       `json:"ThreeDSecureStatusId,omitempty"`
	ErrorCode                int       `json:"ErrorCode,omitempty"`
	ErrorText                string    `json:"ErrorText,omitempty"`
	Timestamp                time.Time `json:"TimeStamp,omitempty"`
	CorrelationID            string    `json:"CorrelationId,omitempty"`
	EventID                  int       `json:"EventId,omitempty"`
	Success                  bool      `json:"Success,omitempty"`
}

// CreateTransaction creates a new transaction for a recurring payment or a pre-auth
// order payment
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1api~1transactions~1{transaction_id}/post
func (c BasicAuthClient) CreateTransaction(id string, payload CreateTransaction) (*TransactionResponse, error) {
	uri := getCreateTransactionUri(c.Config, id)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction %s", err.Error())
	}

	trx := &TransactionResponse{}
	reqErr := c.Post(uri, bytes.NewReader(data), &trx)

	if reqErr != nil {
		return nil, reqErr
	}

	return trx, nil
}

func getCreateTransactionUri(c Config, id string) string {
	return fmt.Sprintf("%s/api/transactions/%s", AppUri(c), id)
}

// CancelTransaction cancels a transaction
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1api~1transactions~1{transaction_id}/delete
func (c BasicAuthClient) CancelTransaction(id string, amount int64, sourceCode string) (*TransactionResponse, error) {
	uri := getCancelTransactionUri(c.Config, id, amount, sourceCode)

	trx := &TransactionResponse{}
	reqErr := c.Delete(uri, nil, &trx)

	if reqErr != nil {
		return nil, reqErr
	}

	return trx, nil
}

func getCancelTransactionUri(c Config, id string, amount int64, sourceCode string) string {
	var sourceParam = ""
	if sourceCode != "" {
		sourceParam = "&sourceCode=" + sourceCode
	}

	return fmt.Sprintf("%s/api/transactions/%s?amount=%d%s", AppUri(c), id, amount, sourceParam)
}

// CancelPartialAuthorization cancels a partial authorization
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1acquiring~1v1~1transactions~1{transactionId}/delete
func (c OAuthClient) CancelPartialAuthorization(id string, amount int64, sourceCode string) error {
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return fmt.Errorf("authentication error %s", authErr)
	}
	uri := getCancelPartialAuthUri(c.Config, id, amount, sourceCode)

	var response string
	reqErr := c.Delete(uri, nil, &response)
	if reqErr != nil {
		return reqErr
	}

	return nil
}

func getCancelPartialAuthUri(c Config, id string, amount int64, sourceCode string) string {
	var sourceParam = ""
	if sourceCode != "" {
		sourceParam = "&sourceCode=" + sourceCode
	}

	return fmt.Sprintf("%s/acquiring/v1/transactions/%s?amount=%d%s", ApiUri(c), id, amount, sourceParam)
}
