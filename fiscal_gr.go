package vivawallet

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Greek fiscalisation (ΑΑΔΕ myDATA).
//
// Every ft…Case value is an Int64 built from the hex pattern CCCC_vlll_gggg_xxxx: country
// GR (0x4752), tagging version 2000, 16 bits of flags (gggg, OR-ed together), and 16 bits
// whose meaning depends on the field. Viva's docs show the hex; the API takes the decimal.
// Ref: https://developer.viva.com/viva-fiscalisation/receiptcase/
// Ref: https://developer.viva.com/viva-fiscalisation/chargeitemcase/
// Ref: https://developer.viva.com/viva-fiscalisation/payitemcase/

const grCaseBase = 0x4752_2000_0000_0000

func grCase(flags, low uint16) int64 {
	return int64(grCaseBase | uint64(flags)<<16 | uint64(low))
}

// ReceiptCase is a receipt's ftReceiptCase.
type ReceiptCase int64

// GRReceiptType is the receipt type part (txcc) of a Greek ftReceiptCase.
type GRReceiptType uint16

const (
	GRReceiptUnknown         GRReceiptType = 0x0000
	GRReceiptPOS             GRReceiptType = 0x0001
	GRReceiptPaymentTransfer GRReceiptType = 0x0002
	GRReceiptPOSNoFiscal     GRReceiptType = 0x0003
	// GRReceiptECommerce is the receipt type of an online (Smart Checkout) sale, as Viva
	// confirmed.
	GRReceiptECommerce  GRReceiptType = 0x0004
	GRReceiptZero       GRReceiptType = 0x2000
	GRReceiptQueueStart GRReceiptType = 0x4001
	GRReceiptQueueStop  GRReceiptType = 0x4002
)

// ReceiptFlag is a flag (gggg) of an ftReceiptCase. Combine flags with |.
type ReceiptFlag uint16

const (
	ReceiptFlagLateSigning ReceiptFlag = 0x0001
	ReceiptFlagTraining    ReceiptFlag = 0x0002
	ReceiptFlagVoid        ReceiptFlag = 0x0004
	// ReceiptFlagCustomNumbered marks a receipt that was issued offline with the merchant's
	// own numbering (Series/AA), for example by hand during an outage, and is only now sent
	// for signing. Viva's docs call it "handwritten". Do not set it on receipts signed at the
	// time of sale, even though those carry Series/AA too: Viva confirmed (2026-11-16) that
	// they use no flag, whatever its samples show.
	ReceiptFlagCustomNumbered     ReceiptFlag = 0x0008
	ReceiptFlagSmallBusiness      ReceiptFlag = 0x0010
	ReceiptFlagReceiverIsBusiness ReceiptFlag = 0x0020
	ReceiptFlagReceiverKnown      ReceiptFlag = 0x0040
	ReceiptFlagSaleAbroad         ReceiptFlag = 0x0080
	ReceiptFlagRefund             ReceiptFlag = 0x0100
	ReceiptFlagTransportInfo      ReceiptFlag = 0x0400
	ReceiptFlagGroupByPosition    ReceiptFlag = 0x0800
	// ReceiptFlagReceiptRequest asks for the stored result of an earlier request, e.g. after
	// a timeout, instead of signing again.
	ReceiptFlagReceiptRequest ReceiptFlag = 0x8000
)

// GRReceiptCase builds a Greek ftReceiptCase.
func GRReceiptCase(t GRReceiptType, flags ReceiptFlag) ReceiptCase {
	return ReceiptCase(grCase(uint16(flags), uint16(t)))
}

// ChargeItemCase is a line's ftChargeItemCase.
type ChargeItemCase int64

// GRVATNature is the nature of VAT (NN) of a Greek ftChargeItemCase. Values other than
// GRVATNatureUsual select an exemption article; see Viva's ChargeItemCase page for the
// codes and their myDATA mapping.
type GRVATNature uint8

// GRVATNatureUsual means ordinary VAT applies.
const GRVATNatureUsual GRVATNature = 0x00

// GRServiceType is the type of service (S) of a Greek ftChargeItemCase. It decides the
// myDATA income category: goods → category1_1, services → category1_3, unknown →
// category1_95.
type GRServiceType uint8

