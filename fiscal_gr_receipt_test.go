package vivawallet

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// expectedFromSample is a Viva sample adjusted to what the builder sends: the receipt case
// without flag 0008 (Viva: only for receipts signed after the fact), and cbReceiptAmount,
// which the core docs mark as required but the samples omit.
func expectedFromSample(t *testing.T, fixture string, rc ReceiptCase, total int64) []byte {
	t.Helper()
	d := loadGRSample(t, fixture)
	d.FTReceiptCase = rc
	d.CBReceiptAmount = total
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertJSONEqualBytes(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w interface{}
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatal(err)
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	if string(gb) != string(wb) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", gb, wb)
	}
}

func marshal(t *testing.T, d FiscalisationData, err error) []byte {
	t.Helper()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGRReceiptSaleMatchesSample(t *testing.T) {
	r := GRReceipt{
		MerchantVATID: "123456789", Series: "AB", AA: 1,
		Reference: "testreceiptreference1", Moment: mustMoment(t, "2025-08-27T16:11:06Z"),
		Lines: []ChargeItem{{Amount: 124, Description: "product x", Quantity: 100, VATRate: 2400, VATAmount: 24,
			FTChargeItemCase: GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, 0)}},
		Payments: []PayItem{{Amount: 124, Description: "cash-payment", FTPayItemCase: GRPayItemCase(GRPaymentCash, 0)}},
	}
	d, err := r.Sale()
	assertJSONEqualBytes(t, marshal(t, d, err),
		expectedFromSample(t, "testdata/fiscal_gr_receipt_11_1.json", 5139205309155246081, 124))
}

func TestGRReceiptOnlineSale(t *testing.T) {
	r := GRReceipt{
		MerchantVATID: "123456789", Series: "W", AA: 7,
		Reference: "web-7", Moment: mustMoment(t, "2026-11-16T10:00:00Z"),
		Lines: []ChargeItem{{Amount: 1240, Description: "Room night", Quantity: 100, VATRate: 2400, VATAmount: 240,
			FTChargeItemCase: GRChargeItemCase(GRVATNatureUsual, GRServiceOther, GRVATNormal, 0)}},
		Payments: []PayItem{{Amount: 1240, Description: "Card", TransactionID: "27efa6c4-1e51-440e-96c8-e674a04c9184",
			FTPayItemCase: GRPayItemCase(GRPaymentEFTPOS, 0)}},
	}
	d, err := r.OnlineSale()
	if err != nil {
		t.Fatal(err)
	}
	if int64(d.FTReceiptCase) != 5139205309155246084 {
		t.Errorf("ftReceiptCase = %d, want e-commerce 5139205309155246084", d.FTReceiptCase)
	}
	if d.CBPayItems[0].TransactionID == "" {
		t.Error("TransactionID dropped")
	}
	if got := d.FTReceiptCaseData.GR.HashPayload; got != "123456789-W-7-web-7-2026-11-16T10:00:00Z-12.4" {
		t.Errorf("HashPayload = %q", got)
	}
}

// The unlinked 11.4 sample, built from the lines as sold and a card payment: Return negates
// everything, sets the refund flags and switches the card payment to PP 04.
func TestGRReceiptReturnMatchesSample(t *testing.T) {
	r := GRReceipt{
		MerchantVATID: "099565360", Series: "A12341B", AA: 2133,
		Reference: "posactionrefund2", Moment: mustMoment(t, "2025-10-16T00:01:06Z"),
		Lines: []ChargeItem{{Amount: 124, Description: "test product refund", Quantity: 100, VATRate: 2400, VATAmount: 24,
			FTChargeItemCase: GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, 0)}},
		Payments: []PayItem{{Amount: 124, Description: "test-card-refund", Quantity: 100,
			FTPayItemCase: GRPayItemCase(GRPaymentEFTPOS, 0)}},
	}
	d, err := r.Return()
	assertJSONEqualBytes(t, marshal(t, d, err),
		expectedFromSample(t, "testdata/fiscal_gr_return_11_4_unlinked.json", 5139205309172023297, -124))
}

func TestGRReceiptReturnLinkedByMarks(t *testing.T) {
	r := GRReceipt{
		MerchantVATID: "099565360", Series: "A12341B", AA: 21333,
		Reference: "posactionrefund2", Moment: mustMoment(t, "2026-05-19T00:01:06Z"),
		Lines: []ChargeItem{{Amount: 124, Description: "test product refund", Quantity: 100, VATRate: 2400, VATAmount: 24,
			FTChargeItemCase: GRChargeItemCase(GRVATNatureUsual, GRServiceGoods, GRVATNormal, 0)}},
		Payments: []PayItem{{Amount: 124, Description: "test-card-refund", Quantity: 100,
			FTPayItemCase: GRPayItemCase(GRPaymentEFTPOS, 0)}},
		PreviousMarks: InvoiceMarks{400001962641135, 400001962641150},
	}
	d, err := r.Return()
	assertJSONEqualBytes(t, marshal(t, d, err),
		expectedFromSample(t, "testdata/fiscal_gr_return_11_4_marks.json", 5139205309172023297, -124))
}

