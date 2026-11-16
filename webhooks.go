package vivawallet

import (
	"encoding/json"
	"fmt"
)

// Webhook event type ids, as sent in WebhookEvent.EventTypeID.
// Ref: https://developer.viva.com/webhooks-for-payments/#webhook-events
const (
	EventTransactionPaymentCreated        = 1796
	EventTransactionReversalCreated       = 1797
	EventTransactionFailed                = 1798
	EventAccountConnected                 = 8193
	EventAccountVerificationStatusChanged = 8194
)

// WebhookEvent is the envelope every Viva webhook notification arrives in. EventData is left
// undecoded because its shape depends on EventTypeID; use the typed accessor for the event,
// e.g. TransactionPaymentCreated.
//
// Viva can deliver the same event more than once. Deduplicate on MessageID: the SDK keeps no
// state and cannot do it for you.
//
// Date-times are left as strings: Viva sends them in more than one format, some without a
// timezone, and a webhook that fails to parse is retried by Viva for a day.
type WebhookEvent struct {
	URL           string          `json:"Url"`
	EventData     json.RawMessage `json:"EventData"`
	Created       string          `json:"Created"`
	CorrelationID string          `json:"CorrelationId"`
	EventTypeID   int             `json:"EventTypeId"`
	// Delay is null in every documented payload and its type is not documented, so it is
	// left undecoded rather than risk failing the whole parse.
	Delay               json.RawMessage `json:"Delay"`
	RetryCount          int             `json:"RetryCount"`
	RetryDelayInSeconds *int            `json:"RetryDelayInSeconds"`
	MessageID           string          `json:"MessageId"`
	RecipientID         string          `json:"RecipientId"`
	MessageTypeID       int             `json:"MessageTypeId"`
}

// ParseWebhook decodes a webhook notification body. Only the envelope is decoded, so an
// event type this SDK does not know still parses; check EventTypeID and call the matching
// accessor for the event data.
func ParseWebhook(body []byte) (*WebhookEvent, error) {
	event := &WebhookEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	return event, nil
}

// TransactionEventData is the event data of the Transaction Payment Created, Transaction
// Reversal Created and Transaction Failed webhooks, which share one shape.
//
// Unlike the order APIs, amounts here are decimal currency units (10.5 is €10.50), not
// minor units, and a reversal carries a negative Amount.
type TransactionEventData struct {
	TransactionID     string            `json:"TransactionId"`
	OrderCode         int64             `json:"OrderCode"`
	MerchantID        string            `json:"MerchantId"`
	ParentID          string            `json:"ParentId"`
	StatusID          TransactionStatus `json:"StatusId"`
	TransactionTypeID int               `json:"TransactionTypeId"`
	ResponseCode      string            `json:"ResponseCode"`
	ResponseEventID   *int              `json:"ResponseEventId"`

	Amount               float64  `json:"Amount"`
	OriginalAmount       float64  `json:"OriginalAmount"`
	TipAmount            float64  `json:"TipAmount"`
	TotalFee             float64  `json:"TotalFee"`
	RedeemedAmount       float64  `json:"RedeemedAmount"`
	SurchargeAmount      *float64 `json:"SurchargeAmount"`
	ConversionRate       float64  `json:"ConversionRate"`
	CurrencyCode         string   `json:"CurrencyCode"`
	OriginalCurrencyCode string   `json:"OriginalCurrencyCode"`
	TotalInstallments    int      `json:"TotalInstallments"`
	CurrentInstallment   int      `json:"CurrentInstallment"`

	MerchantTrns string   `json:"MerchantTrns"`
	CustomerTrns string   `json:"CustomerTrns"`
	Tags         []string `json:"Tags"`
	SourceCode   string   `json:"SourceCode"`
	SourceName   string   `json:"SourceName"`
	OrderCulture string   `json:"OrderCulture"`

	Email    string `json:"Email"`
	Phone    string `json:"Phone"`
	FullName string `json:"FullName"`

	CardNumber                      string `json:"CardNumber"`
	CardToken                       string `json:"CardToken"`
	CardUniqueReference             string `json:"CardUniqueReference"`
	CardCountryCode                 string `json:"CardCountryCode"`
	CardIssuingBank                 string `json:"CardIssuingBank"`
	CardExpirationDate              string `json:"CardExpirationDate"`
	CardTypeID                      int    `json:"CardTypeId"`
	PrimaryAccountNumberLast4Digits string `json:"PrimaryAccountNumberLast4Digits"`
	DigitalWalletID                 *int   `json:"DigitalWalletId"`
	BankID                          string `json:"BankId"`
	AuthorizationID                 string `json:"AuthorizationId"`
	ReferenceNumber                 int64  `json:"ReferenceNumber"`
	RetrievalReferenceNumber        string `json:"RetrievalReferenceNumber"`
	// ElectronicCommerceIndicator arrives as a string ("5") in some events and a number (5)
	// in others; json.Number accepts both.
	ElectronicCommerceIndicator json.Number `json:"ElectronicCommerceIndicator"`
	PanEntryMode                string      `json:"PanEntryMode"`
	Moto                        bool        `json:"Moto"`
	IsDcc                       bool        `json:"IsDcc"`
	IsManualRefund              bool        `json:"IsManualRefund"`

	TerminalID           int64  `json:"TerminalId"`
	ChannelID            string `json:"ChannelId"`
	BatchID              string `json:"BatchId"`
	MerchantCategoryCode int    `json:"MerchantCategoryCode"`
	CompanyName          string `json:"CompanyName"`
	CompanyTitle         string `json:"CompanyTitle"`
	ResellerID           string `json:"ResellerId"`
	ResellerSourceCode   string `json:"ResellerSourceCode"`
	ConnectedAccountID   string `json:"ConnectedAccountId"`
	InsDate              string `json:"InsDate"`
	ClearanceDate        string `json:"ClearanceDate"`
}

