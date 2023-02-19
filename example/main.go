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

	fmt.Printf("\nCreate order\n")
	req := vivawallet.CheckoutOrder{
		Amount: 1000,
		PreAuth: true,
	}
	op, err2 := oauthClient.CreateOrderPayment(req)
	if err2 != nil {
		fmt.Printf("\nerr: %s\n", err2.Error())
	} else {
		fmt.Printf("\nOrderPayment: %d\n", op.OrderCode)
	}

	fmt.Printf("\nGet wallets\n")
	wallets, err3 := basicAuthClient.GetWallets()
	if err3 != nil {
		fmt.Printf("\nerr: %s\n", err3.Error())
	} else {
		for _, w := range wallets {
			fmt.Printf("\nWallet: %v\n", w)
		}
	}

	fmt.Printf("\nGet transaction\n")
	trxID := "a9531058-f0f7-44ff-a718-98920804ceab"
	trx, err4 := oauthClient.GetTransaction(trxID)
	if err4 != nil {
		fmt.Printf("\nerr: %s\n", err4.Error())
	} else {
		fmt.Printf("\nTrx: %v\n", trx)
	}

	fmt.Printf("\nUpdate orderpayment\n")
	update := vivawallet.UpdateOrderPayment{
		Amount: 1200,
	}
	err6 := basicAuthClient.UpdateOrderPayment(op.OrderCode, update)
	if err6 != nil {
		fmt.Printf("\nerr: %s\n", err6.Error())
	} else {
		fmt.Println("\nsuccess")
	}

	fmt.Printf("\nGet orderpayment\n")
	opGet, err7 := basicAuthClient.GetOrderPayment(op.OrderCode)
	if err7 != nil {
		fmt.Printf("\nerr: %s\n", err7.Error())
	} else {
		fmt.Printf("%v", opGet)
		fmt.Println("\nsuccess")
	}

	fmt.Printf("\nCancel orderpayment\n")
	opCancel, err8 := basicAuthClient.CancelOrderPayment(op.OrderCode)
	if err8 != nil {
		fmt.Printf("\nerr: %s\n", err8.Error())
	} else {
		fmt.Printf("%v", opCancel)
		fmt.Println("\nsuccess")
	}

	trx2, err9 := basicAuthClient.CancelTransaction("aacf07cf-9102-4b02-8172-72b7e1efd5d9", 100, "Default")
	if err9 != nil {
		fmt.Printf("\nerr: %s\n", err9.Error())
	} else {
		fmt.Printf("%v", trx2)
		fmt.Println("\nsuccess")
	}

	payload := vivawallet.CreateTransaction{
		Amount: 100,
	}
	trx3, err2 := basicAuthClient.CreateTransaction("cdc8e764-daf3-49de-9f44-c7f3b563c2d6", payload)
	fmt.Printf("%v\nERR: %v\n", trx3, err2)
}
