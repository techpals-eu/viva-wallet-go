package ginviva

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeKeys is a KeySource that counts its calls.
type fakeKeys struct {
	calls int32
	err   error
}

func (f *fakeKeys) GetWebhookVerificationKey() (*vivawallet.WebhookVerificationKey, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return nil, f.err
	}
	return &vivawallet.WebhookVerificationKey{Key: "B3248222FDCD1885AEAFE51CCC1B5607F00903F6"}, nil
}

func newRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.Any("/webhooks/viva", h.Handle)
	return r
}

func do(r http.Handler, method, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, "/webhooks/viva", strings.NewReader(body)))
	return w
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGetReturnsKey(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{}))

	w := do(r, http.MethodGet, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got, want := w.Body.String(), `{"key":"B3248222FDCD1885AEAFE51CCC1B5607F00903F6"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestKeyIsFetchedOnce(t *testing.T) {
	keys := &fakeKeys{}
	r := newRouter(NewHandler(keys, Config{}))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if w := do(r, http.MethodGet, ""); w.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", w.Code)
			}
		}()
	}
	wg.Wait()

	if n := atomic.LoadInt32(&keys.calls); n != 1 {
		t.Errorf("key fetched %d times, want 1", n)
	}
}

// TestKeyFetchFailureIsNotCached: a failed fetch answers 502 and is retried on the next GET.
func TestKeyFetchFailureIsNotCached(t *testing.T) {
	keys := &fakeKeys{err: errors.New("viva down")}
	r := newRouter(NewHandler(keys, Config{}))

	if w := do(r, http.MethodGet, ""); w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", w.Code)
	}

	keys.err = nil
	if w := do(r, http.MethodGet, ""); w.Code != http.StatusOK {
		t.Fatalf("status after recovery = %d, want 200", w.Code)
	}
	if n := atomic.LoadInt32(&keys.calls); n != 2 {
		t.Errorf("key fetched %d times, want 2", n)
	}
}

func TestPostInvokesMatchingCallback(t *testing.T) {
	var got *vivawallet.TransactionEventData
	var gotMessageID string
	var otherCalled bool

	r := newRouter(NewHandler(&fakeKeys{}, Config{
		OnPaymentCreated: func(c *gin.Context, d *vivawallet.TransactionEventData) error {
			got = d
			gotMessageID = c.MustGet(EventKey).(*vivawallet.WebhookEvent).MessageID
			return nil
		},
		OnTransactionFailed: func(*gin.Context, *vivawallet.TransactionEventData) error {
			otherCalled = true
			return nil
		},
	}))

	w := do(r, http.MethodPost, fixture(t, "webhook_transaction_payment_created.json"))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got == nil || got.TransactionID != "997ab1e3-e6ce-45c9-970d-4d902f27ce71" {
		t.Fatalf("OnPaymentCreated got %+v", got)
	}
	if gotMessageID != "e8b09fc2-d4a4-43fc-8251-acd87ad04d96" {
		t.Errorf("MessageID via EventKey = %q", gotMessageID)
	}
	if otherCalled {
		t.Error("OnTransactionFailed called for a payment-created event")
	}
}

func TestPostEachEventType(t *testing.T) {
	var called []string
	note := func(name string) { called = append(called, name) }

	r := newRouter(NewHandler(&fakeKeys{}, Config{
		OnPaymentCreated:    func(*gin.Context, *vivawallet.TransactionEventData) error { note("payment"); return nil },
		OnReversalCreated:   func(*gin.Context, *vivawallet.TransactionEventData) error { note("reversal"); return nil },
		OnTransactionFailed: func(*gin.Context, *vivawallet.TransactionEventData) error { note("failed"); return nil },
		OnAccountConnected:  func(*gin.Context, *vivawallet.AccountConnectedData) error { note("connected"); return nil },
		OnAccountVerificationStatusChanged: func(*gin.Context, *vivawallet.AccountVerificationStatusChangedData) error {
			note("verification")
			return nil
		},
	}))

	for _, name := range []string{
		"webhook_transaction_payment_created.json",
		"webhook_transaction_reversal_created.json",
		"webhook_transaction_failed.json",
		"webhook_account_connected.json",
		"webhook_account_verification_status_changed.json",
	} {
		if w := do(r, http.MethodPost, fixture(t, name)); w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", name, w.Code)
		}
	}

	if got, want := strings.Join(called, ","), "payment,reversal,failed,connected,verification"; got != want {
		t.Errorf("callbacks = %s, want %s", got, want)
	}
}

func TestUnknownEventIsAcknowledged(t *testing.T) {
	var unhandled *vivawallet.WebhookEvent
	r := newRouter(NewHandler(&fakeKeys{}, Config{
		OnUnhandled: func(_ *gin.Context, e *vivawallet.WebhookEvent) { unhandled = e },
	}))

	w := do(r, http.MethodPost, `{"EventTypeId": 1802, "MessageId": "m-1", "EventData": {}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if unhandled == nil || unhandled.EventTypeID != 1802 {
		t.Errorf("OnUnhandled got %+v", unhandled)
	}
}

// TestKnownEventWithoutCallbackIsAcknowledged: an event the app did not register for is not
// an error either.
func TestKnownEventWithoutCallbackIsAcknowledged(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{}))

	if w := do(r, http.MethodPost, fixture(t, "webhook_transaction_failed.json")); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestCallbackErrorReturns500(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{
		OnPaymentCreated: func(*gin.Context, *vivawallet.TransactionEventData) error {
			return errors.New("database unavailable")
		},
	}))

	if w := do(r, http.MethodPost, fixture(t, "webhook_transaction_payment_created.json")); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestMalformedBodyReturns400(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{}))

	if w := do(r, http.MethodPost, `{"EventTypeId": `); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestMalformedEventDataReturns400: the envelope parses but the event data does not fit its
// type; redelivering the same body cannot help.
func TestMalformedEventDataReturns400(t *testing.T) {
	var called bool
	r := newRouter(NewHandler(&fakeKeys{}, Config{
		OnPaymentCreated: func(*gin.Context, *vivawallet.TransactionEventData) error { called = true; return nil },
	}))

	w := do(r, http.MethodPost, `{"EventTypeId": 1796, "EventData": {"OrderCode": "not a number"}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if called {
		t.Error("callback ran on undecodable event data")
	}
}

func TestOversizedBodyReturns400(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{MaxBodyBytes: 16}))

	if w := do(r, http.MethodPost, fixture(t, "webhook_account_connected.json")); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestOtherMethodsReturn405(t *testing.T) {
	r := newRouter(NewHandler(&fakeKeys{}, Config{}))

	w := do(r, http.MethodPut, "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
	if got := w.Header().Get("Allow"); got != "GET, POST" {
		t.Errorf("Allow = %q, want %q", got, "GET, POST")
	}
}

// The SDK clients satisfy KeySource.
var (
	_ KeySource = (*vivawallet.BasicAuthClient)(nil)
	_ KeySource = (*vivawallet.ISVClient)(nil)
)
