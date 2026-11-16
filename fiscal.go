package vivawallet

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Viva Fiscalisation wire types. They are country-neutral: what makes a receipt Greek is the
// case values (see fiscal_gr.go) and ReceiptCaseData.GR.
//
// Amounts are in minor units (cents), and quantities and VAT rates are fixed-point ×100:
// a quantity of 1 is sent as 100, and 24% VAT as 2400. Refund lines carry negative amounts
// and quantities.
// Ref: https://developer.viva.com/viva-fiscalisation/core/

// FiscalCurrencyEUR is the euro's ISO 4217 numeric code as fiscalisation requests carry it:
// a string, the type Viva confirmed, although some of its samples send a number.
const FiscalCurrencyEUR = "978"

// fiscalMomentLayout is the only timestamp format fiscalisation accepts: UTC, to the second.
// cbReceiptMoment must also appear byte-for-byte in the Greek HashPayload, so every moment
// goes through FormatFiscalMoment.
const fiscalMomentLayout = "2006-01-02T15:04:05Z"

// FormatFiscalMoment formats t as a fiscalisation timestamp, e.g. "2025-08-27T16:11:06Z". It
// converts to UTC and drops sub-second precision.
func FormatFiscalMoment(t time.Time) string {
	return t.UTC().Format(fiscalMomentLayout)
}

// ChargeItem is one line of a receipt: a product, service, tip or fee.
type ChargeItem struct {
	// Amount is the gross line total (unit price × quantity, VAT included), in minor units.
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	// Position is the line number, starting at 1.
	Position int `json:"position"`
	// Quantity is ×100: send 1 as 100, 2.5 as 250, a returned unit as -100.
	Quantity int64 `json:"quantity"`
	// VATRate is ×100: 24% is 2400.
	VATRate int64 `json:"vatRate"`
	// VATAmount is the VAT included in Amount, in minor units.
	VATAmount        int64          `json:"vatAmount"`
	FTChargeItemCase ChargeItemCase `json:"ftChargeItemCase"`
	CurrencyCode     string         `json:"currencyCode"`
	// Moment is the time of the line, formatted with FormatFiscalMoment.
	Moment         string `json:"moment,omitempty"`
	Unit           string `json:"unit,omitempty"`
	UnitPrice      *int64 `json:"unitPrice,omitempty"`
	UnitQuantity   *int64 `json:"unitQuantity,omitempty"`
	ProductNumber  string `json:"productNumber,omitempty"`
	ProductBarcode string `json:"productBarcode,omitempty"`
}

// PayItem is one payment towards a receipt.
type PayItem struct {
	// Amount is in minor units. Across all pay items it should equal the receipt total.
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	// Quantity is ×100 and usually omitted; refund samples send -100.
	Quantity      int64       `json:"quantity,omitempty"`
	FTPayItemCase PayItemCase `json:"ftPayItemCase"`
	CurrencyCode  string      `json:"currencyCode"`
	Moment        string      `json:"moment,omitempty"`
	// TransactionID is the Viva transaction this payment was made with, e.g. a Smart
	// Checkout payment being fiscalised afterwards.
	TransactionID string `json:"transactionId,omitempty"`
}

// FiscalUser identifies who issued the receipt.
type FiscalUser struct {
	MerchantName string         `json:"merchantName,omitempty"`
	Cashier      *FiscalCashier `json:"cashier,omitempty"`
}

// FiscalCashier is the cashier part of FiscalUser.
type FiscalCashier struct {
	Name string `json:"name,omitempty"`
	ID   int64  `json:"id,omitempty"`
}

// FiscalCustomer identifies a business, or an identified retail, customer.
type FiscalCustomer struct {
	VATID   string `json:"customerVATId,omitempty"`
	Country string `json:"customerCountry,omitempty"`
	Name    string `json:"customerName,omitempty"`
	Street  string `json:"customerStreet,omitempty"`
	Zip     string `json:"customerZip,omitempty"`
	City    string `json:"customerCity,omitempty"`
}

// ReceiptCaseData carries the per-country data a receipt needs, keyed by country.
type ReceiptCaseData struct {
	GR *GRReceiptCaseData `json:"GR,omitempty"`
}