// AccountConnectedData is the event data of the Account Connected webhook.
type AccountConnectedData struct {
	PersonID           string `json:"PersonId"`
	WalletID           int64  `json:"WalletId"`
	PlatformPersonID   string `json:"PlatformPersonId"`
	ConnectedAccountID string `json:"ConnectedAccountId"`
}

// AccountVerificationStatusChangedData is the event data of the Account Verification Status
// Changed webhook.
type AccountVerificationStatusChangedData struct {
	Verified           bool   `json:"Verified"`
	PersonID           string `json:"PersonId"`
	PlatformPersonID   string `json:"PlatformPersonId"`
	ConnectedAccountID string `json:"ConnectedAccountId"`
}

// TransactionPaymentCreated returns the event data of a Transaction Payment Created event.
// It returns an error if the event is of another type.
// Ref: https://developer.viva.com/webhooks-for-payments/transaction-payment-created/
func (e *WebhookEvent) TransactionPaymentCreated() (*TransactionEventData, error) {
	data := &TransactionEventData{}
	return data, e.decodeData(EventTransactionPaymentCreated, data)
}

// TransactionReversalCreated returns the event data of a Transaction Reversal Created event.
// It returns an error if the event is of another type.
// Ref: https://developer.viva.com/webhooks-for-payments/transaction-reversal-created/
func (e *WebhookEvent) TransactionReversalCreated() (*TransactionEventData, error) {
	data := &TransactionEventData{}
	return data, e.decodeData(EventTransactionReversalCreated, data)
}

// TransactionFailed returns the event data of a Transaction Failed event. It returns an
// error if the event is of another type.
// Ref: https://developer.viva.com/webhooks-for-payments/transaction-failed/
func (e *WebhookEvent) TransactionFailed() (*TransactionEventData, error) {
	data := &TransactionEventData{}
	return data, e.decodeData(EventTransactionFailed, data)
}

// AccountConnected returns the event data of an Account Connected event. It returns an error
// if the event is of another type.
// Ref: https://developer.viva.com/webhooks-for-payments/account-connected/
func (e *WebhookEvent) AccountConnected() (*AccountConnectedData, error) {
	data := &AccountConnectedData{}
	return data, e.decodeData(EventAccountConnected, data)
}

// AccountVerificationStatusChanged returns the event data of an Account Verification Status
// Changed event. It returns an error if the event is of another type.
// Ref: https://developer.viva.com/webhooks-for-payments/account-verification-status-changed/
func (e *WebhookEvent) AccountVerificationStatusChanged() (*AccountVerificationStatusChangedData, error) {
	data := &AccountVerificationStatusChangedData{}
	return data, e.decodeData(EventAccountVerificationStatusChanged, data)
}

func (e *WebhookEvent) decodeData(want int, v interface{}) error {
	if e.EventTypeID != want {
		return fmt.Errorf("webhook event type is %d, not %d", e.EventTypeID, want)
	}
	if err := json.Unmarshal(e.EventData, v); err != nil {
		return fmt.Errorf("failed to parse webhook event data: %w", err)
	}
	return nil
}

// WebhookVerificationKey is the key a webhook endpoint returns to Viva's verification GET.
type WebhookVerificationKey struct {
	// Key is decoded from either "Key" (merchant endpoint) or "key" (ISV endpoint).
	Key string `json:"key"`
}

// GetWebhookVerificationKey fetches the key a merchant's webhook endpoint must return, as
// JSON, from a GET on the webhook URL. Viva does not activate a webhook until that GET does.
// Ref: https://developer.viva.com/apis-for-payments/payment-api/#tag/Webhooks/paths/~1api~1messages~1config~1token/get
func (c BasicAuthClient) GetWebhookVerificationKey() (*WebhookVerificationKey, error) {
	key := &WebhookVerificationKey{}
	if err := c.Get(webhookKeyUri(c.Config), key); err != nil {
		return nil, err
	}
	return key, nil
}

func webhookKeyUri(c Config) string {
	return fmt.Sprintf("%s/api/messages/config/token", AppUri(c))
}

// GetWebhookVerificationKey fetches the key an ISV-level webhook endpoint must return, as
// JSON, from a GET on the webhook URL. ISV webhooks cover every connected merchant; Viva
// recommends not also configuring merchant-level webhooks.
// Ref: https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Webhooks/paths/~1isv~1v1~1webhooks~1token/get
func (c *ISVClient) GetWebhookVerificationKey() (*WebhookVerificationKey, error) {
	key := &WebhookVerificationKey{}
	if err := c.Get(isvWebhookKeyUri(c.Config), key); err != nil {
		return nil, err
	}
	return key, nil
}

func isvWebhookKeyUri(c Config) string {
	return fmt.Sprintf("%s/isv/v1/webhooks/token", ApiUri(c))
}
