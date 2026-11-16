package vivawallet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func gr111Receipt(t *testing.T) FiscalisationData {
	t.Helper()
	var d FiscalisationData
	if err := json.Unmarshal(readFixture(t, "testdata/fiscal_gr_receipt_11_1.json"), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestISVSignReceipt(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, string(readFixture(t, "testdata/fiscal_session_details.json")), &req)
	c := newISVTestClient(t, srv)
	c.SetToken("fiscal-token", futureExpiry())

	const sessionID = "9e346121-cb51-4234-96cb-7c347bfe05d1"
	res, err := c.SignReceipt("merchant-uuid", sessionID, gr111Receipt(t))
	if err != nil {
		t.Fatalf("SignReceipt() error = %v", err)
	}

	if req.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", req.Method)
	}
	// RequestURI is the path exactly as sent, so this also catches ":" being percent-encoded.
	if want := "/fiscalization/v1/isv/receipts:sign?merchantId=merchant-uuid"; req.RequestURI != want {
		t.Errorf("request URI = %s, want %s", req.RequestURI, want)
	}
	if req.Auth != "Bearer fiscal-token" {
		t.Errorf("Authorization = %q, want bearer token", req.Auth)
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(req.Body, &sent); err != nil {
		t.Fatalf("request body is not valid JSON: %v\n%s", err, req.Body)
	}
	if sent["sessionId"] != sessionID {
		t.Errorf("sent sessionId = %#v, want %s", sent["sessionId"], sessionID)
	}
	if sent["cbReceiptReference"] != "testreceiptreference1" {
		t.Errorf("sent cbReceiptReference = %#v", sent["cbReceiptReference"])
	}
	// Decoded as float64, which cannot hold the case exactly, so compare the raw text.
	if !strings.Contains(string(req.Body), `"ftReceiptCase":5139205309155770369`) {
		t.Errorf("ftReceiptCase not sent as an exact decimal number:\n%s", req.Body)
	}

	if res.Mark() != "400001948544829" || res.Transmission() != TransmissionOK {
		t.Errorf("result = mark %q, transmission %v", res.Mark(), res.Transmission())
	}
}

func TestISVSignReceiptEscapesMerchantID(t *testing.T) {
	got := isvSignReceiptUri(Config{Demo: true}, "a&b=c")
	if want := "https://demo-api.vivapayments.com/fiscalization/v1/isv/receipts:sign?merchantId=a%26b%3Dc"; got != want {
		t.Errorf("uri = %s, want %s", got, want)
	}
}

func TestSignReceiptHosts(t *testing.T) {
	if got := signReceiptUri(Config{Demo: true}); got != "https://demo-api.vivapayments.com/fiscalization/v1/receipts:sign" {
		t.Errorf("demo = %s", got)
	}
	if got := signReceiptUri(Config{}); got != "https://api.vivapayments.com/fiscalization/v1/receipts:sign" {
		t.Errorf("production = %s", got)
	}
}

func TestSignReceiptRequiresIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL)
	}))
	defer srv.Close()
	c := newISVTestClient(t, srv)
	c.SetToken("fiscal-token", futureExpiry())

	if _, err := c.SignReceipt("merchant-uuid", "", gr111Receipt(t)); err == nil {
		t.Error("SignReceipt() with no session id: want error")
	}
	if _, err := c.SignReceipt("", "9e346121-cb51-4234-96cb-7c347bfe05d1", gr111Receipt(t)); err == nil {
		t.Error("SignReceipt() with no merchant id: want error")
	}
}

