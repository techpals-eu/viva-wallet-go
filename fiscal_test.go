package vivawallet

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"
)

// Case values decoded from Viva's Greek samples page (see plans/25-greek-fiscalisation.md).

func TestGRReceiptCase(t *testing.T) {
	tests := []struct {
		name string
		got  ReceiptCase
		want int64
	}{
		{"11.1 retail receipt", GRReceiptCase(GRReceiptPOS, ReceiptFlagCustomNumbered), 5139205309155770369},
		{"11.4 return", GRReceiptCase(GRReceiptPOS, ReceiptFlagRefund|ReceiptFlagCustomNumbered), 5139205309172547585},
		// What to send for receipts signed at the time of sale; Viva's answer, 2026-11-16.
		{"POS receipt, signed online", GRReceiptCase(GRReceiptPOS, 0), 5139205309155246081},
		{"online (Smart Checkout) receipt", GRReceiptCase(GRReceiptECommerce, 0), 5139205309155246084},
		{"return, signed online", GRReceiptCase(GRReceiptPOS, ReceiptFlagRefund), 5139205309172023297},
		{"online sales sample", GRReceiptCase(GRReceiptPaymentTransfer, ReceiptFlagCustomNumbered), 5139205309155770370},
		{"queue start", GRReceiptCase(GRReceiptQueueStart, 0), 5139205309155262465},
		{"zero receipt", GRReceiptCase(GRReceiptZero, 0), 5139205309155254272},
	}
	for _, tt := range tests {
		if int64(tt.got) != tt.want {
			t.Errorf("%s = %d (%#x), want %d", tt.name, tt.got, tt.got, tt.want)
		}
	}
}

func TestGRChargeItemCase(t *testing.T) {
	tests := []struct {
		name string
		got  ChargeItemCase
		want int64
	}{
		// The ChargeItemCase page's own "goods 24%" example gives the decimal of …0003;
		// every 11.1 sample uses …0013, which is what the pattern encodes.
		{"goods, 24%", GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, 0), 5139205309155246099},
		{"service, 24%", GRChargeItemCase(GRVATNatureUsual, GRServiceOther, GRVATNormal, 0), 5139205309155246115},
		{"refunded goods, 24%", GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, ChargeItemFlagRefund), 5139205309155377171},
		{"exemption nature in NN", GRChargeItemCase(GRVATNature(0x11), GRServiceGoods, GRVATNotTaxable, 0), grCaseBase + 0x1118},
	}
	for _, tt := range tests {
		if int64(tt.got) != tt.want {
			t.Errorf("%s = %d (%#x), want %d", tt.name, tt.got, tt.got, tt.want)
		}
	}
}

func TestGRPayItemCase(t *testing.T) {
	tests := []struct {
		name string
		got  PayItemCase
		want int64
	}{
		{"cash", GRPayItemCase(GRPaymentCash, 0), 5139205309155246081},
		{"EFTPOS", GRPayItemCase(GRPaymentEFTPOS, 0), 5139205309155246082},
		{"on credit", GRPayItemCase(GRPaymentOnCredit, 0), 5139205309155246089},
		// PP=04 is not in the Greek table, but Viva confirmed it for card refunds.
		{"card refund", GRPayItemCase(GRPaymentCardRefund, PayItemFlagRefund), 5139205309155377156},
	}
	for _, tt := range tests {
		if int64(tt.got) != tt.want {
			t.Errorf("%s = %d (%#x), want %d", tt.name, tt.got, tt.got, tt.want)
		}
	}
}

func TestFormatGRAmount(t *testing.T) {
	tests := map[int64]string{
		0:      "0.0",
		1:      "0.01",
		5:      "0.05",
		10:     "0.1",
		105:    "1.05",
		124:    "1.24",
		225:    "2.25",
		1240:   "12.4",
		1780:   "17.8",
		2300:   "23.0",
		-124:   "-1.24",
		-225:   "-2.25",
		-12400: "-124.0",
	}
	for in, want := range tests {
		if got := formatGRAmount(in); got != want {
			t.Errorf("formatGRAmount(%d) = %q, want %q", in, got, want)
		}
	}
}

