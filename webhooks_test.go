package vivawallet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func parseFixture(t *testing.T, name string) *WebhookEvent {
	t.Helper()

	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	event, err := ParseWebhook(body)
	if err != nil {
		t.Fatalf("ParseWebhook(%s) error = %v", name, err)
	}
	return event
}

func TestWebhookTransactionPaymentCreated(t *testing.T) {
	event := parseFixture(t, "webhook_transaction_payment_created.json")

	if event.EventTypeID != EventTransactionPaymentCreated {
		t.Errorf("EventTypeID = %d, want %d", event.EventTypeID, EventTransactionPaymentCreated)
	}
	if event.MessageID != "e8b09fc2-d4a4-43fc-8251-acd87ad04d96" {
		t.Errorf("MessageID = %q", event.MessageID)
	}
	if event.Created != "2021-09-02T07:39:28.8496795Z" || event.CorrelationID != "21-245-DB33F8C9" {
		t.Errorf("Created/CorrelationID = %q/%q", event.Created, event.CorrelationID)
	}
	if event.RecipientID != "bdf4c6b3-c26d-4046-b5df-5c443ec39d09" || event.MessageTypeID != 512 {
		t.Errorf("RecipientID/MessageTypeID = %q/%d", event.RecipientID, event.MessageTypeID)
	}

	data, err := event.TransactionPaymentCreated()
	if err != nil {
		t.Fatalf("TransactionPaymentCreated() error = %v", err)
	}
	if data.TransactionID != "997ab1e3-e6ce-45c9-970d-4d902f27ce71" {
		t.Errorf("TransactionID = %q", data.TransactionID)
	}
	if data.OrderCode != 2271655739472609 {
		t.Errorf("OrderCode = %d", data.OrderCode)
	}
	if data.StatusID != TransactionStatusFinished {
		t.Errorf("StatusID = %q, want F", data.StatusID)
	}
	if data.Amount != 10 || data.CurrencyCode != "978" {
		t.Errorf("Amount/CurrencyCode = %v/%q", data.Amount, data.CurrencyCode)
	}
	if data.ElectronicCommerceIndicator != "5" {
		t.Errorf("ElectronicCommerceIndicator = %q, want 5", data.ElectronicCommerceIndicator)
	}
	if data.TerminalID != 90000000 || data.ReferenceNumber != 944981 {
		t.Errorf("TerminalID/ReferenceNumber = %d/%d", data.TerminalID, data.ReferenceNumber)
	}
	if len(data.Tags) != 3 || data.InsDate != "2021-09-02T10:39:28.73" {
		t.Errorf("Tags/InsDate = %v/%q", data.Tags, data.InsDate)
	}
}

func TestWebhookTransactionReversalCreated(t *testing.T) {
	data, err := parseFixture(t, "webhook_transaction_reversal_created.json").TransactionReversalCreated()
	if err != nil {
		t.Fatalf("TransactionReversalCreated() error = %v", err)
	}
	if data.Amount != -1000 {
		t.Errorf("Amount = %v, want -1000", data.Amount)
	}
	if data.ParentID != "d406ca77-f045-4902-9176-506f56fcd793" {
		t.Errorf("ParentID = %q", data.ParentID)
	}
	if data.TransactionTypeID != 7 {
		t.Errorf("TransactionTypeID = %d, want 7", data.TransactionTypeID)
	}
}

// TestWebhookTransactionFailed uses the payload where Viva sends ElectronicCommerceIndicator
// as a number and OriginalAmount with decimals — both of which a naive struct fails on.
func TestWebhookTransactionFailed(t *testing.T) {
	data, err := parseFixture(t, "webhook_transaction_failed.json").TransactionFailed()
	if err != nil {
		t.Fatalf("TransactionFailed() error = %v", err)
	}
	if data.StatusID != TransactionStatusError {
		t.Errorf("StatusID = %q, want E", data.StatusID)
	}
	if data.ElectronicCommerceIndicator != "5" {
		t.Errorf("ElectronicCommerceIndicator = %q, want 5", data.ElectronicCommerceIndicator)
	}
	if data.OriginalAmount != 99.06 {
		t.Errorf("OriginalAmount = %v, want 99.06", data.OriginalAmount)
	}
	if data.ResponseEventID == nil || *data.ResponseEventID != 310 {
		t.Errorf("ResponseEventID = %v, want 310", data.ResponseEventID)
	}
	if data.DigitalWalletID != nil {
		t.Errorf("DigitalWalletID = %v, want nil", *data.DigitalWalletID)
	}
}