const (
	GRServiceUnknown        GRServiceType = 0x0
	GRServiceGoods          GRServiceType = 0x1
	GRServiceOther          GRServiceType = 0x2
	GRServiceTip            GRServiceType = 0x3
	GRServiceVoucher        GRServiceType = 0x4
	GRServiceCatalog        GRServiceType = 0x5
	GRServiceAgency         GRServiceType = 0x6
	GRServiceOwnConsumption GRServiceType = 0x7
	GRServiceGrant          GRServiceType = 0x8
	GRServiceReceivable     GRServiceType = 0x9
	GRServiceCashTransfer   GRServiceType = 0xA
)

// GRVATBucket is the VAT category (V) of a Greek ftChargeItemCase. The rate itself is sent
// separately in ChargeItem.VATRate.
type GRVATBucket uint8

const (
	GRVATUnknown GRVATBucket = 0
	// GRVATDiscounted1 and GRVATDiscounted2 are the reduced rates (13%, 6%).
	GRVATDiscounted1 GRVATBucket = 1
	GRVATDiscounted2 GRVATBucket = 2
	// GRVATNormal is the normal rate, 24%.
	GRVATNormal GRVATBucket = 3
	// GRVATSuperReduced1 and GRVATSuperReduced2 are the super-reduced rate, 4%.
	GRVATSuperReduced1 GRVATBucket = 4
	GRVATSuperReduced2 GRVATBucket = 5
	GRVATParking       GRVATBucket = 6
	GRVATZero          GRVATBucket = 7
	GRVATNotTaxable    GRVATBucket = 8
)

// ChargeItemFlag is a flag (gggg) of an ftChargeItemCase. Combine flags with |.
type ChargeItemFlag uint16

const (
	ChargeItemFlagVoid           ChargeItemFlag = 0x0001
	ChargeItemFlagRefund         ChargeItemFlag = 0x0002
	ChargeItemFlagDiscount       ChargeItemFlag = 0x0004
	ChargeItemFlagDownpayment    ChargeItemFlag = 0x0008
	ChargeItemFlagReturnable     ChargeItemFlag = 0x0010
	ChargeItemFlagTakeAway       ChargeItemFlag = 0x0020
	ChargeItemFlagShowInPayments ChargeItemFlag = 0x8000
)

// GRChargeItemCase builds a Greek ftChargeItemCase. Service and VAT are single hex digits;
// only their low four bits are used.
func GRChargeItemCase(nature GRVATNature, service GRServiceType, vat GRVATBucket, flags ChargeItemFlag) ChargeItemCase {
	low := uint16(nature)<<8 | uint16(service&0xF)<<4 | uint16(vat&0xF)
	return ChargeItemCase(grCase(uint16(flags), low))
}

// PayItemCase is a payment's ftPayItemCase.
type PayItemCase int64

// GRPaymentType is the payment type (PP) of a Greek ftPayItemCase.
type GRPaymentType uint8

const (
	GRPaymentCash GRPaymentType = 0x01
	// GRPaymentEFTPOS covers every electronic method Viva supports: cards, wallets, IRIS
	// through the terminal, Smart Checkout.
	GRPaymentEFTPOS GRPaymentType = 0x02
	// GRPaymentCardRefund is the payment type of a card refund, with PayItemFlagRefund. Viva
	// confirmed it (2026-11-16) although the Greek PP table does not list it; card sales use
	// GRPaymentEFTPOS.
	GRPaymentCardRefund          GRPaymentType = 0x04
	GRPaymentCheque              GRPaymentType = 0x03
	GRPaymentVoucher             GRPaymentType = 0x06
	GRPaymentOnCredit            GRPaymentType = 0x09
	GRPaymentBankTransfer        GRPaymentType = 0x0A
	GRPaymentForeignBankTransfer GRPaymentType = 0x0B
)

// PayItemFlag is a flag (gggg) of an ftPayItemCase. Combine flags with |.
type PayItemFlag uint16

const (
	PayItemFlagVoid              PayItemFlag = 0x0001
	PayItemFlagRefund            PayItemFlag = 0x0002
	PayItemFlagDownpayment       PayItemFlag = 0x0008
	PayItemFlagForeignCurrency   PayItemFlag = 0x0010
	PayItemFlagChange            PayItemFlag = 0x0020
	PayItemFlagTip               PayItemFlag = 0x0040
	PayItemFlagElectronic        PayItemFlag = 0x0080
	PayItemFlagInterfaceVerified PayItemFlag = 0x0100
	PayItemFlagShowInChargeItems PayItemFlag = 0x8000
)

// GRPayItemCase builds a Greek ftPayItemCase.
func GRPayItemCase(pp GRPaymentType, flags PayItemFlag) PayItemCase {
	return PayItemCase(grCase(uint16(flags), uint16(pp)))
}

