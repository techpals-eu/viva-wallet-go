# 25 — Greek fiscalisation (myDATA / ΑΑΔΕ): retail receipts

**Severity:** High — the feature the branch is named for
**Scope (this plan):** retail receipts only: 11.1 ΑΛΠ, 11.2 ΑΠΥ, 11.4 return, plus the
Queue-Start and Zero receipts that onboarding requires. Invoices, delivery notes and 8.x
documents come later.
**Files:** new `fiscal.go`, `fiscal_gr.go`, `fiscal_test.go`, `testdata/fiscal_*.json`; edits
to `eftpos.go` and `README.md`
**Depends on:** 19 (ISV client). Plan 21 (EFT POS) is done; this plan extends it.

## Sources

Read on 2026-11-16 with curl, since WebFetch times out on developer.viva.com:

- https://developer.viva.com/viva-fiscalisation/fiscalisation-isv/
- https://developer.viva.com/viva-fiscalisation/core/
- https://developer.viva.com/viva-fiscalisation/receiptcase/, /chargeitemcase/, /payitemcase/
- https://developer.viva.com/viva-fiscalisation/greece/fiscalisation_for_online_sales/
- https://developer.viva.com/viva-fiscalisation/greece/samples/ (last updated 27/10/2026)
- https://developer.viva.com/downloads/eft-pos-api.yml (the OpenAPI behind the EFT POS page;
  `ISVCreateSaleTransactionOptions`, `ISVCreateRefundTransactionOptions`, `ISVGetSessionResponse`)

## How it works

Viva fiscalises on the merchant's behalf. Viva runs fiskaltrust as the signing middleware and
transmits to myDATA. We never talk to ΑΑΔΕ directly. Viva returns the signatures, MARK and QR
data, and **we must print all of it on the receipt**.

The same `fiscalisationData` object reaches Viva through one of two channels:

| Channel | When | How |
|---|---|---|
| **A. Terminal** | Card-present sale or refund at the EFT POS | `fiscalisationData` field on the existing `/ecr/isv/v1/transactions:sale` and `:refund` bodies. The result comes back in `GetSession` as `fiscalisationDetails`. |
| **B. Standalone sign** | Cash, Smart Checkout (online), Start/Zero receipts, refunds without a terminal | `POST {ApiUri}/fiscalization/v1/isv/receipts:sign?merchantId={uuid}`, OAuth bearer from the ISV token. A merchant variant exists at `/fiscalization/v1/receipts:sign`. The response is the fiscal result itself. |

Both use `ApiUri` (`demo-api.` / `api.vivapayments.com`). Neither uses `AppUri`.

For an online sale (Smart Checkout), the flow is: payment succeeds (webhook 1796 or
`GetISVOrder`), then channel B is called with `cbPayItems[].transactionId` set to the Viva
transaction id. The core page still says "Fiscal API is not yet available for web payments", but
the online-sales page documents it, so we have to confirm with Viva which is current.

Mutual exclusions from the OpenAPI spec:
- `fiscalisationData` must not be combined with `preauth: true`.
- Merchants on Viva fiscalisation **must not** also send `aadeProviderId`,
  `aadeProviderSignatureData` or `aadeProviderSignature`. Those belong to the other route, where
  a third-party provider (ΥΠΑΗΕΣ) signs. We do not add them.

## Prerequisites (per merchant, not code)

1. Viva enables the fiscal flags: send them the Merchant ID and ΑΦΜ. The account also needs the
   **AllowFiscalization** role.
2. **One Queue-Start receipt per merchant** before anything else. Without it, every sign call
   fails with `FiskaltrustSignReceiptUnknownState`.
3. If onboarding is incomplete, sign calls return HTTP 404 `FiskaltrustInvalidMerchantOrTerminal`
   (eventId 22254).

The SDK should make step 2 a single call and surface both errors as recognisable `*APIError`s.

## Wire format (verified against the docs)

### Case values

Every `ft…Case` is an Int64 sent as a **decimal JSON number**, built from a hex pattern.
Country GR = `0x4752`, version `2000`, so every Greek value starts with `0x4752_2000`.
Base `0x4752200000000000` = `5139205309155246080`.

| Field | Pattern | Notes |
|---|---|---|
| `ftReceiptCase` | `4752_2000_gggg_txcc` | gggg flags are bitwise-OR'd |
| `ftChargeItemCase` | `4752_2000_gggg_NNSV` | NN nature of VAT, S type of service, V VAT bucket |
| `ftPayItemCase` | `4752_2000_gggg_00PP` | PP payment type |