func TestWebhookAccountConnected(t *testing.T) {
	data, err := parseFixture(t, "webhook_account_connected.json").AccountConnected()
	if err != nil {
		t.Fatalf("AccountConnected() error = %v", err)
	}
	if data.WalletID != 3059898763296 || data.ConnectedAccountID != "592d2e2f-11ec-408b-8552-ea06af1d1eee" {
		t.Errorf("data = %+v", data)
	}
}

func TestWebhookAccountVerificationStatusChanged(t *testing.T) {
	data, err := parseFixture(t, "webhook_account_verification_status_changed.json").AccountVerificationStatusChanged()
	if err != nil {
		t.Fatalf("AccountVerificationStatusChanged() error = %v", err)
	}
	if data.Verified || data.PersonID != "1aabc6f0-b419-4be5-b898-12345f4c912f" {
		t.Errorf("data = %+v", data)
	}
}

// TestParseWebhookUnknownEventType: an event the SDK has no type for must still parse, or a
// webhook endpoint would fail it and Viva would keep retrying.
func TestParseWebhookUnknownEventType(t *testing.T) {
	event, err := ParseWebhook([]byte(`{"EventTypeId": 1802, "MessageId": "m-1", "EventData": {"Anything": [1, "two"]}}`))
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v, want nil for an unknown event type", err)
	}
	if event.EventTypeID != 1802 || event.MessageID != "m-1" {
		t.Errorf("event = %+v", event)
	}
}

func TestParseWebhookMalformed(t *testing.T) {
	if _, err := ParseWebhook([]byte(`{"EventTypeId": `)); err == nil {
		t.Fatal("ParseWebhook() error = nil, want error for malformed JSON")
	}
}

// TestWebhookAccessorWrongEventType: calling an accessor for another event type must fail
// rather than hand back a zero-valued struct that looks like real data.
func TestWebhookAccessorWrongEventType(t *testing.T) {
	event := parseFixture(t, "webhook_account_connected.json")

	data, err := event.TransactionPaymentCreated()
	if err == nil {
		t.Fatalf("TransactionPaymentCreated() on an Account Connected event = %+v, want error", data)
	}
	if _, err := event.AccountVerificationStatusChanged(); err == nil {
		t.Fatal("AccountVerificationStatusChanged() on an Account Connected event, want error")
	}
}

func TestGetWebhookVerificationKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/messages/config/token" {
			t.Errorf("request = %s %s, want GET /api/messages/config/token", r.Method, r.URL.Path)
		}
		if u, p, ok := r.BasicAuth(); !ok || u != "merchant-id" || p != "api-key" {
			t.Errorf("basic auth = %q/%q (ok=%v)", u, p, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Key": "B3248222FDCD1885AEAFE51CCC1B5607F00903F6"}`))
	}))
	defer srv.Close()

	key, err := newBasicAuthTestClient(t, srv).GetWebhookVerificationKey()
	if err != nil {
		t.Fatalf("GetWebhookVerificationKey() error = %v", err)
	}
	if key.Key != "B3248222FDCD1885AEAFE51CCC1B5607F00903F6" {
		t.Errorf("Key = %q", key.Key)
	}
}

func TestISVGetWebhookVerificationKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/isv/v1/webhooks/token" {
			t.Errorf("request = %s %s, want GET /isv/v1/webhooks/token", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer isv-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer isv-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key": "C3248222FDCD1885AEAFE51CCC1B5607F00903F6"}`))
	}))
	defer srv.Close()

	c := newISVTestClient(t, srv)
	c.SetToken("isv-token", futureExpiry())

	key, err := c.GetWebhookVerificationKey()
	if err != nil {
		t.Fatalf("GetWebhookVerificationKey() error = %v", err)
	}
	if key.Key != "C3248222FDCD1885AEAFE51CCC1B5607F00903F6" {
		t.Errorf("Key = %q", key.Key)
	}
}

func TestWebhookKeyHosts(t *testing.T) {
	demo := Config{Demo: true}
	if got, want := webhookKeyUri(demo), "https://demo.vivapayments.com/api/messages/config/token"; got != want {
		t.Errorf("webhookKeyUri() = %q, want %q", got, want)
	}
	if got, want := isvWebhookKeyUri(demo), "https://demo-api.vivapayments.com/isv/v1/webhooks/token"; got != want {
		t.Errorf("isvWebhookKeyUri() = %q, want %q", got, want)
	}

	// The handler echoes the key as lowercase "key", the form Viva's own example returns.
	out, _ := json.Marshal(WebhookVerificationKey{Key: "k"})
	if string(out) != `{"key":"k"}` {
		t.Errorf("marshalled key = %s, want {\"key\":\"k\"}", out)
	}
}
