# Viva Wallet Go SDK

viva-wallet is an SDK for connecting to the payment provider Viva Wallet developed by [TechPals](https://techpals.eu).

[![Get started with TechPals](./cover.jpg)](https://techpals.eu/?utm_source=viva-wallet-go&utm_medium=github)

## Supported Endpoints

- [Authentication (OAuth)](https://developer.vivawallet.com/apis-for-payments/payment-api/#section/Authentication)
- [Payments](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments)
    - [Create](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments/paths/~1checkout~1v2~1orders/post)
    - [Retrieve](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/get)
    - [Update](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/patch)
    - [Cancel](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments-(Deprecated)/paths/~1api~1orders~1{orderCode}/delete)
- [Transactions](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions)
  - [Create](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1api~1transactions~1{transaction_id}/post)
  - [Retrieve](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions/paths/~1checkout~1v2~1transactions~1{transactionId}/get)
  - [Cancel](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1api~1transactions~1{transaction_id}/delete)
  - [Cancel Partial Authorization](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions-(Deprecated)/paths/~1acquiring~1v1~1transactions~1{transactionId}/delete)
- [Wallet](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Balance-Transfer)
  - [Balance Tranfer](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Balance-Transfer)
  - [Retrieve Wallets](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Retrieve-Wallet)
- [ISV Payments](https://developer.viva.com/isv-partner-program/payment-isv-api/)
  - [Create](https://developer.viva.com/isv-partner-program/payment-isv-api/) — `CreateISVOrder`
  - [Retrieve](https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Payments/paths/~1api~1orders~1{order_code}/get) — `GetISVOrder`
- [EFT POS Cloud REST (ISV)](https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/)
  - [Sale](https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Make-Transactions/paths/~1ecr~1isv~1v1~1transactions:sale/post) — `Sale`
  - [Refund](https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Make-Transactions/paths/~1ecr~1isv~1v1~1transactions:refund/post) — `Refund`
  - [Retrieve Session](https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Session-Details/paths/~1ecr~1isv~1v1~1sessions~1{sessionId}/get) — `GetSession`
- [Webhooks](https://developer.viva.com/webhooks-for-payments/)
  - [Retrieve webhook key](https://developer.viva.com/apis-for-payments/payment-api/#tag/Webhooks/paths/~1api~1messages~1config~1token/get) — `BasicAuthClient.GetWebhookVerificationKey`
  - [Retrieve webhook key for ISV](https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Webhooks/paths/~1isv~1v1~1webhooks~1token/get) — `ISVClient.GetWebhookVerificationKey`
  - Payloads: [Transaction Payment Created](https://developer.viva.com/webhooks-for-payments/transaction-payment-created/), [Transaction Reversal Created](https://developer.viva.com/webhooks-for-payments/transaction-reversal-created/), [Transaction Failed](https://developer.viva.com/webhooks-for-payments/transaction-failed/), [Account Connected](https://developer.viva.com/webhooks-for-payments/account-connected/), [Account Verification Status Changed](https://developer.viva.com/webhooks-for-payments/account-verification-status-changed/) — `ParseWebhook`


# Usage

There are 3 types of clients, one using basic authentication mechanism, another
using OAuth, and a third for ISV partners. This is due to the implementation of the API
itself, meaning that different API calls are using different type of authenication, hence
this is unavoidable.

### ISV partner client

`ISVClient` serves Viva's [ISV Partner APIs](https://developer.viva.com/isv-partner-program/payment-isv-api/).
An ISV acts on behalf of many merchants, so the merchant id is an argument to each call
rather than client state — one client is safe to share across goroutines serving different
merchants:

```golang
isvClient := vivawallet.NewISV(resellerID, resellerAPIKey, clientID, clientSecret, true)

// Bearer-token endpoints (/checkout/v2/isv/*) use the ISV's OAuth credentials. The token
// is fetched and refreshed automatically; Authenticate is only needed to fail fast.

// Basic-auth endpoints (/api/*) carry the merchant the call is made for.
var order GetOrderPaymentResponse
err := isvClient.GetWithMerchant(merchantID, uri, &order)
```

The basic-auth credential is composed as Viva's ISV API spec documents it: username
`resellerID:merchantID`, password `resellerAPIKey`.

#### Taking a payment as an ISV

Create the order, redirect the customer to Smart Checkout, then find out how it went:

```golang
order, err := isvClient.CreateISVOrder(merchantID, vivawallet.ISVCheckoutOrder{
    Amount:               1000, // minor units: €10.00
    ISVAmount:            50,   // your commission; must not exceed Amount
    CustomerTransactions: "Coffee",
    Customer:             &vivawallet.Customer{Email: "customer@example.com", RequestLang: "el-GR"},
    CurrencyCode:         vivawallet.CurrencyEUR,
    PaymentTimeout:       600, // seconds
    DisableCash:          true,
})
if err != nil {
    return err
}

http.Redirect(w, r, vivawallet.SmartCheckoutURL(isvClient.Config, order.OrderCode,
    vivawallet.WithColor("#d5791a")), http.StatusFound)
```

`CreateISVOrder` returns an error without calling the API when `ISVAmount` exceeds `Amount`.
`WithColor` takes the hex value with or without a leading `#`.

`GetISVOrder` fetches the order once. The SDK does not poll for you; Viva recommends checking
every 3 seconds until the order leaves `OrderStatePending`, giving up after the order's
`PaymentTimeout` (10 minutes above). Own that loop so you can cancel it — and prefer
webhooks where you can:

```golang
ticker := time.NewTicker(3 * time.Second)
defer ticker.Stop()
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-ticker.C:
        op, err := isvClient.GetISVOrder(merchantID, order.OrderCode)
        if err != nil {
            return err
        }
        if op.StateID != vivawallet.OrderStatePending {
            return handle(op.StateID) // OrderStatePaid, OrderStateExpired, ...
        }
    }
}
```

#### Taking a payment on a card terminal

The EFT POS API is asynchronous: `Sale` returns once Viva accepts the request, and the
outcome arrives only when the customer finishes at the terminal. It needs a token with its
own OAuth scope, so use a separate `ISVClient` created with that scope (see below):

```golang
sessionID, err := vivawallet.NewSessionID() // or any unique id of your own
if err != nil {
    return err
}

err = ecrClient.Sale(vivawallet.ECRSaleRequest{
    SessionID:         sessionID,
    TerminalID:        "16013000",
    CashRegisterID:    "desk-1",
    Amount:            734, // €7.34, excluding the tip
    TipAmount:         100, // charged on top of Amount — do not add it to Amount too
    CurrencyCode:      vivawallet.ECRCurrencyEUR, // "978": a string here, unlike orders
    MerchantReference: bookingNumber,
    CustomerTrns:      "Room 12",
    ShowReceipt:       true,
})
```

Then poll `GetSession(sessionID)` — Viva recommends every 3 seconds, for up to 3 minutes — in
a loop you own and can cancel, as with `GetISVOrder` above.

Do not treat `Success == false` as proof the customer was not charged: Viva has been reported
returning it for sessions that settled
([VivaPayments/API#2461](https://github.com/VivaPayments/API/issues/2461)). Before refunding
or re-charging, confirm through the payment webhooks, matching on `TransactionID` or
`OrderCode`.

Viva's ISV and EFT POS flows use scoped OAuth tokens. Pass the scope at construction. For
ISV orders and ISV webhooks it is, per Viva's ISV API spec:

```golang
isvClient := vivawallet.NewISV(resellerID, resellerAPIKey, clientID, clientSecret, true,
    vivawallet.WithScope("urn:viva:payments:core:api:isv urn:viva:payments:core:api:redirectcheckout"))
```

The EFT POS API needs a different scope, so give it its own `ISVClient`.

With no `WithScope`, the token request carries no `scope` parameter, which is what the plain
payments flow expects.

### Webhooks

Viva notifies your endpoint of payments with a `POST`, and verifies you own the endpoint
with a `GET` on the *same URL*, which must return the key from `GetWebhookVerificationKey`
(`BasicAuthClient` for merchant webhooks, `ISVClient` for ISV ones). Viva will not activate
a webhook until that `GET` works.

`ParseWebhook` decodes the envelope only, so event types the SDK does not know still parse.
Then use the accessor for the event:

```golang
event, err := vivawallet.ParseWebhook(body)
if err != nil {
    return err // malformed: answer 400
}
switch event.EventTypeID {
case vivawallet.EventTransactionPaymentCreated:
    data, err := event.TransactionPaymentCreated()
    // data.OrderCode, data.TransactionID, data.StatusID, data.Amount (decimal euros) ...
}
```

Viva can deliver an event more than once: deduplicate on `event.MessageID`. Any non-2xx
answer is retried hourly for a day, so acknowledge event types you do not handle with 200.

#### Gin

Gin applications can use the `ginviva` module, which serves both the verification `GET`
(fetching the key once and caching it) and event delivery on one route. It is a separate
module so the SDK itself stays dependency-free; it needs go 1.20+.

```golang
import "github.com/techpals-eu/viva-wallet-go/ginviva"

h := ginviva.NewHandler(isvClient, ginviva.Config{
    OnPaymentCreated: func(c *gin.Context, d *vivawallet.TransactionEventData) error {
        // Return only once the event is stored; an error answers 500 and Viva redelivers.
        return payments.MarkPaid(c, d.OrderCode, d.TransactionID)
    },
})
r.Any("/webhooks/viva", h.Handle)
```

`ginviva` is versioned on its own, with tags of the form `ginviva/v1.2.0`.

### Typed transaction status and order state

The status fields decode into named types instead of bare `string` / `int`, so you can compare
against constants rather than magic values:

```golang
trx, err := oauthClient.GetTransaction(transactionID)
if err != nil {
    return err
}
if trx.StatusID == vivawallet.TransactionStatusFinished {
    // payment successful
}

order, err := basicAuthClient.GetOrderPayment(orderCode)
if err != nil {
    return err
}
if order.StateID == vivawallet.OrderStatePaid {
    // order has been paid
}
```

`GetTransactionResponse.StatusID` and `TransactionResponse.StatusID` are now
`TransactionStatus` (underlying type `string`); `GetOrderPaymentResponse.StateID` is now
`OrderState` (underlying type `int`). The JSON encoding is unchanged, and comparisons against
untyped string/int literals (`trx.StatusID == "F"`) still compile. Code that assigns these
fields to an explicitly-typed `string` or `int` variable, or passes them to a function taking
one, needs a conversion: `string(trx.StatusID)`.

Note that `TransactionStatus` is a string, not a single character — the dispute statuses
(`MA`, `MI`, `ML`, `MW`, `MS`) are two characters wide.

### OAuthClient now takes pointer receivers (1.1.0)

As of **1.1.0**, `OAuthClient` methods moved from value receivers to pointer receivers. This is
technically a breaking change, released under a minor version because no realistic caller is
affected — the patterns it breaks are ones the documented API never asked you to write. If you
pin versions, read this before upgrading from 1.0.x.

`NewOAuth` returns `*OAuthClient`, so if you hold on to what the constructor gives you — as every
example below does — nothing changes:

```golang
// Before and after: unchanged.
oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
token, err := oauthClient.Authenticate()
```

What no longer compiles is calling a method on a non-addressable `OAuthClient` *value*:

```golang
// Before: compiled. Now: does not compile.
vivawallet.OAuthClient{Config: cfg}.Authenticate()   // method on a composite literal
clients["eu"].Authenticate()                          // OAuthClient value stored in a map
var c vivawallet.Client = *vivawallet.NewOAuth(...)   // value assigned to the interface

// After: take the pointer.
c := vivawallet.NewOAuth(clientID, clientSecret, true)
c.Authenticate()
var _ vivawallet.Client = c
```

Dereferencing into a local variable (`v := *vivawallet.NewOAuth(...)`) still compiles, since Go
takes the address of an addressable value automatically — but it no longer does what it used to.
Previously a copied client *shared* its token with the original, so authenticating one
authenticated both; each copy now carries its own token. `go vet` reports these copies
(`assignment copies lock value ... contains sync.RWMutex`), so pass the `*OAuthClient` around
instead of copying the client.

## Installation

Under your project directory run the following:

```
go get -u github.com/techpals-eu/viva-wallet-go
```

## Payments

### Create order payment

```golang
oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
token, err := oauthClient.Authenticate()

req := vivawallet.CheckoutOrder{
		Amount: 1000,
}
op, err2 := oauthClient.CreateOrderPayment(req)
```

## Transactions

### Get a transaction

```golang
oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
token, err := oauthClient.Authenticate()

trx, err2 := oauthClient.GetTransaction("some-transaction-id")
```

For more examples check out: [main.go](./example/main.go)

---

<p align="center">
  <i>👩‍💻 Built by <a href="https://techpals.eu/">TechPals</a> 👨‍💻</i>
</p>
