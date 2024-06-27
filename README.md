# Viva Wallet Go SDK

viva-wallet is an SDK for connecting to the payment provider Viva Wallet.

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

## Installation

Under your project directory run the following:

```
go get -u github.com/techpals-eu/viva-wallet
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
