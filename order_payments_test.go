package vivawallet

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCreateOrderPayment(t *testing.T) {
	fixture, err := os.ReadFile("testdata/checkout_order_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/checkout/v2/orders" {
			t.Errorf("path = %s, want /checkout/v2/orders", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fresh-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer fresh-token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)
	c.SetToken("fresh-token", futureExpiry())

	resp, err := c.CreateOrderPayment(CheckoutOrder{Amount: 1000, CustomerTransactions: "ref-1"})
	if err != nil {
		t.Fatalf("CreateOrderPayment() error = %v", err)
	}
	if resp.OrderCode != 5099296005794370 {
		t.Errorf("OrderCode = %d, want 5099296005794370", resp.OrderCode)
	}

	var sentBody map[string]interface{}
	if err := json.Unmarshal(gotBody, &sentBody); err != nil {
		t.Fatalf("request body was not valid JSON: %v", err)
	}
	if sentBody["amount"] != float64(1000) {
		t.Errorf("sent amount = %v, want 1000", sentBody["amount"])
	}
	if sentBody["customerTrns"] != "ref-1" {
		t.Errorf("sent customerTrns = %v, want ref-1", sentBody["customerTrns"])
	}
}

func TestGetOrderPayment(t *testing.T) {
	fixture, err := os.ReadFile("testdata/get_order_payment_response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/orders/5099296005794370" {
			t.Errorf("path = %s, want /api/orders/5099296005794370", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "merchant-id" || password != "api-key" {
			t.Errorf("unexpected basic auth: %q/%q (ok=%v)", username, password, ok)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newBasicAuthTestClient(t, srv)

	resp, err := c.GetOrderPayment(5099296005794370)
	if err != nil {
		t.Fatalf("GetOrderPayment() error = %v", err)
	}
	if resp.OrderCode != 5099296005794370 {
		t.Errorf("OrderCode = %d, want 5099296005794370", resp.OrderCode)
	}
	if resp.SourceCode != "Default" {
		t.Errorf("SourceCode = %q, want Default", resp.SourceCode)
	}
	if len(resp.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 entries", resp.Tags)
	}
}

func TestCancelOrderPaymentPropagatesAPIError(t *testing.T) {
	fixture, err := os.ReadFile("testdata/api_error_response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newBasicAuthTestClient(t, srv)

	_, err = c.CancelOrderPayment(123)
	if err == nil {
		t.Fatal("CancelOrderPayment() error = nil, want error on 400 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.ErrorCode != 1013 {
		t.Errorf("ErrorCode = %d, want 1013", apiErr.ErrorCode)
	}
}
