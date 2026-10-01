package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CurrencyEUR is the ISO 4217 numeric code for the euro, as the ISV order API expects it in
// ISVCheckoutOrder.CurrencyCode.
const CurrencyEUR = 978

// Customer is the customer a payment order is for. Every field is optional; what is set is
// used to prefill Smart Checkout.
type Customer struct {
	Email       string `json:"email,omitempty"`
	FullName    string `json:"fullName,omitempty"`
	Phone       string `json:"phone,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
	// RequestLang is the language Smart Checkout is shown in, e.g. "el-GR".
	RequestLang string `json:"requestLang,omitempty"`
}

// ISVCheckoutOrder is the body of an ISV payment order. It is deliberately separate from
// CheckoutOrder: the ISV-only fields must not appear on ordinary orders, and Viva versions
// the two independently.
type ISVCheckoutOrder struct {
	// Amount is the total the customer is charged, in minor units.
	Amount int64 `json:"amount"`
	// ISVAmount is the ISV's commission, in minor units. It must not exceed Amount;
	// CreateISVOrder rejects such an order before sending it.
	ISVAmount int64 `json:"isvAmount,omitempty"`
	// CustomerTransactions is the description shown on Smart Checkout.
	CustomerTransactions string    `json:"customerTrns,omitempty"`
	Customer             *Customer `json:"customer,omitempty"`
	// CurrencyCode is the ISO 4217 numeric code, e.g. CurrencyEUR. The order API takes it as
	// a number, unlike the transaction APIs which return it as a string.
	CurrencyCode int `json:"currencyCode,omitempty"`
	// PaymentTimeout is how long the order can be paid for, in seconds.
	PaymentTimeout int  `json:"paymentTimeout,omitempty"`
	DisableCash    bool `json:"disableCash,omitempty"`
}

// CreateISVOrder creates a payment order on behalf of merchantID and returns its
// `orderCode`. Redirect the customer to SmartCheckoutURL with that code to take payment.
// Ref: https://developer.viva.com/isv-partner-program/payment-isv-api/
func (c *ISVClient) CreateISVOrder(merchantID string, payload ISVCheckoutOrder) (*CheckoutOrderResponse, error) {
	if payload.ISVAmount > payload.Amount {
		return nil, fmt.Errorf("isv amount %d exceeds order amount %d", payload.ISVAmount, payload.Amount)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order %s", err)
	}

	response := &CheckoutOrderResponse{}
	reqErr := c.Post(createISVOrderUri(c.Config, merchantID), bytes.NewReader(data), response)
	if reqErr != nil {
		return nil, reqErr
	}

	return response, nil
}

func createISVOrderUri(c Config, merchantID string) string {
	return fmt.Sprintf("%s/checkout/v2/isv/orders?merchantId=%s", ApiUri(c), url.QueryEscape(merchantID))
}

// GetISVOrder retrieves a payment order created on behalf of merchantID. Poll it until
// StateID leaves OrderStatePending to learn the outcome of a Smart Checkout payment.
// Ref: https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Payments/paths/~1api~1orders~1{order_code}/get
func (c *ISVClient) GetISVOrder(merchantID string, orderCode int64) (*GetOrderPaymentResponse, error) {
	op := &GetOrderPaymentResponse{}
	reqErr := c.GetWithMerchant(merchantID, getISVOrderUri(c.Config, orderCode), op)
	if reqErr != nil {
		return nil, reqErr
	}

	return op, nil
}

func getISVOrderUri(c Config, orderCode int64) string {
	return fmt.Sprintf("%s/api/orders/%d", AppUri(c), orderCode)
}

type checkoutURLOptions struct {
	color string
}

// CheckoutURLOption customises a Smart Checkout URL built by SmartCheckoutURL.
type CheckoutURLOption func(*checkoutURLOptions)

// WithColor sets the Smart Checkout accent colour as a hex value, with or without a leading
// "#", e.g. "d5791a" or "#d5791a".
func WithColor(hex string) CheckoutURLOption {
	return func(o *checkoutURLOptions) {
		o.color = strings.TrimPrefix(hex, "#")
	}
}

// SmartCheckoutURL returns the URL to redirect a customer to in order to complete payment of
// the order with orderCode.
func SmartCheckoutURL(c Config, orderCode int64, opts ...CheckoutURLOption) string {
	var o checkoutURLOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	q := url.Values{"ref": {fmt.Sprintf("%d", orderCode)}}
	if o.color != "" {
		q.Set("color", o.color)
	}

	return AppUri(c) + "/web2?" + q.Encode()
}