func TestGRReceiptReturnLinkedByReference(t *testing.T) {
	r := GRReceipt{
		MerchantVATID: "099565360", Series: "R", AA: 1, Reference: "ret-1",
		Moment:             mustMoment(t, "2026-02-05T15:22:07Z"),
		Lines:              []ChargeItem{{Amount: 12400, Description: "Couch", Quantity: 100, VATRate: 2400, VATAmount: 2400, FTChargeItemCase: GRChargeItemCase(0, GRServiceGoods, GRVATNormal, 0)}},
		Payments:           []PayItem{{Amount: 12400, Description: "Cash", FTPayItemCase: GRPayItemCase(GRPaymentCash, 0)}},
		PreviousReferences: ReceiptReferences{"rcpt-000123"},
	}
	d, err := r.Return()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(d)
	for _, want := range []string{
		`"cbPreviousReceiptReference":"rcpt-000123"`,
		`"ftPayItemCase":5139205309155377153`, // cash stays cash (PP 01), with the refund flag
		`"HashPayload":"099565360-R-1-ret-1-2026-02-05T15:22:07Z--124.0"`,
		`"cbReceiptAmount":-12400`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s in\n%s", want, b)
		}
	}
	if strings.Contains(string(b), "PreviousReceiptReference\":{") {
		t.Error("GR PreviousReceiptReference sent with no marks")
	}
}

// The builder must not modify the caller's slices: building a return from a sale's lines
// must leave the sale intact.
func TestGRReceiptDoesNotMutateInput(t *testing.T) {
	lines := []ChargeItem{{Amount: 124, Description: "x", Quantity: 100, FTChargeItemCase: GRChargeItemCase(0, GRServiceGoods, GRVATNormal, 0)}}
	pays := []PayItem{{Amount: 124, Description: "Card", FTPayItemCase: GRPayItemCase(GRPaymentEFTPOS, 0)}}
	r := GRReceipt{MerchantVATID: "1", Series: "A", AA: 1, Reference: "r", Moment: mustMoment(t, "2026-01-01T00:00:00Z"), Lines: lines, Payments: pays}
	if _, err := r.Return(); err != nil {
		t.Fatal(err)
	}
	if lines[0].Amount != 124 || lines[0].Position != 0 || pays[0].FTPayItemCase != GRPayItemCase(GRPaymentEFTPOS, 0) {
		t.Errorf("input mutated: %+v %+v", lines[0], pays[0])
	}
}

func TestGRReceiptValidation(t *testing.T) {
	good := func() GRReceipt {
		return GRReceipt{
			MerchantVATID: "1", Series: "A", AA: 1, Reference: "r", Moment: mustMoment(t, "2026-01-01T00:00:00Z"),
			Lines:    []ChargeItem{{Amount: 100, Description: "x", Quantity: 100, FTChargeItemCase: GRChargeItemCase(0, GRServiceGoods, GRVATNormal, 0)}},
			Payments: []PayItem{{Amount: 100, Description: "Cash", FTPayItemCase: GRPayItemCase(GRPaymentCash, 0)}},
		}
	}
	if _, err := good().Sale(); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*GRReceipt)
		want   string
	}{
		{"no VAT id", func(r *GRReceipt) { r.MerchantVATID = "" }, "MerchantVATID"},
		{"no AA", func(r *GRReceipt) { r.AA = 0 }, "AA"},
		{"no moment", func(r *GRReceipt) { r.Moment = time.Time{} }, "Moment"},
		{"no lines", func(r *GRReceipt) { r.Lines = nil }, "no Lines"},
		{"totals differ", func(r *GRReceipt) { r.Payments[0].Amount = 99 }, "lines total 100 but payments total 99"},
		{"quantity not scaled", func(r *GRReceipt) { r.Lines[0].Quantity = 0 }, "Quantity"},
		{"negative line", func(r *GRReceipt) { r.Lines[0].Amount = -100; r.Payments[0].Amount = -100 }, "positive"},
		{"raw case value", func(r *GRReceipt) { r.Lines[0].FTChargeItemCase = 0x13 }, "GRChargeItemCase"},
		{"link on a sale", func(r *GRReceipt) { r.PreviousMarks = InvoiceMarks{1} }, "only valid on a Return"},
	}
	for _, tt := range tests {
		r := good()
		tt.mutate(&r)
		_, err := r.Sale()
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want it to mention %q", tt.name, err, tt.want)
		}
	}
}
