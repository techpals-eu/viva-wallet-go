# Viva Wallet Go SDK

viva-wallet is an SDK for connecting to the payment provider Viva Wallet.

## Supported Endpoints

- [Authentication (OAuth)](https://developer.vivawallet.com/apis-for-payments/payment-api/#section/Authentication)
- [Payments](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Payments)
- [Transactions](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions)
  - [Retrieve transaction](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions/paths/~1checkout~1v2~1transactions~1{transactionId}/get)
  - [Created card token](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Transactions/paths/~1acquiring~1v1~1cards~1tokens/post)
- [Wallet](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Balance-Transfer)
  - [Balance Tranfer](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Balance-Transfer)
  - [Retrieve Wallets](https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Retrieve-Wallet)


# Usage

There are 2 types of clients, one using basic authentication mechanism and another
using OAuth. This is due to the implementation of the API itself, meaning that
different API calls are using different type of authenication, hence this is unavoidable.

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
