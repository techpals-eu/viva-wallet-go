package main

import (
	"fmt"

	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

func main() {
	clientID := ""
	clientSecret := ""
	client := vivawallet.New(clientID, clientSecret, true)

	token, _ := client.Authenticate()
	fmt.Printf("Token: %s\n", token.AccessToken)

	req := vivawallet.CheckoutOrderRequest{
		Amount: 1000,
	}
	op, _ := client.CreateOrderPayment(req)
	fmt.Printf("OrderPayment: %d\n", op.OrderCode)
}
