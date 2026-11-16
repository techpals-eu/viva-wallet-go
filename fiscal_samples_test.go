package vivawallet

import (
	"encoding/json"
	"testing"
	"time"
)

// Viva's Greek business-case samples, as fixtures. Each is copied from
// https://developer.viva.com/viva-fiscalisation/greece/samples/ (updated 27/10/2026) with one
// normalisation: every currencyCode is the string "978", which Viva confirmed is the official
// type, where the samples mix in the number 978.
var grSampleFixtures = []string{
	"testdata/fiscal_gr_receipt_11_1.json",
	"testdata/fiscal_gr_receipt_11_2.json",
	"testdata/fiscal_gr_return_11_4_unlinked.json",
	"testdata/fiscal_gr_return_11_4_linked.json",
	"testdata/fiscal_gr_return_11_4_marks.json",
	"testdata/fiscal_gr_queue_start.json",
	"testdata/fiscal_gr_zero.json",
}

func loadGRSample(t *testing.T, name string) FiscalisationData {
	t.Helper()
	var d FiscalisationData
	if err := json.Unmarshal(readFixture(t, name), &d); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return d
}

// Every sample must decode into the SDK types and encode back unchanged: a field the types
// cannot hold would be lost here.
func TestGRSamplesRoundTrip(t *testing.T) {
	for _, name := range grSampleFixtures {
		got, err := json.Marshal(loadGRSample(t, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		assertJSONEqual(t, got, name)
	}
}

// Every sample's HashPayload must be what GRHashPayload builds from the same receipt, with
// the total taken from the pay items.
func TestGRSamplesHashPayload(t *testing.T) {
	for _, name := range grSampleFixtures {
		d := loadGRSample(t, name)
		if d.FTReceiptCaseData == nil || d.FTReceiptCaseData.GR == nil {
			continue // Queue-Start and Zero receipts carry no document data
		}
		gr := d.FTReceiptCaseData.GR
		var total int64
		for _, p := range d.CBPayItems {
			total += p.Amount
		}
		moment, err := time.Parse(fiscalMomentLayout, d.CBReceiptMoment)
		if err != nil {
			t.Fatalf("%s: cbReceiptMoment: %v", name, err)
		}
		got := GRHashPayload(gr.MerchantVATID, gr.Series, gr.AA, d.CBReceiptReference, moment, total)
		if got != gr.HashPayload {
			t.Errorf("%s: GRHashPayload() = %q, sample has %q", name, got, gr.HashPayload)
		}
	}
}

// grReceipt assembles a Greek receipt from the SDK's builders the way a caller would.
func grReceipt(t *testing.T, vatID, series string, aa int64, ref, moment string, rc ReceiptCase,
	lines []ChargeItem, pays []PayItem) FiscalisationData {
	t.Helper()
	m := mustMoment(t, moment)
	var total int64
	for i := range lines {
		lines[i].Moment = FormatFiscalMoment(m)
		lines[i].CurrencyCode = FiscalCurrencyEUR
	}
	for i := range pays {
		pays[i].Moment = FormatFiscalMoment(m)
		pays[i].CurrencyCode = FiscalCurrencyEUR
		total += pays[i].Amount
	}
	return FiscalisationData{
		CBChargeItems:      lines,
		CBPayItems:         pays,
		CBReceiptMoment:    FormatFiscalMoment(m),
		CBReceiptReference: ref,
		CurrencyCode:       FiscalCurrencyEUR,
		FTReceiptCase:      rc,
		FTReceiptCaseData: &ReceiptCaseData{GR: &GRReceiptCaseData{
			MerchantVATID: vatID, Series: series, AA: aa, HashAlg: GRHashAlg,
			HashPayload: GRHashPayload(vatID, series, aa, ref, m, total),
		}},
	}
}

// The samples all set ReceiptFlagCustomNumbered (0008). Viva confirmed that receipts signed
// at the time of sale use no flag, so these reproduce the samples, not what to send.
var (
	grRetailReceipt   = GRReceiptCase(GRReceiptPOS, ReceiptFlagCustomNumbered)
	grRetailReturn    = GRReceiptCase(GRReceiptPOS, ReceiptFlagRefund|ReceiptFlagCustomNumbered)
	grService24       = GRChargeItemCase(GRVATNatureUsual, GRServiceOther, GRVATNormal, 0)
	grGoods24Return   = GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, ChargeItemFlagRefund)
	grCash            = GRPayItemCase(GRPaymentCash, 0)
	grSampleRefundPay = GRPayItemCase(GRPaymentCardRefund, PayItemFlagRefund)
)

// Every sample must be reproducible from the SDK's builders: the case values, the hash, the
// one-or-many reference types and the empty-item handling.
func TestGRSamplesFromBuilders(t *testing.T) {
	tests := []struct {
		fixture string
		build   func() FiscalisationData
	}{
		{"testdata/fiscal_gr_receipt_11_2.json", func() FiscalisationData {
			return grReceipt(t, "12345678", "AB", 1, "testreceiptreference1", "2025-08-27T16:11:06Z", grRetailReceipt,
				[]ChargeItem{{Amount: 124, Description: "product x", Position: 1, Quantity: 100, VATRate: 2400, VATAmount: 24, FTChargeItemCase: grService24}},
				[]PayItem{{Amount: 124, Description: "cash-payment", Position: 1, FTPayItemCase: grCash}})
		}},
		{"testdata/fiscal_gr_return_11_4_unlinked.json", func() FiscalisationData {
			return grReceipt(t, "099565360", "A12341B", 2133, "posactionrefund2", "2025-10-16T00:01:06Z", grRetailReturn,
				[]ChargeItem{{Amount: -124, Description: "test product refund", Position: 1, Quantity: -100, VATRate: 2400, VATAmount: -24, FTChargeItemCase: grGoods24Return}},
				[]PayItem{{Amount: -124, Description: "test-card-refund", Position: 1, Quantity: -100, FTPayItemCase: grSampleRefundPay}})
		}},
		{"testdata/fiscal_gr_return_11_4_linked.json", func() FiscalisationData {
			d := grReceipt(t, "099565360", "ABCX", 12233243215123, "refundinvoice2125132", "2026-02-05T15:22:07Z", grRetailReturn,
				[]ChargeItem{{Amount: -12400, Description: "Couch", Position: 1, Quantity: -100, VATRate: 2400, VATAmount: -2400, FTChargeItemCase: grGoods24Return}},
				[]PayItem{{Amount: -12400, Description: "Cash", Position: 1, FTPayItemCase: grSampleRefundPay}})
			d.CBPreviousReceiptReference = ReceiptReferences{"Delivery_Note150123"}
			d.FTReceiptCaseData.GR.PreviousReceiptReference = &GRPreviousReceiptReference{InvoiceMarks: InvoiceMarks{400001962634793}}
			return d
		}},
		{"testdata/fiscal_gr_return_11_4_marks.json", func() FiscalisationData {
			d := grReceipt(t, "099565360", "A12341B", 21333, "posactionrefund2", "2026-05-19T00:01:06Z", grRetailReturn,
				[]ChargeItem{{Amount: -124, Description: "test product refund", Position: 1, Quantity: -100, VATRate: 2400, VATAmount: -24, FTChargeItemCase: grGoods24Return}},
				[]PayItem{{Amount: -124, Description: "test-card-refund", Position: 1, Quantity: -100, FTPayItemCase: grSampleRefundPay}})
			d.FTReceiptCaseData.GR.PreviousReceiptReference = &GRPreviousReceiptReference{InvoiceMarks: InvoiceMarks{400001962641135, 400001962641150}}
			return d
		}},
		{"testdata/fiscal_gr_zero.json", func() FiscalisationData {
			d := grOperationReceipt(GRReceiptZero, "zero", "00000000", mustMoment(t, "2026-10-26T11:53:38Z"))
			d.CBReceiptReference = "health-check-2026-10-26T11:53:38Z" // the sample's own reference
			return d
		}},
	}
	for _, tt := range tests {
		got, err := json.Marshal(tt.build())
		if err != nil {
			t.Fatalf("%s: %v", tt.fixture, err)
		}
		assertJSONEqual(t, got, tt.fixture)
	}
}

func TestReceiptReferencesJSON(t *testing.T) {
	encode := map[string]ReceiptReferences{
		`"a"`:       {"a"},
		`["a","b"]`: {"a", "b"},
	}
	for want, refs := range encode {
		if got, _ := json.Marshal(refs); string(got) != want {
			t.Errorf("Marshal(%v) = %s, want %s", refs, got, want)
		}
	}
	var r ReceiptReferences
	for in, want := range map[string]int{`"a"`: 1, `["a","b"]`: 2} {
		if err := json.Unmarshal([]byte(in), &r); err != nil || len(r) != want {
			t.Errorf("Unmarshal(%s) = %v, %v", in, r, err)
		}
	}
	if err := json.Unmarshal([]byte(`{"a":1}`), &r); err == nil {
		t.Error("Unmarshal(object): want error")
	}
	// An unset reference is left out of the request entirely.
	b, _ := json.Marshal(FiscalisationData{})
	if string(b) != `{"cbChargeItems":[],"cbPayItems":[],"cbReceiptMoment":"","cbReceiptReference":"","currencyCode":"","ftReceiptCase":0}` {
		t.Errorf("empty receipt = %s", b)
	}
}

func TestInvoiceMarksJSON(t *testing.T) {
	encode := map[string]InvoiceMarks{
		`400001962634793`:                   {400001962634793},
		`[400001962641135,400001962641150]`: {400001962641135, 400001962641150},
	}
	for want, marks := range encode {
		if got, _ := json.Marshal(marks); string(got) != want {
			t.Errorf("Marshal(%v) = %s, want %s", marks, got, want)
		}
	}
	var m InvoiceMarks
	if err := json.Unmarshal([]byte(`400001962634793`), &m); err != nil || len(m) != 1 || m[0] != 400001962634793 {
		t.Errorf("Unmarshal(number) = %v, %v", m, err)
	}
	if err := json.Unmarshal([]byte(`[1,2]`), &m); err != nil || len(m) != 2 {
		t.Errorf("Unmarshal(array) = %v, %v", m, err)
	}
	if err := json.Unmarshal([]byte(`"400001962634793"`), &m); err == nil {
		t.Error("Unmarshal(string): want error, MARKs are numbers")
	}
}
