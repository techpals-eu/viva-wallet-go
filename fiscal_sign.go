package vivawallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Receipt signing: fiscalising a receipt on its own, without a terminal. Use it for cash
// sales, for Smart Checkout payments once they succeed (set PayItem.TransactionID), and for
// the Queue-Start and Zero receipts. Card-present sales are fiscalised by the terminal
// instead, through ECRSaleRequest.FiscalisationData.
//
// The account must have Viva's AllowFiscalization role, and Viva must have enabled
// fiscalisation for the merchant.
// Ref: https://developer.viva.com/viva-fiscalisation/greece/fiscalisation_for_online_sales/

// SignReceipt fiscalises receipt on behalf of merchantID and returns what must be printed.
// merchantID is the same merchant id the ISV order and EFT POS calls take. The ISV client's
// existing token works; fiscalisation needs no extra OAuth scope, only Viva's
// AllowFiscalization role on the account.
//
// sessionID must be a UUID v4 (see NewFiscalSessionID) and is the idempotency key: after a
// timeout or network error, call SignReceipt again with the same sessionID and receipt to
// get the stored result rather than signing twice. Persist it before the first attempt.
// Ref: https://developer.viva.com/viva-fiscalisation/greece/fiscalisation_for_online_sales/
func (c *ISVClient) SignReceipt(merchantID, sessionID string, receipt FiscalisationData) (*FiscalResult, error) {
	if merchantID == "" {
		return nil, errors.New("vivawallet: SignReceipt needs a merchant id")
	}
	return signReceipt(c.Post, isvSignReceiptUri(c.Config, merchantID), sessionID, receipt)
}

// StartFiscalQueue sends the Queue-Start receipt that initialises merchantID's fiscal queue.
// Send it once per merchant, after Viva has enabled fiscalisation and before any other
// receipt; until then signing fails with an error IsFiscalQueueNotStarted recognises.
// Ref: https://developer.viva.com/viva-fiscalisation/greece/samples/
func (c *ISVClient) StartFiscalQueue(merchantID string) (*FiscalResult, error) {
	return c.signGROperation(merchantID, GRReceiptQueueStart, "queue-start")
}

// ZeroReceipt sends a Zero receipt for merchantID: a no-turnover receipt that confirms the
// queue can sign, and prints any pending message or clears an SCU failure. It creates no
// document and nothing is sent to myDATA; check the result's State. Send one when you need
// the answer (after onboarding, at the start of a day, after a failed sign), never in a
// polling loop or per transaction.
// Ref: https://developer.viva.com/viva-fiscalisation/greece/samples/
func (c *ISVClient) ZeroReceipt(merchantID string) (*FiscalResult, error) {
	return c.signGROperation(merchantID, GRReceiptZero, "zero")
}

func (c *ISVClient) signGROperation(merchantID string, t GRReceiptType, refPrefix string) (*FiscalResult, error) {
	sessionID, err := NewFiscalSessionID()
	if err != nil {
		return nil, err
	}
	// Each one needs a fresh reference as well as a fresh session id.
	return c.SignReceipt(merchantID, sessionID, grOperationReceipt(t, refPrefix, sessionID, time.Now()))
}

func grOperationReceipt(t GRReceiptType, refPrefix, sessionID string, now time.Time) FiscalisationData {
	moment := FormatFiscalMoment(now)
	return FiscalisationData{
		CBReceiptReference: refPrefix + "-" + moment + "-" + sessionID[:8],
		CBReceiptMoment:    moment,
		CurrencyCode:       FiscalCurrencyEUR,
		FTReceiptCase:      GRReceiptCase(t, 0),
	}
}

// SignReceipt fiscalises receipt for the authenticated merchant. It is the merchant variant
// of ISVClient.SignReceipt; see there for sessionID.
// Ref: https://developer.viva.com/viva-fiscalisation/greece/fiscalisation_for_online_sales/
func (c *OAuthClient) SignReceipt(sessionID string, receipt FiscalisationData) (*FiscalResult, error) {
	return signReceipt(c.Post, signReceiptUri(c.Config), sessionID, receipt)
}

func signReceipt(post func(string, *bytes.Reader, interface{}) error, uri, sessionID string, receipt FiscalisationData) (*FiscalResult, error) {
	if sessionID == "" {
		return nil, errors.New("vivawallet: SignReceipt needs a session id; see NewFiscalSessionID")
	}
	body, err := signReceiptBody(sessionID, receipt)
	if err != nil {
		return nil, err
	}

	result := &FiscalResult{}
	if err := post(uri, bytes.NewReader(body), result); err != nil {
		return nil, err
	}
	return result, nil
}

// signReceiptBody is the receipt with sessionId added at the top level. It is spliced in,
// rather than built by embedding FiscalisationData in a struct, because the embedded
// MarshalJSON would be promoted and silently drop sessionId.
func signReceiptBody(sessionID string, receipt FiscalisationData) ([]byte, error) {
	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to encode receipt: %w", err)
	}
	id, err := json.Marshal(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode session id: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`{"sessionId":`)
	b.Write(id)
	b.WriteByte(',')
	b.Write(data[1:]) // data is a non-empty JSON object: it always has cbChargeItems
	return b.Bytes(), nil
}

// The ":sign" suffix is custom-method syntax and part of the path; it must not be
// percent-encoded.
func isvSignReceiptUri(c Config, merchantID string) string {
	return fmt.Sprintf("%s/fiscalization/v1/isv/receipts:sign?merchantId=%s", ApiUri(c), url.QueryEscape(merchantID))
}

func signReceiptUri(c Config) string {
	return fmt.Sprintf("%s/fiscalization/v1/receipts:sign", ApiUri(c))
}

// fiscalEventIDNotEnabled is the eventId Viva documents for a merchant whose fiscalisation
// onboarding is incomplete.
const fiscalEventIDNotEnabled = 22254

// IsFiscalQueueNotStarted reports whether err means the merchant's fiscal queue has not been
// initialised: send StartFiscalQueue first.
//
// Viva documents the error by name (FiskaltrustSignReceiptUnknownState) but not the shape
// of its body, so this matches the name anywhere in the response.
func IsFiscalQueueNotStarted(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && strings.Contains(apiErr.Body, "FiskaltrustSignReceiptUnknownState")
}

// IsFiscalisationNotEnabled reports whether err means Viva has not enabled fiscalisation for
// the merchant or terminal, which is resolved by Viva rather than by retrying.
//
// Viva documents it as HTTP 404 FiskaltrustInvalidMerchantOrTerminal, eventId 22254.
func IsFiscalisationNotEnabled(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		return false
	}
	return apiErr.EventID == fiscalEventIDNotEnabled ||
		strings.Contains(apiErr.Body, "FiskaltrustInvalidMerchantOrTerminal")
}