// FiscalisationData is a receipt to be fiscalised. It is sent either embedded in a terminal
// sale or refund, or on its own to the receipt signing endpoint.
type FiscalisationData struct {
	// CBChargeItems and CBPayItems are sent as [] rather than null when empty, as the
	// Queue-Start and Zero receipts require.
	CBChargeItems []ChargeItem    `json:"cbChargeItems"`
	CBPayItems    []PayItem       `json:"cbPayItems"`
	CBUser        *FiscalUser     `json:"cbUser,omitempty"`
	CBCustomer    *FiscalCustomer `json:"cbCustomer,omitempty"`
	CBArea        string          `json:"cbArea,omitempty"`
	// CBReceiptMoment is formatted with FormatFiscalMoment.
	CBReceiptMoment string `json:"cbReceiptMoment"`
	// CBReceiptReference is the receipt's unique id in your own system.
	CBReceiptReference string `json:"cbReceiptReference"`
	// CBPreviousReceiptReference links a return or void to the CBReceiptReference of the
	// receipt(s) it reverses.
	CBPreviousReceiptReference ReceiptReferences `json:"cbPreviousReceiptReference,omitempty"`
	CBReceiptAmount            int64             `json:"cbReceiptAmount,omitempty"`
	CurrencyCode               string            `json:"currencyCode"`
	FTReceiptCase              ReceiptCase       `json:"ftReceiptCase"`
	FTReceiptCaseData          *ReceiptCaseData  `json:"ftReceiptCaseData,omitempty"`
}

// MarshalJSON sends nil item slices as [].
func (d FiscalisationData) MarshalJSON() ([]byte, error) {
	type plain FiscalisationData
	p := plain(d)
	if p.CBChargeItems == nil {
		p.CBChargeItems = []ChargeItem{}
	}
	if p.CBPayItems == nil {
		p.CBPayItems = []PayItem{}
	}
	return json.Marshal(p)
}

// NewFiscalSessionID returns a random UUID v4, the format fiscalisation requires for
// sessionId. A session id is also the idempotency key: resending a request with the same
// one returns the stored result of the first instead of signing again. Note that
// NewSessionID, for EFT POS requests, does not produce a UUID.
func NewFiscalSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate fiscal session id: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// fiscalInt is an Int64 that Viva sends as a JSON number in some responses and as a string
// in others: ftState and ftSignatureType are numbers from the signing endpoint but strings
// in an EFT POS session's fiscalisationDetails.
type fiscalInt int64

func (f *fiscalInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		*f = 0
		return nil
	}
	s := string(b)
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid fiscal integer %s: %w", b, err)
	}
	*f = fiscalInt(v)
	return nil
}

// FiscalState is the state of the merchant's fiscal queue after a request (ftState). The
// high 32 bits are the country and version, and the low 32 are flags.
type FiscalState int64

// UnmarshalJSON accepts ftState as a number or a numeric string.
func (s *FiscalState) UnmarshalJSON(b []byte) error {
	var v fiscalInt
	if err := v.UnmarshalJSON(b); err != nil {
		return err
	}
	*s = FiscalState(v)
	return nil
}

func (s FiscalState) flags() uint32 { return uint32(uint64(s)) }

// Ready reports whether the queue is in its normal state, with no flags set.
func (s FiscalState) Ready() bool { return s.flags() == 0 }

// OutOfOperation reports that the queue is not started or has been stopped.
func (s FiscalState) OutOfOperation() bool { return !s.failed() && s.flags()&0x1 != 0 }

// SCUOutOfService reports that the signature creation unit failed for at least one receipt.
// A Zero receipt clears it.
func (s FiscalState) SCUOutOfService() bool { return !s.failed() && s.flags()&0x2 != 0 }

// MessagePending reports that the queue has a message for the cashier. A Zero receipt
// prints it.
func (s FiscalState) MessagePending() bool { return !s.failed() && s.flags()&0x40 != 0 }

// DailyClosingDue reports that more than two days have passed since the last daily closing.
func (s FiscalState) DailyClosingDue() bool { return !s.failed() && s.flags()&0x100 != 0 }

// Error reports that the last request was queued but not signed. The reason is in the
// response's signatures.
func (s FiscalState) Error() bool { return s.flags() == 0xEEEEEEEE }

// Fail reports that the last request was not persisted at all. The reason is in the
// response's signatures.
func (s FiscalState) Fail() bool { return s.flags() == 0xFFFFFFFF }

func (s FiscalState) failed() bool { return s.Error() || s.Fail() }

// String returns the state as the hex pattern Viva documents it in, e.g. 0x4752200000000000.
func (s FiscalState) String() string { return fmt.Sprintf("%#016x", uint64(s)) }