// GRReceiptCaseData is the Greek part of ftReceiptCaseData. Series and AA are the receipt's
// numbering in the merchant's own books: AA must increase per Series and never be reused.
// Viva does not assign them.
type GRReceiptCaseData struct {
	// MerchantVATID is the merchant's ΑΦΜ.
	MerchantVATID string `json:"MerchantVATID"`
	Series        string `json:"Series"`
	AA            int64  `json:"AA"`
	// HashAlg is always "sha256".
	HashAlg string `json:"HashAlg"`
	// HashPayload is built by GRHashPayload from the same values as the request body.
	HashPayload string `json:"HashPayload"`
	// PreviousReceiptReference links a return to the ΑΑΔΕ MARK(s) of the receipt(s) it
	// reverses. A return is linked either this way or by
	// FiscalisationData.CBPreviousReceiptReference.
	PreviousReceiptReference *GRPreviousReceiptReference `json:"PreviousReceiptReference,omitempty"`
}

// GRPreviousReceiptReference identifies the receipt(s) a return reverses by MARK.
type GRPreviousReceiptReference struct {
	InvoiceMarks InvoiceMarks `json:"invoiceMark"`
}

// InvoiceMarks is one or more ΑΑΔΕ MARKs. Viva takes a single MARK as a JSON number and
// several as an array, so it is sent in whichever form it has, and accepts either.
type InvoiceMarks []int64

// MarshalJSON sends one MARK as a number and several as an array.
func (m InvoiceMarks) MarshalJSON() ([]byte, error) {
	if len(m) == 1 {
		return json.Marshal(m[0])
	}
	return json.Marshal([]int64(m))
}

// UnmarshalJSON accepts a number or an array of numbers.
func (m *InvoiceMarks) UnmarshalJSON(b []byte) error {
	if one, ok := unmarshalOne(b, new(int64)); ok {
		*m = InvoiceMarks{*one.(*int64)}
		return nil
	}
	var many []int64
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("invoiceMark: want a number or an array of numbers: %w", err)
	}
	*m = many
	return nil
}

// GRHashAlg is the only hash algorithm Greek receipts use.
const GRHashAlg = "sha256"

// GRHashPayload builds the Greek HashPayload:
//
//	MerchantVATID-Series-AA-cbReceiptReference-cbReceiptMoment-Amount
//
// amount is the receipt total in minor units. A negative total (a return) keeps its minus
// sign, so the payload contains "--" before it; that is what Viva expects.
func GRHashPayload(merchantVATID, series string, aa int64, receiptReference string, moment time.Time, amount int64) string {
	return merchantVATID + "-" + series + "-" + strconv.FormatInt(aa, 10) + "-" +
		receiptReference + "-" + FormatFiscalMoment(moment) + "-" + formatGRAmount(amount)
}

// formatGRAmount formats minor units the way HashPayload expects: a dot decimal with
// trailing zeros trimmed but at least one decimal place (1780 → "17.8", 2300 → "23.0",
// 124 → "1.24", -12400 → "-124.0"). Checked against every HashPayload in Viva's Greek
// samples.
func formatGRAmount(minor int64) string {
	sign := ""
	abs := uint64(minor)
	if minor < 0 {
		sign = "-"
		abs = uint64(-minor) // also correct for math.MinInt64, by two's complement
	}
	whole, frac := abs/100, abs%100
	s := sign + strconv.FormatUint(whole, 10) + "."
	switch {
	case frac == 0:
		return s + "0"
	case frac%10 == 0:
		return s + strconv.FormatUint(frac/10, 10)
	case frac < 10:
		return s + "0" + strconv.FormatUint(frac, 10)
	default:
		return s + strconv.FormatUint(frac, 10)
	}
}

// GRReceiptQRURL returns the URL that every Greek fiscal receipt must print as a QR code,
// even when transmission failed: Viva's receipt host followed by the base64url-encoded
// (unpadded) SHA-256 digest of the receipt's HashPayload.
// Ref: https://developer.viva.com/viva-fiscalisation/core/
func GRReceiptQRURL(c Config, hashPayload string) string {
	base := "https://r.ft.ms/"
	if isDemo(c) {
		base = "https://r-sb.ft.ms/"
	}
	sum := sha256.Sum256([]byte(hashPayload))
	return base + base64.RawURLEncoding.EncodeToString(sum[:])
}
