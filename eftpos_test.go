package vivawallet

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// ecrRequest is what ecrServer records of the one request it receives.
type ecrRequest struct {
	Method     string
	RequestURI string
	Auth       string
	Body       []byte
}

// ecrServer records the one request it receives into got and answers it with body.
func ecrServer(t *testing.T, body string, got *ecrRequest) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Method = r.Method
		got.RequestURI = r.RequestURI
		got.Auth = r.Header.Get("Authorization")
		got.Body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestECRSale(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, "", &req)

	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	err := c.Sale(ECRSaleRequest{
		SessionID:             "viva321",
		TerminalID:            "16013000",
		CashRegisterID:        "tameio1",
		Amount:                734,
		CurrencyCode:          ECRCurrencyEUR,
		MerchantReference:     "bookingnumberabc",
		CustomerTrns:          "paymentforcustomer",
		ShowTransactionResult: true,
		ShowReceipt:           true,
		ISVDetails:            &ECRISVDetails{Amount: 10, TerminalMerchantID: "merchant-1"},
	})
	if err != nil {
		t.Fatalf("Sale() error = %v", err)
	}

	if req.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", req.Method)
	}
	// RequestURI is the path exactly as sent, so this also catches ":" being percent-encoded.
	if req.RequestURI != "/ecr/isv/v1/transactions:sale" {
		t.Errorf("request URI = %s, want /ecr/isv/v1/transactions:sale", req.RequestURI)
	}
	if got := req.Auth; got != "Bearer ecr-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer ecr-token")
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(req.Body, &sent); err != nil {
		t.Fatalf("request body was not valid JSON: %v", err)
	}
	// currencyCode is a string here, unlike the ISV order API's number.
	if sent["currencyCode"] != "978" {
		t.Errorf("sent currencyCode = %#v, want string \"978\"", sent["currencyCode"])
	}
	if sent["sessionId"] != "viva321" || sent["terminalId"] != "16013000" || sent["cashRegisterId"] != "tameio1" {
		t.Errorf("sent ids = %v/%v/%v", sent["sessionId"], sent["terminalId"], sent["cashRegisterId"])
	}
	isv, _ := sent["isvDetails"].(map[string]interface{})
	if isv["amount"] != float64(10) || isv["terminalMerchantId"] != "merchant-1" {
		t.Errorf("sent isvDetails = %#v", sent["isvDetails"])
	}
}

// TestECRSaleTipIsSeparate pins that the tip is sent beside the amount, never folded into it:
// Viva adds the two, so folding would charge the tip twice.
func TestECRSaleTipIsSeparate(t *testing.T) {
	data, err := json.Marshal(ECRSaleRequest{Amount: 10000, TipAmount: 1000})
	if err != nil {
		t.Fatal(err)
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(data, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["amount"] != float64(10000) {
		t.Errorf("amount = %v, want 10000", sent["amount"])
	}
	if sent["tipAmount"] != float64(1000) {
		t.Errorf("tipAmount = %v, want 1000", sent["tipAmount"])
	}
	if _, ok := sent["isvDetails"]; ok {
		t.Error("isvDetails sent although unset, want it omitted")
	}
}

func TestECRRefund(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, `{}`, &req)

	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	err := c.Refund(ECRRefundRequest{
		SessionID:         "refund-1",
		TerminalID:        "16013000",
		CashRegisterID:    "tameio1",
		Amount:            734,
		CurrencyCode:      ECRCurrencyEUR,
		MerchantReference: "bookingnumberabc",
	})
	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}

	if req.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", req.Method)
	}
	if req.RequestURI != "/ecr/isv/v1/transactions:refund" {
		t.Errorf("request URI = %s, want /ecr/isv/v1/transactions:refund", req.RequestURI)
	}
	if got := req.Auth; got != "Bearer ecr-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer ecr-token")
	}
}

func TestECRGetSession(t *testing.T) {
	fixture, err := os.ReadFile("testdata/ecr_session_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var req ecrRequest
	srv := ecrServer(t, string(fixture), &req)

	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	session, err := c.GetSession("16304888_11369")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", req.Method)
	}
	if req.RequestURI != "/ecr/isv/v1/sessions/16304888_11369" {
		t.Errorf("request URI = %s, want /ecr/isv/v1/sessions/16304888_11369", req.RequestURI)
	}
	if got := req.Auth; got != "Bearer ecr-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer ecr-token")
	}

	if session.SessionID != "16304888_11369" || session.TID != "16304888" || session.CashRegisterID != "SH00001455" {
		t.Errorf("session ids = %+v", session)
	}
	if session.Amount != 16074 {
		t.Errorf("Amount = %d, want 16074", session.Amount)
	}
	if session.Success || session.Message != "Invalid card" {
		t.Errorf("Success/Message = %v/%q, want false/\"Invalid card\"", session.Success, session.Message)
	}
	if session.OrderCode != "0" || session.ShortOrderCode != "0" {
		t.Errorf("OrderCode/ShortOrderCode = %q/%q, want \"0\"/\"0\"", session.OrderCode, session.ShortOrderCode)
	}
	if session.TransactionID != "" || session.TransactionDateTime != "" {
		t.Errorf("null fields decoded as %q/%q, want empty", session.TransactionID, session.TransactionDateTime)
	}
}

func TestECRSessionUriEscapesSessionID(t *testing.T) {
	got := ecrSessionUri(Config{Demo: true}, "a/b c")
	want := "https://demo-api.vivapayments.com/ecr/isv/v1/sessions/a%2Fb%20c"
	if got != want {
		t.Errorf("ecrSessionUri() = %q, want %q", got, want)
	}
}

func TestNewSessionID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := NewSessionID()
		if err != nil {
			t.Fatalf("NewSessionID() error = %v", err)
		}
		if len(id) != 32 {
			t.Errorf("NewSessionID() = %q, want 32 hex characters", id)
		}
		if seen[id] {
			t.Fatalf("NewSessionID() repeated %q", id)
		}
		seen[id] = true
	}
}
