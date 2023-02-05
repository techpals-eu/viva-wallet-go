package main

import (
	"fmt"

	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

func main() {
	clientID := ""
	clientSecret := ""
	merchantID := ""
	apiKey := ""
	oauthClient := vivawallet.NewOAuth(clientID, clientSecret, true)
	basicAuthClient := vivawallet.NewBasicAuth(merchantID, apiKey, true)

	token, err := oauthClient.Authenticate()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fmt.Printf("Token: %s\n\n", token.AccessToken)

	req := vivawallet.CheckoutOrder{
		Amount: 1000,
	}
	op, err2 := oauthClient.CreateOrderPayment(req)
	if err2 != nil {
		fmt.Printf("err: %s\n", err2.Error())
	} else {
		fmt.Printf("OrderPayment: %d\n", op.OrderCode)
	}

	wallets, err3 := basicAuthClient.GetWallets()
	if err3 != nil {
		fmt.Printf("err: %s\n", err3.Error())
	} else {
		for _, w := range wallets {
			fmt.Printf("Wallet: %v\n", w)
		}
	}

	trx, err4 := oauthClient.GetTransaction("a9531058-f0f7-44ff-a718-98920804ceab")
	if err4 != nil {
		fmt.Printf("err: %s\n", err4.Error())
	} else {
		fmt.Printf("Trx: %v\n", trx)
	}
}
