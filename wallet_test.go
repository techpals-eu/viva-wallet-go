package vivawallet

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetWallets(t *testing.T) {
	fixture, err := os.ReadFile("testdata/get_wallets_response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/wallets" {
			t.Errorf("path = %s, want /api/wallets", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newBasicAuthTestClient(t, srv)

	wallets, err := c.GetWallets()
	if err != nil {
		t.Fatalf("GetWallets() error = %v", err)
	}
	if len(wallets) != 1 {
		t.Fatalf("len(wallets) = %d, want 1", len(wallets))
	}
	if wallets[0].WalletID != 1 {
		t.Errorf("WalletID = %d, want 1", wallets[0].WalletID)
	}
	if wallets[0].IBAN != "GR1601101250000000012300695" {
		t.Errorf("IBAN = %q, want GR1601101250000000012300695", wallets[0].IBAN)
	}
}

func TestBalanceTransfer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/wallets/w1/balancetransfer/w2" {
			t.Errorf("path = %s, want /api/wallets/w1/balancetransfer/w2", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"DebitTransactionId":"d1","CreditTransactionId":"c1"}`))
	}))
	defer srv.Close()

	c := newBasicAuthTestClient(t, srv)

	resp, err := c.BalanceTranfer("w1", "w2", BalanceTransfer{Amount: 100, Description: "test"})
	if err != nil {
		t.Fatalf("BalanceTranfer() error = %v", err)
	}
	if resp.DebitTransactionID != "d1" {
		t.Errorf("DebitTransactionID = %q, want d1", resp.DebitTransactionID)
	}
	if resp.CreditTransactionID != "c1" {
		t.Errorf("CreditTransactionID = %q, want c1", resp.CreditTransactionID)
	}
}
