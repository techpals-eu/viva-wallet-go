package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type CheckoutOrder struct {
	Amount               int64  `json:"amount"`
	CustomerTransactions string `json:"customerTrns,omitempty"`
	Customer             struct {
		Email       string `json:"email,omitempty"`
		FullName    string `json:"fullName,omitempty"`
		Phone       string `json:"phone,omitempty"`
		CountryCode string `json:"countryCode,omitempty"`
		RequestLang string `json:"requestLang,omitempty"`
	} `json:"customer,omitempty"`
	PaymentTimeout       int      `json:"paymentTimeout,omitempty"`
	PreAuth              bool     `json:"preauth,omitempty"`
	AllowRecurring       bool     `json:"allowRecurring,omitempty"`
	MaxInstallments      int      `json:"maxInstallments,omitempty"`
	PaymentNotification  bool     `json:"paymentNotification,omitempty"`
	TipAmount            int64    `json:"tipAmount,omitempty"`
	DisableExactAmount   bool     `json:"disableExactAmount,omitempty"`
	DisableCash          bool     `json:"disableCash,omitempty"`
	DisableWallet        bool     `json:"disableWallet,omitempty"`
	SourceCode           string   `json:"sourceCode,omitempty"`
	MerchantTransactions string   `json:"merchantTrns,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	CardTokens           []string `json:"cardTokens,omitempty"`
}

type CheckoutOrderResponse struct {
	OrderCode int64 `json:"orderCode"`
}

// CreateOrderPayment creates a new order payment and returns the `orderCode`.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments/paths/~1checkout~1v2~1orders/post
func (c OAuthClient) CreateOrderPayment(payload CheckoutOrder) (*CheckoutOrderResponse, error) {
	// Check if auth expired and if so authenticate again
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	uri := checkoutOrderUri(c.Config)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order %s", err)
	}

	response := &CheckoutOrderResponse{}
	reqErr := c.Post(uri, bytes.NewReader(data), &response)
	if reqErr != nil {
		return nil, reqErr
	}

	return response, nil
}

func checkoutOrderUri(c Config) string {
	return fmt.Sprintf("%s/checkout/v2/orders", ApiUri(c))
}

type UpdateOrderPayment struct {
	Amount           int64  `json:"amount"`
	DisablePaidState bool   `json:"disablePaidState,omitempty"`
	ExpirationDate   string `json:"expirationDate,omitempty"`
	IsCancelled      bool   `json:"isCancelled,omitempty"`
}

// UpdareOrderPayment updates a new order payment.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/patch
func (c BasicAuthClient) UpdateOrderPayment(orderCode int64, payload UpdateOrderPayment) error {
	uri := updateOrderUri(c.Config, orderCode)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to parse order %s", err)
	}

	reqErr := c.Patch(uri, bytes.NewReader(data))
	if reqErr != nil {
		return reqErr
	}

	return nil
}

func updateOrderUri(c Config, orderCode int64) string {
	return fmt.Sprintf("%s/api/orders/%d", AppUri(c), orderCode)
}

type GetOrderPaymentResponse struct {
	OrderCode       int64    `json:"OrderCode"`
	SourceCode      string   `json:"SourceCode"`
	Tags            []string `json:"Tags"`
	TipAmount       float64  `json:"TipAmount"`
	RequestLang     string   `json:"RequestLang"`
	MerchantTrns    string   `json:"MerchantTrns"`
	CustomerTrns    string   `json:"CustomerTrns"`
	MaxInstallments float64  `json:"MaxInstallments"`
	RequestAmount   float64  `json:"RequestAmount"`
	ExpirationDate  string   `json:"ExpirationDate"`
	StateID         int      `json:"StateId"`
}

// GetOrderPayment retrieves an order payment
// https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/get
func (c BasicAuthClient) GetOrderPayment(orderCode int64) (*GetOrderPaymentResponse, error) {
	uri := getOrderPaymentUri(c.Config, orderCode)

	op := &GetOrderPaymentResponse{}
	reqErr := c.Get(uri, op)
	if reqErr != nil {
		return nil, reqErr
	}
	return op, nil
}

func getOrderPaymentUri(c Config, orderCode int64) string {
	return fmt.Sprintf("%s/api/orders/%d", AppUri(c), orderCode)
}

type CancelOrderPayment struct {
	OrderCode     int64     `json:"OrderCode"`
	ErrorCode     int       `json:"ErrorCode"`
	ErrorText     string    `json:"ErrorText"`
	TimeStamp     time.Time `json:"TimeStamp"`
	CorrelationID string    `json:"CorrelationId"`
	EventID       int       `json:"EventId"`
	Success       bool      `json:"Success"`
}

// CancelOrderPayment cancels an existing order payment
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/delete
func (c BasicAuthClient) CancelOrderPayment(orderCode int64) (*CancelOrderPayment, error) {
	uri := deleteOrderPaymentUri(c.Config, orderCode)

	result := &CancelOrderPayment{}
	reqErr := c.Delete(uri, nil, result)
	if reqErr != nil {
		return nil, reqErr
	}
	return result, nil
}

func deleteOrderPaymentUri(c Config, orderCode int64) string {
	return fmt.Sprintf("%s/api/orders/%d", AppUri(c), orderCode)
}
