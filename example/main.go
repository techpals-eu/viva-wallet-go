package main

import (
	"fmt"
	"os"

	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

func main() {
	clientID := os.Getenv("VIVA_CLIENT_ID")
	clientSecret := os.Getenv("VIVA_CLIENT_SECRET")
	merchantID := os.Getenv("VIVA_MERCHANT_ID")
	apiKey := os.Getenv("VIVA_API_KEY")

	oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
	basicAuthClient := vivawallet.NewBasicAuth(merchantID, apiKey, true)

	token, err := oauthClient.Authenticate()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fmt.Printf("Token: %s\n\n", token.AccessToken)

	fmt.Println("\nCreate order\n")
	req := vivawallet.CheckoutOrder{
		Amount: 1000,
	}
	op, err2 := oauthClient.CreateOrderPayment(req)
	if err2 != nil {
		fmt.Printf("err: %s\n", err2.Error())
	} else {
		fmt.Printf("OrderPayment: %d\n", op.OrderCode)
	}

	fmt.Println("\nGet wallets")
	wallets, err3 := basicAuthClient.GetWallets()
	if err3 != nil {
		fmt.Printf("err: %s\n", err3.Error())
	} else {
		for _, w := range wallets {
			fmt.Printf("Wallet: %v\n", w)
		}
	}

	fmt.Println("\nGet transaction")
	trx, err4 := oauthClient.GetTransaction("a9531058-f0f7-44ff-a718-98920804ceab")
	if err4 != nil {
		fmt.Printf("err: %s\n", err4.Error())
	} else {
		fmt.Printf("Trx: %v\n", trx)
	}

	fmt.Println("\nCreate card token")
	createCardToken := vivawallet.CreateCardToken{
		TransactionID: "a9531058-f0f7-44ff-a718-98920804ceab",
	}
	cardToken, err5 := oauthClient.CreateCardToken(createCardToken)
	if err5 != nil {
		fmt.Printf("err: %s\n", err5.Error())
	} else {
		fmt.Printf("Card token: %v\n", cardToken)
	}

	fmt.Println("\nUpdate orderpayment")
	update := vivawallet.UpdateOrderPayment{
		Amount: 1200,
	}
	err6 := basicAuthClient.UpdateOrderPayment(op.OrderCode, update)
	if err6 != nil {
		fmt.Printf("err: %s\n", err6.Error())
	} else {
		fmt.Printf("success\n")
	}
}
