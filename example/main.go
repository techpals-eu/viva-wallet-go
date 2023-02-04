package main

import (
	"fmt"

	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

func main() {
	clientID := "yjp82d6eub7hva6y9usesqtuzd8ambj914odu50n49jz3.apps.vivapayments.com"
	clientSecret := "ODX4vwQVmeYo373814yYf2p6Vq85yR"
	merchantID := "393969b6-c18e-4770-ba9a-2838c2beafee"
	apiKey := "YZ}z>_"
	client := vivawallet.New(clientID, clientSecret, merchantID, apiKey, true)

	token, err := client.Authenticate()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fmt.Printf("Token: %s\n\n", token.AccessToken)

	req := vivawallet.CheckoutOrder{
		Amount: 1000,
	}
	op, err2 := client.CreateOrderPayment(req)
	if err2 != nil {
		fmt.Printf("err: %s\n", err2.Error())
	} else {
		fmt.Printf("OrderPayment: %d\n", op.OrderCode)
	}

	wallets, err3 := client.GetWallets()
	if err3 != nil {
		fmt.Printf("err: %s\n", err3.Error())
	} else {
		for _, w := range wallets {
			fmt.Printf("Wallet: %v\n", w)
		}
	}

}
