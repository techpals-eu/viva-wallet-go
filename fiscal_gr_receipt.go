package vivawallet

import (
	"errors"
	"fmt"
	"time"
)

// GRReceipt is a Greek retail receipt (ΑΛΠ 11.1 / ΑΠΥ 11.2) or return (11.4) as a caller
// describes it. Sale, OnlineSale and Return turn it into FiscalisationData: they set the
// receipt case, the per-line moments and currency, the total and the HashPayload, and
// validate the result before anything is sent.
//
// Lines and Payments are given as sold, with positive amounts, for a return as well. Return
// negates them and sets the refund flags itself.
type GRReceipt struct {
	// MerchantVATID is the merchant's ΑΦΜ.
	MerchantVATID string
	// Series and AA number the receipt in the merchant's books. AA must increase per
	// Series and never be reused; the SDK cannot track it.
	Series string
	AA     int64
	// Reference is the receipt's unique id in your system (cbReceiptReference).
	Reference string
	// Moment is when the receipt is issued. It is sent in UTC, to the second.
	Moment time.Time

	// Lines and Payments must each have at least one entry, and must add up to the same
	// total. Position, Moment and CurrencyCode may be left empty: they default to the line's
	// index (from 1), the receipt's Moment and FiscalCurrencyEUR.
	Lines    []ChargeItem
	Payments []PayItem

	User     *FiscalUser
	Customer *FiscalCustomer
	Area     string

	// PreviousReferences and PreviousMarks link a return to the receipt(s) it reverses, by
	// cbReceiptReference or by ΑΑΔΕ MARK. Both are optional, and only valid on a Return.
	PreviousReferences ReceiptReferences
	PreviousMarks      InvoiceMarks
}

// Sale builds a receipt for a sale at the till (11.1 for goods, 11.2 for services; which one
// follows from the lines' service type), signed at the time of sale.
func (r GRReceipt) Sale() (FiscalisationData, error) {
	return r.build(GRReceiptCase(GRReceiptPOS, 0), false)
}

// OnlineSale builds the receipt for an online (Smart Checkout) sale. Set each payment's
// TransactionID to the Viva transaction it was paid with.
func (r GRReceipt) OnlineSale() (FiscalisationData, error) {
	return r.build(GRReceiptCase(GRReceiptECommerce, 0), false)
}

// Return builds a return receipt (11.4). It negates the given lines and payments, marks them
// as refunds, and turns card payments (GRPaymentEFTPOS) into GRPaymentCardRefund, which is
// what Viva expects for a card refund.
func (r GRReceipt) Return() (FiscalisationData, error) {
	return r.build(GRReceiptCase(GRReceiptPOS, ReceiptFlagRefund), true)
}