func TestOAuthSignReceipt(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, string(readFixture(t, "testdata/fiscal_sign_response_zero.json")), &req)
	c := newOAuthTestClient(t, srv)
	c.SetToken("merchant-token", futureExpiry())

	res, err := c.SignReceipt("9e346121-cb51-4234-96cb-7c347bfe05d1", gr111Receipt(t))
	if err != nil {
		t.Fatalf("SignReceipt() error = %v", err)
	}
	if req.RequestURI != "/fiscalization/v1/receipts:sign" {
		t.Errorf("request URI = %s", req.RequestURI)
	}
	if req.Auth != "Bearer merchant-token" {
		t.Errorf("Authorization = %q", req.Auth)
	}
	if !res.State.Ready() {
		t.Errorf("State = %s, want ready", res.State)
	}
}

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestQueueStartAndZeroReceipts(t *testing.T) {
	tests := []struct {
		name string
		call func(*ISVClient) (*FiscalResult, error)
		want ReceiptCase
	}{
		{"StartFiscalQueue", func(c *ISVClient) (*FiscalResult, error) { return c.StartFiscalQueue("m1") }, GRReceiptCase(GRReceiptQueueStart, 0)},
		{"ZeroReceipt", func(c *ISVClient) (*FiscalResult, error) { return c.ZeroReceipt("m1") }, GRReceiptCase(GRReceiptZero, 0)},
	}
	for _, tt := range tests {
		var req ecrRequest
		srv := ecrServer(t, string(readFixture(t, "testdata/fiscal_sign_response_zero.json")), &req)
		c := newISVTestClient(t, srv)
		c.SetToken("fiscal-token", futureExpiry())

		if _, err := tt.call(c); err != nil {
			t.Fatalf("%s() error = %v", tt.name, err)
		}

		var sent struct {
			SessionID    string        `json:"sessionId"`
			Reference    string        `json:"cbReceiptReference"`
			Moment       string        `json:"cbReceiptMoment"`
			ReceiptCase  ReceiptCase   `json:"ftReceiptCase"`
			ChargeItems  []interface{} `json:"cbChargeItems"`
			PayItems     []interface{} `json:"cbPayItems"`
			HasCaseData  *interface{}  `json:"ftReceiptCaseData"`
			CurrencyCode string        `json:"currencyCode"`
		}
		if err := json.Unmarshal(req.Body, &sent); err != nil {
			t.Fatalf("%s: body: %v", tt.name, err)
		}
		if sent.ReceiptCase != tt.want {
			t.Errorf("%s: ftReceiptCase = %d, want %d", tt.name, sent.ReceiptCase, tt.want)
		}
		if !uuidV4.MatchString(sent.SessionID) {
			t.Errorf("%s: sessionId %q is not a UUID v4", tt.name, sent.SessionID)
		}
		if sent.Reference == "" || !strings.HasSuffix(sent.Reference, sent.SessionID[:8]) {
			t.Errorf("%s: cbReceiptReference %q is not unique per call", tt.name, sent.Reference)
		}
		if _, err := time.Parse(fiscalMomentLayout, sent.Moment); err != nil {
			t.Errorf("%s: cbReceiptMoment %q: %v", tt.name, sent.Moment, err)
		}
		if sent.ChargeItems == nil || sent.PayItems == nil || len(sent.ChargeItems)+len(sent.PayItems) != 0 {
			t.Errorf("%s: items = %v / %v, want [] / []", tt.name, sent.ChargeItems, sent.PayItems)
		}
		if sent.HasCaseData != nil || sent.CurrencyCode != "978" {
			t.Errorf("%s: unexpected ftReceiptCaseData or currencyCode %q", tt.name, sent.CurrencyCode)
		}
	}
}

func TestGROperationReceiptMatchesQueueStartSample(t *testing.T) {
	d := grOperationReceipt(GRReceiptQueueStart, "queue-start", "abcdef12-0000", time.Date(2025, 8, 13, 11, 30, 4, 0, time.UTC))
	d.CBReceiptReference = "Start-receipt-time-13Aug-2025-11:30" // the sample's own reference
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, got, "testdata/fiscal_gr_queue_start.json")
}

func TestFiscalErrorHelpers(t *testing.T) {
	notStarted := `{"message":"FiskaltrustSignReceiptUnknownState","eventId":0}`
	notEnabled := `{"message":"FiskaltrustInvalidMerchantOrTerminal","eventId":22254}`

	tests := []struct {
		name                   string
		status                 int
		body                   string
		wantNotStarted, wantNE bool
	}{
		{"queue not started", http.StatusBadRequest, notStarted, true, false},
		{"not enabled", http.StatusNotFound, notEnabled, false, true},
		{"not enabled, eventId only", http.StatusNotFound, `{"eventId":22254}`, false, true},
		{"other 404", http.StatusNotFound, `{"message":"no such order"}`, false, false},
		{"eventId but not 404", http.StatusBadRequest, `{"eventId":22254}`, false, false},
	}
	for _, tt := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
			_, _ = w.Write([]byte(tt.body))
		}))
		c := newISVTestClient(t, srv)
		c.SetToken("fiscal-token", futureExpiry())

		_, err := c.SignReceipt("m1", "9e346121-cb51-4234-96cb-7c347bfe05d1", gr111Receipt(t))
		srv.Close()
		if err == nil {
			t.Fatalf("%s: want error", tt.name)
		}
		if got := IsFiscalQueueNotStarted(err); got != tt.wantNotStarted {
			t.Errorf("%s: IsFiscalQueueNotStarted = %v, want %v", tt.name, got, tt.wantNotStarted)
		}
		if got := IsFiscalisationNotEnabled(err); got != tt.wantNE {
			t.Errorf("%s: IsFiscalisationNotEnabled = %v, want %v", tt.name, got, tt.wantNE)
		}
	}
	if IsFiscalQueueNotStarted(nil) || IsFiscalisationNotEnabled(nil) {
		t.Error("nil error matched")
	}
}