**Receipt case, txcc (retail subset):** `0001` POS receipt, `0002` payment transfer,
`0004` e-commerce, `2000` zero receipt, `4001` queue start, `4002` queue stop.

**Receipt flags, gggg:**

| Flag | Meaning |
|---|---|
| `0001` | late signing |
| `0002` | training |
| `0004` | void |
| `0008` | handwritten / custom-numbered (we supply Series and AA) |
| `0010` | small business |
| `0020` | receiver is business |
| `0040` | receiver known |
| `0080` | sale abroad |
| `0100` | refund |
| `0400` | transport info |
| `0800` | group by position |
| `8000` | ReceiptRequest (fetch the stored result after a timeout) |

**Charge item, S (type of service):**

| S | Meaning | myDATA category |
|---|---|---|
| `0` | unknown | category1_95 |
| `1` | goods | category1_1 |
| `2` | service | category1_3 |
| `3` | tip | — |
| `4` | voucher | — |
| `5` | catalog / self-produced | — |
| `6` | agency | — |
| `7` | own consumption | — |
| `8` | grant | — |
| `9` | receivable | — |
| `A` | cash transfer | — |

**Charge item, V (VAT bucket):** `1`/`2` reduced (13% or 6%), `3` normal 24%, `4`/`5` super-reduced
4%, `7` zero, `8` not taxable. `vatRate` itself is sent separately as fixed-point (`2400` = 24%).

**Charge item, NN:** `00` usual VAT. `1x`/`3x`/`4x`/`5x`/`6x`/`8x` are exemption articles, each
with a myDATA code. Out of scope for now; expose NN as a raw field so callers can set it.

**Charge item flags, gggg:** `0001` void, `0002` refund, `0004` discount/extra, `0008` downpayment,
`0020` takeaway, `8000` show-in-payments.

**Pay item, PP (GR):**

| PP | Meaning |
|---|---|
| `01` | cash |
| `02` | EFTPOS / any Viva electronic method |
| `03` | cheque |
| `06` | voucher |
| `09` | on credit (επί πιστώσει) |
| `0A` | bank transfer / IRIS / RF |
| `0B` | foreign bank transfer |

**Pay item flags, gggg:** `0001` void, `0002` refund, `0010` foreign currency, `0020` change,
`0040` tip, `0080` electronic, `0100` interface-verified, `8000` show-in-charge-items.

Test vectors (decimal from the samples page, decoded):

| Decimal | Hex suffix | Meaning |
|---|---|---|
| `5139205309155770369` | `0008_0001` | 11.1/11.2 receipt (custom-numbered POS receipt) |
| `5139205309172547585` | `0108_0001` | 11.4 return (refund + custom-numbered) |
| `5139205309155262465` | `0000_4001` | Queue-Start |
| `5139205309155254272` | `0000_2000` | Zero receipt |
| `5139205309155246099` | `0000_0013` | charge: goods, 24% |
| `5139205309155246115` | `0000_0023` | charge: service, 24% |
| `5139205309155377171` | `0002_0013` | charge: refunded goods, 24% |
| `5139205309155246081` | `0000_0001` | pay: cash |
| `5139205309155246082` | `0000_0002` | pay: EFTPOS |
| `5139205309155246089` | `0000_0009` | pay: on credit |

### Amounts, quantities, times

- Amounts (`amount`, `vatAmount`, `cbReceiptAmount`) are in minor units. Refunds are negative.
- `quantity` is ×100 (1 → `100`, refund → `-100`).
- `vatRate` is ×100 (24% → `2400`).
- `currencyCode` is `978`. The samples mix number and string; send the string `"978"` as most
  samples do, and confirm.
- Times are ISO-8601 UTC to the second: `2025-08-27T16:11:06Z`.
- `sessionId` is a **UUID v4**. Reusing one returns the stored result, which makes it the
  idempotency key. The existing `NewSessionID` returns 32 hex characters without dashes, which
  is not a UUID, so fiscal requests need a separate `NewFiscalSessionID`.

### `ftReceiptCaseData.GR` and HashPayload

```json
"GR": { "MerchantVATID": "099565360", "Series": "AB", "AA": 123,
        "HashAlg": "sha256",
        "HashPayload": "099565360-AB-123-ref1-2026-07-10T11:28:00Z-17.8" }
```