func (r GRReceipt) build(rc ReceiptCase, isReturn bool) (FiscalisationData, error) {
	if err := r.validate(isReturn); err != nil {
		return FiscalisationData{}, err
	}
	at := FormatFiscalMoment(r.Moment)

	lines := make([]ChargeItem, len(r.Lines))
	var linesTotal int64
	for i, l := range r.Lines {
		if l.Position == 0 {
			l.Position = i + 1
		}
		if l.Moment == "" {
			l.Moment = at
		}
		if l.CurrencyCode == "" {
			l.CurrencyCode = FiscalCurrencyEUR
		}
		if isReturn {
			l.Amount, l.Quantity, l.VATAmount = -l.Amount, -l.Quantity, -l.VATAmount
			l.FTChargeItemCase |= ChargeItemCase(uint64(ChargeItemFlagRefund) << 16)
		}
		linesTotal += l.Amount
		lines[i] = l
	}

	pays := make([]PayItem, len(r.Payments))
	var paysTotal int64
	for i, p := range r.Payments {
		if p.Position == 0 {
			p.Position = i + 1
		}
		if p.Moment == "" {
			p.Moment = at
		}
		if p.CurrencyCode == "" {
			p.CurrencyCode = FiscalCurrencyEUR
		}
		if isReturn {
			p.Amount, p.Quantity = -p.Amount, -p.Quantity
			if grPaymentType(p.FTPayItemCase) == GRPaymentEFTPOS {
				p.FTPayItemCase = p.FTPayItemCase&^0xFF | PayItemCase(GRPaymentCardRefund)
			}
			p.FTPayItemCase |= PayItemCase(uint64(PayItemFlagRefund) << 16)
		}
		paysTotal += p.Amount
		pays[i] = p
	}

	if linesTotal != paysTotal {
		return FiscalisationData{}, fmt.Errorf("vivawallet: receipt %s: lines total %d but payments total %d",
			r.Reference, linesTotal, paysTotal)
	}

	gr := &GRReceiptCaseData{
		MerchantVATID: r.MerchantVATID,
		Series:        r.Series,
		AA:            r.AA,
		HashAlg:       GRHashAlg,
		HashPayload:   GRHashPayload(r.MerchantVATID, r.Series, r.AA, r.Reference, r.Moment, paysTotal),
	}
	if len(r.PreviousMarks) > 0 {
		gr.PreviousReceiptReference = &GRPreviousReceiptReference{InvoiceMarks: r.PreviousMarks}
	}

	return FiscalisationData{
		CBChargeItems:              lines,
		CBPayItems:                 pays,
		CBUser:                     r.User,
		CBCustomer:                 r.Customer,
		CBArea:                     r.Area,
		CBReceiptMoment:            at,
		CBReceiptReference:         r.Reference,
		CBPreviousReceiptReference: r.PreviousReferences,
		CBReceiptAmount:            paysTotal,
		CurrencyCode:               FiscalCurrencyEUR,
		FTReceiptCase:              rc,
		FTReceiptCaseData:          &ReceiptCaseData{GR: gr},
	}, nil
}

func (r GRReceipt) validate(isReturn bool) error {
	var errs []string
	if r.MerchantVATID == "" {
		errs = append(errs, "no MerchantVATID")
	}
	if r.Series == "" {
		errs = append(errs, "no Series")
	}
	if r.AA <= 0 {
		errs = append(errs, "AA must be positive")
	}
	if r.Reference == "" {
		errs = append(errs, "no Reference")
	}
	if r.Moment.IsZero() {
		errs = append(errs, "no Moment")
	}
	if len(r.Lines) == 0 {
		errs = append(errs, "no Lines")
	}
	if len(r.Payments) == 0 {
		errs = append(errs, "no Payments")
	}
	if !isReturn && (len(r.PreviousReferences) > 0 || len(r.PreviousMarks) > 0) {
		errs = append(errs, "PreviousReferences and PreviousMarks are only valid on a Return")
	}
	for i, l := range r.Lines {
		switch {
		case l.Description == "":
			errs = append(errs, fmt.Sprintf("line %d: no Description", i+1))
		case l.Quantity <= 0:
			errs = append(errs, fmt.Sprintf("line %d: Quantity must be positive (×100: 1 is 100)", i+1))
		case l.Amount < 0 || l.VATAmount < 0:
			errs = append(errs, fmt.Sprintf("line %d: give amounts as sold, positive; Return negates them", i+1))
		case !isGRCase(int64(l.FTChargeItemCase)):
			errs = append(errs, fmt.Sprintf("line %d: FTChargeItemCase is not a Greek case; use GRChargeItemCase", i+1))
		}
	}
	for i, p := range r.Payments {
		switch {
		case p.Description == "":
			errs = append(errs, fmt.Sprintf("payment %d: no Description", i+1))
		case p.Amount < 0 || p.Quantity < 0:
			errs = append(errs, fmt.Sprintf("payment %d: give amounts as paid, positive; Return negates them", i+1))
		case !isGRCase(int64(p.FTPayItemCase)):
			errs = append(errs, fmt.Sprintf("payment %d: FTPayItemCase is not a Greek case; use GRPayItemCase", i+1))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New("vivawallet: invalid receipt " + r.Reference + ": " + joinErrs(errs))
}

func joinErrs(errs []string) string {
	s := errs[0]
	for _, e := range errs[1:] {
		s += "; " + e
	}
	return s
}

// isGRCase reports whether v carries Greece's country code and the current tagging version.
func isGRCase(v int64) bool {
	return uint64(v)>>32 == grCaseBase>>32
}

func grPaymentType(c PayItemCase) GRPaymentType {
	return GRPaymentType(uint64(c) & 0xFF)
}