func TestECRSaleWithFiscalisationData(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, "", &req)
	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	receipt := gr111Receipt(t)
	receipt.CBPayItems[0].FTPayItemCase = GRPayItemCase(GRPaymentEFTPOS, 0)
	err := c.Sale(ECRSaleRequest{
		SessionID: "s1", TerminalID: "16013000", CashRegisterID: "tameio1",
		Amount: 124, TipAmount: 100, CurrencyCode: ECRCurrencyEUR,
		MerchantReference: "booking1", CustomerTrns: "room 12",
		FiscalisationData: &receipt,
	})
	if err != nil {
		t.Fatalf("Sale() error = %v", err)
	}

	var sent struct {
		Amount            int64           `json:"amount"`
		TipAmount         int64           `json:"tipAmount"`
		FiscalisationData json.RawMessage `json:"fiscalisationData"`
	}
	if err := json.Unmarshal(req.Body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Amount != 124 || sent.TipAmount != 100 {
		t.Errorf("amount/tip = %d/%d, want 124/100", sent.Amount, sent.TipAmount)
	}
	want, _ := json.Marshal(receipt)
	if string(sent.FiscalisationData) != string(want) {
		t.Errorf("fiscalisationData =\n%s\nwant\n%s", sent.FiscalisationData, want)
	}
	if !strings.Contains(string(sent.FiscalisationData), `"ftPayItemCase":5139205309155246082`) {
		t.Error("EFTPOS pay item case not sent")
	}
}

func TestECRWithoutFiscalisationDataOmitsIt(t *testing.T) {
	for name, payload := range map[string]interface{}{
		"sale":   ECRSaleRequest{SessionID: "s1"},
		"refund": ECRRefundRequest{SessionID: "s1"},
	} {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "fiscalisationData") {
			t.Errorf("%s: fiscalisationData sent when unset: %s", name, b)
		}
	}
}

func TestECRRefundWithFiscalisationData(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, "", &req)
	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	receipt := FiscalisationData{
		CBReceiptReference: "refund-1",
		FTReceiptCase:      GRReceiptCase(GRReceiptPOS, ReceiptFlagRefund|ReceiptFlagCustomNumbered),
	}
	if err := c.Refund(ECRRefundRequest{SessionID: "s2", Amount: 124, FiscalisationData: &receipt}); err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if req.RequestURI != "/ecr/isv/v1/transactions:refund" {
		t.Errorf("request URI = %s", req.RequestURI)
	}
	if !strings.Contains(string(req.Body), `"fiscalisationData":{"cbChargeItems":[],"cbPayItems":[]`) ||
		!strings.Contains(string(req.Body), `"ftReceiptCase":5139205309172547585`) {
		t.Errorf("refund body missing fiscalisationData:\n%s", req.Body)
	}
}

func TestECRGetSessionWithFiscalisationDetails(t *testing.T) {
	var session map[string]json.RawMessage
	if err := json.Unmarshal(readFixture(t, "testdata/ecr_session_response.json"), &session); err != nil {
		t.Fatal(err)
	}
	session["fiscalisationDetails"] = readFixture(t, "testdata/fiscal_session_details.json")
	body, _ := json.Marshal(session)

	var req ecrRequest
	srv := ecrServer(t, string(body), &req)
	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	got, err := c.GetSession("16304888_11369")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	d := got.FiscalisationDetails
	if d == nil {
		t.Fatal("FiscalisationDetails = nil")
	}
	if !d.State.Ready() || d.Mark() != "400001948544829" || d.Transmission() != TransmissionOK || len(d.Signatures) != 6 {
		t.Errorf("FiscalisationDetails = %+v", d)
	}
}

func TestECRGetSessionWithoutFiscalisation(t *testing.T) {
	var req ecrRequest
	srv := ecrServer(t, string(readFixture(t, "testdata/ecr_session_response.json")), &req)
	c := newISVTestClient(t, srv)
	c.SetToken("ecr-token", futureExpiry())

	got, err := c.GetSession("16304888_11369")
	if err != nil {
		t.Fatal(err)
	}
	if got.FiscalisationDetails != nil {
		t.Errorf("FiscalisationDetails = %+v, want nil when the merchant is not fiscalised", got.FiscalisationDetails)
	}
}