- HashPayload is `MerchantVATID-Series-AA-cbReceiptReference-cbReceiptMoment-Amount`.
- Amount is the receipt total in euros with a dot decimal, trailing zeros trimmed but **at least
  one decimal place**: 1780 → `17.8`, 2300 → `23.0`, 124 → `1.24`, −12400 → `-124.0`.
- A negative amount gives `--` before it (`…Z--1.24`). That is correct, so keep it.
- The moment must be byte-identical to `cbReceiptMoment`.
- For an 11.4 return that is linked to its original, `GR.PreviousReceiptReference.invoiceMark`
  carries the original's MARK, or `cbPreviousReceiptReference` the original's
  `cbReceiptReference`. **Both take one value or an array**: a single MARK is a number and
  several are an array, and likewise a single reference is a string and several are an array.
  The SDK models them as `InvoiceMarks` and `ReceiptReferences`, which send whichever form fits
  and accept either.
- **Series and AA are the caller's numbering.** AA must be sequential per series and never
  reused. The SDK cannot own that counter, because it has no storage, so it takes them as inputs
  and says so loudly.

The SDK builds HashPayload itself from the same values it puts in the body, so the two can never
disagree. Callers do not write it.

### Response

```json
{ "ftState": 5139205309155246080, "ftStateData": {}, "ftReceiptIdentification": "ft5#0-5",
  "ftCashBoxIdentification": "fiskaltrust1", "ftSignatures": [ … ], "transmissionIndicator": 0 }
```

- In the terminal session response, `ftState` and `ftSignatureType` are **strings**. In the sign
  response they are numbers. Decode both with a type that accepts either.
- `ftState`:
  - `…0000_0000` ready
  - `…0000_0001` out of operation
  - `…0000_0002` SCU (signing unit) out of service
  - `…0000_0040` message pending
  - `…0000_0100` daily closing due
  - low 32 bits `EEEE_EEEE` error
  - low 32 bits `FFFF_FFFF` fail
- `transmissionIndicator`:
  - `0` OK
  - `1` TRANSMISSION_FAILURE_1
  - `2` **or missing** TRANSMISSION_FAILURE_2

  For 1 and 2, print the literal text on the receipt. Viva retransmits later, so no follow-up
  receipt is needed. Use a pointer, so that missing maps to failure 2 and not to 0.
- `ftSignatures[]`:
  - Fields are `caption`, `data`, `ftSignatureFormat` (0 unknown, 1 text, 2 link, 3 QR, …) and
    `ftSignatureType`.
  - Print each one in the order received: the caption on one line, then the data in the given
    format.
  - Typical captions are `invoiceUid`, `invoiceMark` (the MARK), `authenticationCode`, a receipt
    URL as QR, and `Μοναδικός αριθμός παραστατικού`.
- **The QR code must always be generated by us**, even on failure:
  - URL: `https://r-sb.ft.ms/` (demo) or `https://r.ft.ms/` (production), followed by the
    base64url SHA-256 of HashPayload.
  - Use the raw 32-byte digest, no padding. The result is always 43 characters.
  - Verified: `099565360-ab-12-ab2-2026-09-15T16:11:06Z-1.24` →
    `dz_87y5EqFh0TVlXDPUUFvLHWsoK174HXqFp6ewdwCQ`.

## API to add