// SignatureFormat is how a signature's data must be printed (ftSignatureFormat).
type SignatureFormat int

const (
	SignatureFormatUnknown SignatureFormat = 0
	SignatureFormatText    SignatureFormat = 1
	SignatureFormatLink    SignatureFormat = 2
	SignatureFormatQRCode  SignatureFormat = 3
	SignatureFormatCode128 SignatureFormat = 4
)

// FiscalSignature is one item to print in the receipt's signature block: Caption on one
// line, then Data rendered as Format says.
type FiscalSignature struct {
	Caption string          `json:"caption"`
	Data    string          `json:"data"`
	Format  SignatureFormat `json:"ftSignatureFormat"`
	Type    fiscalInt       `json:"ftSignatureType"`
}

// Transmission is whether the receipt reached the tax authority.
type Transmission int

const (
	// TransmissionOK means the receipt was fiscalised and transmitted.
	TransmissionOK Transmission = 0
	// TransmissionFailure1 means fiscalisation was temporarily unavailable. Viva transmits
	// the receipt later; the receipt must be printed with "TRANSMISSION_FAILURE_1".
	TransmissionFailure1 Transmission = 1
	// TransmissionFailure2 means the tax authority was unreachable. Viva transmits the
	// receipt later; the receipt must be printed with "TRANSMISSION_FAILURE_2".
	TransmissionFailure2 Transmission = 2
)

// String returns the text that must be printed on the receipt for a failure, or "" for OK.
func (t Transmission) String() string {
	switch t {
	case TransmissionOK:
		return ""
	case TransmissionFailure1:
		return "TRANSMISSION_FAILURE_1"
	default:
		return "TRANSMISSION_FAILURE_2"
	}
}

// FiscalResult is the outcome of fiscalising a receipt: the response of the signing
// endpoint, or the fiscalisationDetails of an EFT POS session.
type FiscalResult struct {
	State                    FiscalState       `json:"ftState"`
	StateData                json.RawMessage   `json:"ftStateData,omitempty"`
	ReceiptIdentification    string            `json:"ftReceiptIdentification"`
	CashBoxIdentification    string            `json:"ftCashBoxIdentification"`
	Signatures               []FiscalSignature `json:"ftSignatures"`
	RawTransmissionIndicator *int              `json:"transmissionIndicator,omitempty"`
}

// Transmission reports whether the receipt reached the tax authority. A missing
// transmissionIndicator counts as TransmissionFailure2, as Viva documents it, and so does
// any value it does not document.
func (r *FiscalResult) Transmission() Transmission {
	if r.RawTransmissionIndicator == nil {
		return TransmissionFailure2
	}
	switch *r.RawTransmissionIndicator {
	case 0:
		return TransmissionOK
	case 1:
		return TransmissionFailure1
	default:
		return TransmissionFailure2
	}
}

// Mark returns the ΑΑΔΕ MARK of the receipt (the "invoiceMark" signature), or "" when there
// is none, e.g. for a Zero receipt or a failed transmission.
func (r *FiscalResult) Mark() string {
	for _, s := range r.Signatures {
		if s.Caption == "invoiceMark" {
			return s.Data
		}
	}
	return ""
}

// ReceiptReferences is one or more cbReceiptReference values. Viva takes a single reference
// as a JSON string and several as an array, so it is sent in whichever form it has, and
// accepts either.
type ReceiptReferences []string

// MarshalJSON sends one reference as a string and several as an array.
func (r ReceiptReferences) MarshalJSON() ([]byte, error) {
	if len(r) == 1 {
		return json.Marshal(r[0])
	}
	return json.Marshal([]string(r))
}

// UnmarshalJSON accepts a string or an array of strings.
func (r *ReceiptReferences) UnmarshalJSON(b []byte) error {
	if one, ok := unmarshalOne(b, new(string)); ok {
		*r = ReceiptReferences{*one.(*string)}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("cbPreviousReceiptReference: want a string or an array of strings: %w", err)
	}
	*r = many
	return nil
}

// unmarshalOne decodes b into v unless b is a JSON array or null, reporting whether it did.
func unmarshalOne(b []byte, v interface{}) (interface{}, bool) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] == '[' || bytes.Equal(b, []byte("null")) {
		return nil, false
	}
	if err := json.Unmarshal(b, v); err != nil {
		return nil, false
	}
	return v, true
}
