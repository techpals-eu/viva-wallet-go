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


# Usage

There are 2 types of clients, one using basic authentication mechanism and another
using OAuth. This is due to the implementation of the API itself, meaning that
different API calls are using different type of authenication, hence this is unavoidable.

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

### Create card token

```golang
oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
token, err := oauthClient.Authenticate()

createCardToken := CreateCardToken{
		TransactionID: "some-trx-id",
}
cardToken, err2 := CreateCardToken(cardToken)
```

For more examples check out: [main.go](./example/main.go)

---

<p align="center">
  <i>👩‍💻 Built by <a href="https://techpals.eu/">TechPals</a> 👨‍💻</i>
</p>