func mustMoment(t *testing.T, s string) time.Time {
	t.Helper()
	m, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// HashPayloads copied verbatim from Viva's Greek samples, with the inputs they were built from.
func TestGRHashPayload(t *testing.T) {
	tests := []struct {
		vatID, series string
		aa            int64
		ref, moment   string
		amount        int64
		want          string
	}{
		{"123456789", "AB", 1, "testreceiptreference1", "2025-08-27T16:11:06Z", 124,
			"123456789-AB-1-testreceiptreference1-2025-08-27T16:11:06Z-1.24"},
		{"099565360", "AB", 123, "testreceiptreference1", "2026-07-10T11:28:00Z", 1780,
			"099565360-AB-123-testreceiptreference1-2026-07-10T11:28:00Z-17.8"},
		{"098000979", "A", 397717, "receiptreference1", "2025-07-17T08:11:06Z", 2300,
			"098000979-A-397717-receiptreference1-2025-07-17T08:11:06Z-23.0"},
		{"099565360", "A12341B", 2133, "posactionrefund2", "2025-10-16T00:01:06Z", -124,
			"099565360-A12341B-2133-posactionrefund2-2025-10-16T00:01:06Z--1.24"},
		{"099565360", "ABCX", 12233243215123, "refundinvoice2125132", "2026-02-05T15:22:07Z", -12400,
			"099565360-ABCX-12233243215123-refundinvoice2125132-2026-02-05T15:22:07Z--124.0"},
		{"099565360", "SER", 15, "REF", "2025-11-04T12:40:16Z", -225,
			"099565360-SER-15-REF-2025-11-04T12:40:16Z--2.25"},
	}
	for _, tt := range tests {
		got := GRHashPayload(tt.vatID, tt.series, tt.aa, tt.ref, mustMoment(t, tt.moment), tt.amount)
		if got != tt.want {
			t.Errorf("GRHashPayload() = %q, want %q", got, tt.want)
		}
	}
}

func TestFormatFiscalMomentIsUTCToTheSecond(t *testing.T) {
	athens := time.FixedZone("EEST", 3*60*60)
	m := time.Date(2025, 8, 27, 19, 11, 6, 999_000_000, athens)
	if got, want := FormatFiscalMoment(m), "2025-08-27T16:11:06Z"; got != want {
		t.Errorf("FormatFiscalMoment() = %q, want %q", got, want)
	}
}

func TestGRReceiptQRURL(t *testing.T) {
	// The first example on Viva's core page; the second one there matches no encoding of
	// its input and is not used.
	payload := "099565360-ab-12-ab2-2026-09-15T16:11:06Z-1.24"
	if got, want := GRReceiptQRURL(Config{Demo: true}, payload), "https://r-sb.ft.ms/dz_87y5EqFh0TVlXDPUUFvLHWsoK174HXqFp6ewdwCQ"; got != want {
		t.Errorf("demo = %q, want %q", got, want)
	}
	if got, want := GRReceiptQRURL(Config{}, payload), "https://r.ft.ms/dz_87y5EqFh0TVlXDPUUFvLHWsoK174HXqFp6ewdwCQ"; got != want {
		t.Errorf("production = %q, want %q", got, want)
	}
}

func TestNewFiscalSessionIDIsUUIDv4(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := NewFiscalSessionID()
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(id) {
			t.Fatalf("NewFiscalSessionID() = %q, not a UUID v4", id)
		}
		if seen[id] {
			t.Fatalf("NewFiscalSessionID() repeated %q", id)
		}
		seen[id] = true
	}
}

func assertJSONEqual(t *testing.T, got []byte, fixture string) {
	t.Helper()
	want, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var g, w interface{}
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}

// The 11.1 ΑΛΠ sample, built from the SDK's types and builders, must serialise to the sample.
// The fixture sends every currencyCode as the string "978", the type Viva confirmed; the
// original sample mixes in a
// number in the charge item.
func TestFiscalisationDataMatchesGR111Sample(t *testing.T) {
	moment := mustMoment(t, "2025-08-27T16:11:06Z")
	m := FormatFiscalMoment(moment)
	d := FiscalisationData{
		CBChargeItems: []ChargeItem{{
			Amount: 124, Description: "product x", Position: 1, Moment: m,
			Quantity: 100, VATRate: 2400, VATAmount: 24,
			FTChargeItemCase: GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, 0),
			CurrencyCode:     FiscalCurrencyEUR,
		}},
		CBPayItems: []PayItem{{
			Amount: 124, Description: "cash-payment", Position: 1, Moment: m,
			FTPayItemCase: GRPayItemCase(GRPaymentCash, 0),
			CurrencyCode:  FiscalCurrencyEUR,
		}},
		CBReceiptMoment:    m,
		CBReceiptReference: "testreceiptreference1",
		CurrencyCode:       FiscalCurrencyEUR,
		FTReceiptCase:      GRReceiptCase(GRReceiptPOS, ReceiptFlagCustomNumbered),
		FTReceiptCaseData: &ReceiptCaseData{GR: &GRReceiptCaseData{
			MerchantVATID: "123456789", Series: "AB", AA: 1, HashAlg: GRHashAlg,
			HashPayload: GRHashPayload("123456789", "AB", 1, "testreceiptreference1", moment, 124),
		}},
	}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, got, "testdata/fiscal_gr_receipt_11_1.json")
}

