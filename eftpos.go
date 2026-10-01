package vivawallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
)

// ECRCurrencyEUR is the euro's ISO 4217 numeric code as the EFT POS API takes it: a string,
// unlike the number ISVCheckoutOrder.CurrencyCode takes. The two are deliberately not the
// same type.
const ECRCurrencyEUR = "978"

// ecrISVPrefix is the path prefix of the ISV variant of the EFT POS API. The merchant variant
// differs only by dropping the "isv" segment.
const ecrISVPrefix = "/ecr/isv/v1"

// ECRISVDetails carries the ISV side of a terminal transaction.
type ECRISVDetails struct {
	// Amount is the ISV's fee, in minor units. It is not added to the sale total.
	Amount int64 `json:"amount,omitempty"`
	// TerminalMerchantID is the id of the merchant activated on the terminal.
	TerminalMerchantID string `json:"terminalMerchantId,omitempty"`
}

// ECRSaleRequest asks a card terminal to take a payment.
type ECRSaleRequest struct {
	// SessionID identifies this request and must be unique per request; it is how the
	// outcome is retrieved with GetSession. Generate it yourself, or with NewSessionID.
	SessionID string `json:"sessionId"`
	// TerminalID is the TID of the terminal to send the sale to.
	TerminalID string `json:"terminalId"`
	// CashRegisterID identifies the station that triggered the sale.
	CashRegisterID string `json:"cashRegisterId"`
	// Amount is the amount charged, in minor units, excluding TipAmount.
	Amount int64 `json:"amount"`
	// CurrencyCode is the ISO 4217 numeric code as a string, e.g. ECRCurrencyEUR.
	CurrencyCode string `json:"currencyCode"`
	// MerchantReference ties the payment to your own record, e.g. a booking number. It is
	// searchable in Viva's e-banking.
	MerchantReference string `json:"merchantReference"`
	// CustomerTrns is the description shown to the customer.
	CustomerTrns string `json:"customerTrns"`
	// TipAmount is the tip, in minor units. It is NOT included in Amount: Viva charges
	// Amount + TipAmount. Folding the tip into Amount as well charges it twice.
	TipAmount             int64          `json:"tipAmount,omitempty"`
	ShowTransactionResult bool           `json:"showTransactionResult,omitempty"`
	ShowReceipt           bool           `json:"showReceipt,omitempty"`
	ISVDetails            *ECRISVDetails `json:"isvDetails,omitempty"`
}

// ECRRefundRequest asks a card terminal to refund an amount to any card the customer
// presents.
type ECRRefundRequest struct {
	// SessionID identifies this request and must be unique per request, as for a sale.
	SessionID         string `json:"sessionId"`
	TerminalID        string `json:"terminalId"`
	CashRegisterID    string `json:"cashRegisterId"`
	Amount            int64  `json:"amount"`
	CurrencyCode      string `json:"currencyCode"`
	MerchantReference string `json:"merchantReference"`
	CustomerTrns      string `json:"customerTrns,omitempty"`

	ShowTransactionResult bool           `json:"showTransactionResult,omitempty"`
	ShowReceipt           bool           `json:"showReceipt,omitempty"`
	ISVDetails            *ECRISVDetails `json:"isvDetails,omitempty"`
}

// ECRSession is the state of a terminal request, as returned by GetSession.
//
// Success is not conclusive proof either way. Viva has been reported returning
// success=false with a "Transaction canceled by user" or "Invalid card" message for sessions
// that settled successfully (https://github.com/VivaPayments/API/issues/2461). When money
// is at stake, treat a failed session as "verify" rather than "refund": confirm through the
// Transaction Payment Created / Transaction Failed webhooks, matching on TransactionID or
// OrderCode.
type ECRSession struct {
	SessionID      string `json:"sessionId"`
	TerminalID     string `json:"terminalId"`
	TID            string `json:"tid"`
	CashRegisterID string `json:"cashRegisterId"`
	Amount         int64  `json:"amount"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	// OrderCode and ShortOrderCode arrive as strings ("0" until there is an order), unlike
	// the int64 order codes elsewhere in this SDK.
	OrderCode                string `json:"orderCode"`
	ShortOrderCode           string `json:"shortOrderCode"`
	ReferenceNumber          int64  `json:"referenceNumber"`
	RetrievalReferenceNumber string `json:"retrievalReferenceNumber"`
	AuthorizationID          string `json:"authorizationId"`
	TransactionID            string `json:"transactionId"`
	TransactionTypeID        int    `json:"transactionTypeId"`
	TransactionDateTime      string `json:"transactionDateTime"`
	AbortOperation           bool   `json:"abortOperation"`
	AbortSuccess             bool   `json:"abortSuccess"`
	AbortAckTime             string `json:"abortAckTime"`
}

// NewSessionID returns a random session id for an EFT POS request: 32 hex characters from
// crypto/rand. Any unique string works as a session id; this is a convenience.
func NewSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Sale sends a sale to a card terminal. It returns as soon as Viva accepts the request; the
// customer then pays at the terminal, and the outcome is retrieved with GetSession.
//
// The EFT POS API needs a token with its own OAuth scope, so use an ISVClient created with
// that scope (WithScope) rather than the one used for ISV orders.
// Ref: https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Make-Transactions/paths/~1ecr~1isv~1v1~1transactions:sale/post
func (c *ISVClient) Sale(payload ECRSaleRequest) error {
	return c.postECR(ecrSaleUri(c.Config), payload)
}

// Refund refunds an amount to any card presented at a terminal. Like Sale, it returns once
// the request is accepted; the outcome is retrieved with GetSession.
// Ref: https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Make-Transactions/paths/~1ecr~1isv~1v1~1transactions:refund/post
func (c *ISVClient) Refund(payload ECRRefundRequest) error {
	return c.postECR(ecrRefundUri(c.Config), payload)
}

// GetSession retrieves the state of the sale or refund sent with sessionID. Poll it until
// the terminal reports an outcome; see ECRSession for how far to trust Success.
// Ref: https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Session-Details/paths/~1ecr~1isv~1v1~1sessions~1{sessionId}/get
func (c *ISVClient) GetSession(sessionID string) (*ECRSession, error) {
	session := &ECRSession{}
	reqErr := c.Get(ecrSessionUri(c.Config, sessionID), session)
	if reqErr != nil {
		return nil, reqErr
	}

	return session, nil
}

func (c *ISVClient) postECR(uri string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to parse request %s", err)
	}

	// The response carries nothing the caller needs; the outcome comes from GetSession.
	var ignored json.RawMessage
	return c.Post(uri, bytes.NewReader(data), &ignored)
}

// The ":sale" / ":refund" suffixes are custom-method syntax and part of the path; they must
// not be percent-encoded.
func ecrSaleUri(c Config) string {
	return fmt.Sprintf("%s%s/transactions:sale", ApiUri(c), ecrISVPrefix)
}

func ecrRefundUri(c Config) string {
	return fmt.Sprintf("%s%s/transactions:refund", ApiUri(c), ecrISVPrefix)
}

func ecrSessionUri(c Config, sessionID string) string {
	return fmt.Sprintf("%s%s/sessions/%s", ApiUri(c), ecrISVPrefix, url.PathEscape(sessionID))
}
