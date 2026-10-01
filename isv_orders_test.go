package vivawallet

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCreateISVOrder(t *testing.T) {
	fixture, err := os.ReadFile("testdata/checkout_order_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/checkout/v2/isv/orders" {
			t.Errorf("path = %s, want /checkout/v2/isv/orders", r.URL.Path)
		}
		if got := r.URL.Query().Get("merchantId"); got != "merchant-1" {
			t.Errorf("merchantId = %q, want %q", got, "merchant-1")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer isv-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer isv-token")
		}

		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)
	c.SetToken("isv-token", futureExpiry())

	resp, err := c.CreateISVOrder("merchant-1", ISVCheckoutOrder{
		Amount:               1000,
		ISVAmount:            50,
		CustomerTransactions: "Πληρωμή καφέ",
		Customer:             &Customer{Email: "a@example.com", RequestLang: "el-GR"},
		CurrencyCode:         CurrencyEUR,
		PaymentTimeout:       600,
		DisableCash:          true,
	})
	if err != nil {
		t.Fatalf("CreateISVOrder() error = %v", err)
	}
	if resp.OrderCode != 5099296005794370 {
		t.Errorf("OrderCode = %d, want 5099296005794370", resp.OrderCode)
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body was not valid JSON: %v", err)
	}
	// currencyCode must go out as a JSON number, not the string the EFT POS API takes.
	if sent["currencyCode"] != float64(978) {
		t.Errorf("sent currencyCode = %#v, want number 978", sent["currencyCode"])
	}
	if sent["isvAmount"] != float64(50) {
		t.Errorf("sent isvAmount = %#v, want number 50", sent["isvAmount"])
	}
	if sent["amount"] != float64(1000) {
		t.Errorf("sent amount = %#v, want 1000", sent["amount"])
	}
	if sent["paymentTimeout"] != float64(600) || sent["disableCash"] != true {
		t.Errorf("sent paymentTimeout/disableCash = %#v/%#v, want 600/true", sent["paymentTimeout"], sent["disableCash"])
	}
	customer, _ := sent["customer"].(map[string]interface{})
	if customer["email"] != "a@example.com" || customer["requestLang"] != "el-GR" {
		t.Errorf("sent customer = %#v", sent["customer"])
	}
	if _, ok := customer["fullName"]; ok {
		t.Error("sent customer carries an empty fullName, want it omitted")
	}
}

func TestISVCheckoutOrderOmitsUnsetFields(t *testing.T) {
	data, err := json.Marshal(ISVCheckoutOrder{Amount: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != `{"amount":1000}` {
		t.Errorf("marshalled = %s, want {\"amount\":1000}", got)
	}
}

// TestCreateISVOrderRejectsISVAmountAboveAmount pins the choice to validate client-side: an
// order whose commission exceeds its total never reaches the API.
func TestCreateISVOrderRejectsISVAmountAboveAmount(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orderCode":1}`))
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)
	c.SetToken("isv-token", futureExpiry())

	_, err := c.CreateISVOrder("merchant-1", ISVCheckoutOrder{Amount: 100, ISVAmount: 101})
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("CreateISVOrder() error = %v, want an isv amount error", err)
	}
	if hits != 0 {
		t.Errorf("server hit %d times, want 0", hits)
	}

	// A commission equal to the amount is allowed through.
	if _, err := c.CreateISVOrder("merchant-1", ISVCheckoutOrder{Amount: 100, ISVAmount: 100}); err != nil {
		t.Fatalf("CreateISVOrder() with isvAmount == amount error = %v", err)
	}
	if hits != 1 {
		t.Errorf("server hit %d times, want 1", hits)
	}
}

func TestGetISVOrder(t *testing.T) {
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
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("reseller-id:merchant-1:reseller-api-key"))
		if got := r.Header.Get("Authorization"); got != want {
			t.Errorf("Authorization = %q, want ISV basic auth for merchant-1", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)

	op, err := c.GetISVOrder("merchant-1", 5099296005794370)
	if err != nil {
		t.Fatalf("GetISVOrder() error = %v", err)
	}
	if op.StateID != OrderStatePaid {
		t.Errorf("StateID = %v, want OrderStatePaid", op.StateID)
	}
}

// TestISVOrderHosts guards the ApiUri/AppUri split: creation is a /checkout/v2 call on the
// api host, retrieval an /api/* call on the app host.
func TestISVOrderHosts(t *testing.T) {
	demo := Config{Demo: true}
	if got, want := createISVOrderUri(demo, "m 1"), "https://demo-api.vivapayments.com/checkout/v2/isv/orders?merchantId=m+1"; got != want {
		t.Errorf("createISVOrderUri() = %q, want %q", got, want)
	}
	if got, want := getISVOrderUri(demo, 42), "https://demo.vivapayments.com/api/orders/42"; got != want {
		t.Errorf("getISVOrderUri() = %q, want %q", got, want)
	}
}

func TestSmartCheckoutURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		opts []CheckoutURLOption
		want string
	}{
		{"demo, no color", Config{Demo: true}, nil, "https://demo.vivapayments.com/web2?ref=5522243373579129"},
		{"production, no color", Config{}, nil, "https://www.vivapayments.com/web2?ref=5522243373579129"},
		{"color without #", Config{Demo: true}, []CheckoutURLOption{WithColor("d5791a")}, "https://demo.vivapayments.com/web2?color=d5791a&ref=5522243373579129"},
		{"color with #", Config{Demo: true}, []CheckoutURLOption{WithColor("#d5791a")}, "https://demo.vivapayments.com/web2?color=d5791a&ref=5522243373579129"},
		{"empty color", Config{Demo: true}, []CheckoutURLOption{WithColor("")}, "https://demo.vivapayments.com/web2?ref=5522243373579129"},
		{"bare # color", Config{Demo: true}, []CheckoutURLOption{WithColor("#")}, "https://demo.vivapayments.com/web2?ref=5522243373579129"},
		{"nil option", Config{Demo: true}, []CheckoutURLOption{nil}, "https://demo.vivapayments.com/web2?ref=5522243373579129"},
		{"later color wins", Config{Demo: true}, []CheckoutURLOption{WithColor("000000"), WithColor("ffffff")}, "https://demo.vivapayments.com/web2?color=ffffff&ref=5522243373579129"},
		{"app base override", Config{appBase: "http://127.0.0.1:1"}, nil, "http://127.0.0.1:1/web2?ref=5522243373579129"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SmartCheckoutURL(tt.cfg, 5522243373579129, tt.opts...)
			if got != tt.want {
				t.Errorf("SmartCheckoutURL() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "#") {
				t.Errorf("SmartCheckoutURL() = %q contains a fragment delimiter", got)
			}
		})
	}
}
