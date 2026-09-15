package vivawallet

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetTransaction(t *testing.T) {
	fixture, err := os.ReadFile("testdata/get_transaction_response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/checkout/v2/transactions/trx-1" {
			t.Errorf("path = %s, want /checkout/v2/transactions/trx-1", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fresh-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer fresh-token")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)
	c.SetToken("fresh-token", futureExpiry())

	trx, err := c.GetTransaction("trx-1")
	if err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}
	if trx.OrderCode != 5099296005794370 {
		t.Errorf("OrderCode = %d, want 5099296005794370", trx.OrderCode)
	}
	if trx.Email != "customer@example.com" {
		t.Errorf("Email = %q, want customer@example.com", trx.Email)
	}
	if trx.CardTypeID != 1 {
		t.Errorf("CardTypeID = %d, want 1", trx.CardTypeID)
	}
}

func TestCancelPartialAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/acquiring/v1/transactions/trx-1" {
			t.Errorf("path = %s, want /acquiring/v1/transactions/trx-1", r.URL.Path)
		}
		if got := r.URL.Query().Get("amount"); got != "500" {
			t.Errorf("amount query param = %q, want 500", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"ok"`))
	}))
	defer srv.Close()

	c := newOAuthTestClient(t, srv)
	c.SetToken("fresh-token", futureExpiry())

	if err := c.CancelPartialAuthorization("trx-1", 500, ""); err != nil {
		t.Fatalf("CancelPartialAuthorization() error = %v", err)
	}
}