```go
// fiscal.go — country-neutral wire types
type FiscalisationData struct {
    CBChargeItems   []ChargeItem       `json:"cbChargeItems"`   // [] not null for Start/Zero
    CBPayItems      []PayItem          `json:"cbPayItems"`
    CBUser          *FiscalUser        `json:"cbUser,omitempty"`
    CBCustomer      *FiscalCustomer    `json:"cbCustomer,omitempty"`
    CBArea          string             `json:"cbArea,omitempty"`
    CBReceiptMoment string             `json:"cbReceiptMoment"`  // set by the builder
    CBReceiptReference         string  `json:"cbReceiptReference"`
    CBPreviousReceiptReference string  `json:"cbPreviousReceiptReference,omitempty"`
    CBReceiptAmount int64              `json:"cbReceiptAmount,omitempty"`
    CurrencyCode    string             `json:"currencyCode"`
    FTReceiptCase   ReceiptCase        `json:"ftReceiptCase"`
    FTReceiptCaseData *ReceiptCaseData `json:"ftReceiptCaseData,omitempty"`
}
type SignReceiptRequest struct { SessionID string `json:"sessionId"`; FiscalisationData }
type FiscalResult struct { FTState FiscalState; FTReceiptIdentification, FTCashBoxIdentification string;
    FTSignatures []FiscalSignature; TransmissionIndicator *int }
func (r *FiscalResult) Transmission() Transmission     // OK / Failure1 / Failure2 (nil → Failure2)
func (r *FiscalResult) Mark() string                    // ftSignatures caption "invoiceMark", "" if none

// fiscal_gr.go — Greece
type ReceiptCase, ChargeItemCase, PayItemCase int64
func GRReceiptCase(txcc uint16, flags uint16) ReceiptCase
func GRChargeItemCase(nn uint8, s ServiceType, v VATBucket, flags uint16) ChargeItemCase
func GRPayItemCase(pp PaymentType, flags uint16) PayItemCase
// plus named constants for every value in the test-vector table above

type GRReceipt struct {           // what a caller fills in; the builder does the rest
    MerchantVATID, Series string; AA int64
    Reference string; Moment time.Time
    Lines []ChargeItem; Payments []PayItem
    PreviousReference, PreviousMark string    // 11.4 only
}
func (r GRReceipt) Sale() (FiscalisationData, error)    // 11.1/11.2: validates, sets case, HashPayload
func (r GRReceipt) Return() (FiscalisationData, error)  // 11.4: negates, sets refund flags
func GRHashPayload(vatID, series string, aa int64, ref string, moment time.Time, amount int64) string
func GRReceiptQRURL(c Config, hashPayload string) string
func NewFiscalSessionID() (string, error)               // UUID v4 from crypto/rand

// channel B
func (c *ISVClient) SignReceipt(merchantID string, req SignReceiptRequest) (*FiscalResult, error)
func (c *ISVClient) StartFiscalQueue(merchantID string) (*FiscalResult, error) // 0x4001
func (c *ISVClient) ZeroReceipt(merchantID string) (*FiscalResult, error)      // 0x2000
func (c *OAuthClient) SignReceipt(req SignReceiptRequest) (*FiscalResult, error) // merchant variant

// channel A, in eftpos.go
ECRSaleRequest.FiscalisationData   *FiscalisationData `json:"fiscalisationData,omitempty"`
ECRRefundRequest.FiscalisationData *FiscalisationData `json:"fiscalisationData,omitempty"`
ECRSession.FiscalisationDetails    *FiscalResult      `json:"fiscalisationDetails,omitempty"`
```

