package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CheckoutOrder struct {
	Amount               int64  `json:"amount"`
	CustomerTransactions string `json:"customerTrns"`
	Customer             struct {
		Email       string `json:"email,omitempty"`
		FullName    string `json:"fullName,omitempty"`
		Phone       string `json:"phone,omitempty"`
		CountryCode string `json:"countryCode,omitempty"`
		RequestLang string `json:"requestLang,omitempty"`
	} `json:"customer,omitempty"`
	PaymentTimeout       int      `json:"paymentTimeout,omitempty"`
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
func (c Client) CreateOrderPayment(payload CheckoutOrder) (*CheckoutOrderResponse, error) {
	uri := checkoutEndpoint(c.Config)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order %s", err)
	}

	// Check if auth expired and if so authenticate again
	if c.HasAuthExpired() {
		_, authErr := c.Authenticate()
		return nil, fmt.Errorf("authentication error %s", authErr)
	}

	req, _ := http.NewRequest("POST", uri, bytes.NewReader(data))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AuthToken()))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to parse order %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to make order %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	response := &CheckoutOrderResponse{}
	if jsonErr := json.Unmarshal(body, response); jsonErr != nil {
		return nil, jsonErr
	}

	return response, nil
}

func checkoutEndpoint(c Config) string {
	return fmt.Sprintf("%s/%s", ApiUri(c), "checkout/v2/orders")
}