func TestFiscalisationDataSendsEmptyItemsAsArrays(t *testing.T) {
	d := FiscalisationData{
		CBReceiptReference: "Start-receipt-time-13Aug-2025-11:30",
		CBReceiptMoment:    FormatFiscalMoment(mustMoment(t, "2025-08-13T11:30:04Z")),
		CurrencyCode:       FiscalCurrencyEUR,
		FTReceiptCase:      GRReceiptCase(GRReceiptQueueStart, 0),
	}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, got, "testdata/fiscal_gr_queue_start.json")
}

func decodeFixture(t *testing.T, name string) *FiscalResult {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	r := &FiscalResult{}
	if err := json.Unmarshal(b, r); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return r
}

func TestFiscalResultSignResponse(t *testing.T) {
	r := decodeFixture(t, "testdata/fiscal_sign_response_zero.json")
	if !r.State.Ready() {
		t.Errorf("State %s: Ready() = false, want true", r.State)
	}
	if r.Transmission() != TransmissionOK {
		t.Errorf("Transmission() = %v, want OK", r.Transmission())
	}
	if r.Mark() != "" {
		t.Errorf("Mark() = %q, want none for a zero receipt", r.Mark())
	}
	if r.ReceiptIdentification != "ft814#" || len(r.Signatures) != 1 || r.Signatures[0].Format != SignatureFormatText {
		t.Errorf("unexpected decode: %+v", r)
	}
}

// An EFT POS session's fiscalisationDetails carries ftState and ftSignatureType as strings.
func TestFiscalResultSessionDetailsWithStringIntegers(t *testing.T) {
	r := decodeFixture(t, "testdata/fiscal_session_details.json")
	if int64(r.State) != 5139205309155246080 {
		t.Errorf("State = %d, want 5139205309155246080", r.State)
	}
	if got := int64(r.Signatures[1].Type); got != 5139205309155246100 {
		t.Errorf("Signatures[1].Type = %d, want 5139205309155246100", got)
	}
	if got := r.Signatures[4].Type; got != 0 {
		t.Errorf("missing ftSignatureType = %d, want 0", got)
	}
	if r.Mark() != "400001948544829" {
		t.Errorf("Mark() = %q, want 400001948544829", r.Mark())
	}
	if r.Signatures[3].Format != SignatureFormatQRCode {
		t.Errorf("Signatures[3].Format = %d, want QR code", r.Signatures[3].Format)
	}
}

func TestFiscalResultMissingIndicatorIsFailure2(t *testing.T) {
	r := decodeFixture(t, "testdata/fiscal_sign_response_no_indicator.json")
	if got := r.Transmission(); got != TransmissionFailure2 || got.String() != "TRANSMISSION_FAILURE_2" {
		t.Errorf("Transmission() = %v (%q), want TransmissionFailure2", int(got), got)
	}
	one, nine := 1, 9
	if got := (&FiscalResult{RawTransmissionIndicator: &one}).Transmission(); got != TransmissionFailure1 {
		t.Errorf("indicator 1 = %v, want TransmissionFailure1", int(got))
	}
	if got := (&FiscalResult{RawTransmissionIndicator: &nine}).Transmission(); got != TransmissionFailure2 {
		t.Errorf("undocumented indicator = %v, want TransmissionFailure2", int(got))
	}
}

func TestFiscalState(t *testing.T) {
	base := FiscalState(grCaseBase)
	tests := []struct {
		name                                          string
		s                                             FiscalState
		ready, scu, pending, closing, isError, failed bool
	}{
		{"ready", base, true, false, false, false, false, false},
		{"scu out of service", base | 0x2, false, true, false, false, false, false},
		{"pending and closing due", base | 0x140, false, false, true, true, false, false},
		{"error", base | 0xEEEEEEEE, false, false, false, false, true, false},
		{"fail", base | 0xFFFFFFFF, false, false, false, false, false, true},
	}
	for _, tt := range tests {
		s := tt.s
		got := []bool{s.Ready(), s.SCUOutOfService(), s.MessagePending(), s.DailyClosingDue(), s.Error(), s.Fail()}
		want := []bool{tt.ready, tt.scu, tt.pending, tt.closing, tt.isError, tt.failed}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s (%s): got %v, want %v", tt.name, s, got, want)
		}
	}
}

func TestFiscalIntRejectsGarbage(t *testing.T) {
	var s FiscalState
	if err := json.Unmarshal([]byte(`"not-a-number"`), &s); err == nil {
		t.Error("want error for non-numeric ftState")
	}
}