Builder validation, which returns an error before any request is sent:
- The charge-item amounts sum to the pay-item amounts, which sum to `cbReceiptAmount`.
- Every line has quantity ≠ 0, a position, a description and a case value.
- For a return, every amount and quantity is ≤ 0.
- A sale is rejected if it would be combined with `preauth` (that flag isn't modelled today, so
  this check isn't needed yet).

## Steps

Status (2026-11-16): steps 1–7 are done (step 7: README section "Greek fiscalisation"), in `fiscal.go`, `fiscal_gr.go`, `fiscal_sign.go`, the
`eftpos.go` fields, and `fiscal*_test.go` with 7 sample fixtures. Step 6 found that a return can
reference several MARKs or receipt references, which the first types could not hold. The
`GRReceipt` builder is done (`fiscal_gr_receipt.go`: `Sale`, `OnlineSale`, `Return`). Step 8
has a runner, `example/fiscal` (`make fiscal-demo STEP=…`), that prints full error bodies.

1. **Types and case builders** (offline, no HTTP): `fiscal.go`, `fiscal_gr.go`. Test that every
   decimal in the test-vector table round-trips from its builder call.
2. **HashPayload and QR**: test against every HashPayload string in the samples page (copy them
   into a table test), plus the QR vector above.
3. **Response decoding**: fixtures for a numeric sign response, a string-typed session
   `fiscalisationDetails`, a response with `transmissionIndicator` missing, and the zero-receipt
   response.
4. **`SignReceipt`, `StartFiscalQueue`, `ZeroReceipt`** on `ISVClient`, with httptest checks that
   assert the path, the `merchantId` query, the bearer header and the body. Then the
   `OAuthClient` variant.
5. **Terminal fields** in `eftpos.go`: request fixture from the samples' POS-defined tipping
   example, session fixture with `fiscalisationDetails`.
6. **Samples as fixtures**: put the 11.1, 11.2, 11.4 (unlinked, linked, with MARK), Queue-Start
   and Zero JSON in `testdata/fiscal_gr_*.json`. Each builder's output must match them modulo
   key order.
7. **README** "Supported Endpoints" and a usage section. Example sequence: Start queue once per
   merchant → terminal sale with `fiscalisationData` → print signatures + QR. Online sequence:
   webhook 1796 → `SignReceipt` with the transactionId.
8. **Demo run** with real demo credentials: Start receipt, Zero receipt, one 11.1 cash receipt
   via `SignReceipt`, one terminal sale on the Viva Terminal demo app with fiscalisation enabled.

Steps 1–3 need nothing from Viva and can start now. Steps 4–5 can be written and tested against
fixtures, but only step 8 proves them.

## Open questions for Viva

Drafted for Viva in Greek in [viva-questions-fiscalisation-el.md](viva-questions-fiscalisation-el.md).

Answered (2026-11-16):

- **Online sales:** `receipts:sign` for web payments is in **beta** and may be used. Which
  receipt type to use (`0001`, `0004` or `0002`) is still open; see 1 below.
- **Card refund PP is `04`** (`GRPaymentCardRefund`), even though the Greek PP table does not
  list it. Card sales stay on `02`.
- **`merchantId`** on the ISV sign endpoint is the same merchant id used by ISV orders and ECR.
- **No extra OAuth scope** is needed for `/fiscalization`.
- **`currencyCode` is a string**, the 3-digit ISO 4217 numeric code (`"978"`). Some samples
  send a number; the SDK always sends the string.
- **QR code:** our system always prints it, whatever the terminal prints.

- **Online receipt type is `0004`** (`GRReceiptECommerce`).
- **Flag 0008 is not for normal receipts.** It marks receipts issued offline with our own
  Series/AA and signed afterwards (late signing). Receipts signed at the time of sale use flag
  `0000`, even though every Greek sample sets 0008. So:
  - an 11.1/11.2 receipt is `0x0000_0001`
  - an 11.4 return is `0x0100_0001` (`5139205309172023297`)
  - an online receipt is `0x0000_0004`
  The sample fixtures keep 0008 because they reproduce Viva's samples, not what we send.
- **Terminal numbering:** Viva says the terminal and its backend manage Series/AA and the flag
  automatically. Viva's terminal samples still send `ftReceiptCaseData.GR`. **Check in the demo
  run** whether the terminal needs, uses or overrides it.
- **The online-sales charge item** with V=2 and `vatRate: 0` is not an error. Per Viva it
  belongs to a delivery note (9.2), which carries no values. Goods at 24% are V=3 with
  `vatRate: 2400`.

Still open:

1. **Error bodies.** What do `FiskaltrustSignReceiptUnknownState` and 404
   `FiskaltrustInvalidMerchantOrTerminal` look like? `IsFiscalQueueNotStarted` and
   `IsFiscalisationNotEnabled` currently match on the name anywhere in the body. The demo run
   will show them, so this no longer needs to go to Viva.

Doc errors already found. They are noted here so nobody "fixes" the code to match them:
- The ChargeItemCase page's "goods 24%" example shows the pattern `…0000_0013` but gives the
  decimal of `…0000_0003`. Use `…0013` (`5139205309155246099`), which every 11.1 sample uses.
- The second QR example on the core page (`Y2M0NjI2…`) matches no encoding of its input. The
  first example verifies, so test against that one.
- The PayItemCase page titles its Greek card example "PP=04", but the value it shows is PP=02.
  (Card *refunds* do use PP=04, as Viva confirmed; card *sales* use 02.)

## Out of scope (later plans)

- Invoices (1.x, 2.x), credit invoices (5.x), delivery notes (9.x), 8.x receipts, VAT exemption
  NN codes, B2B `cbCustomer` rules and myDATA classification.
- Daily/monthly closing receipts (`2011`, `2012`). Viva signals them via the `ftState`
  "daily closing due" flag, so expose the flag now and add the receipts later.
- Third-party provider (ΥΠΑΗΕΣ) signing via `aadeProvider*`. That route is mutually exclusive
  with this plan.
- Rendering receipts. The SDK returns what to print; printing is the PMS's job.
- `context.Context` (plan 09). Signing is a natural first caller, but adding context here alone
  would split the API style.
