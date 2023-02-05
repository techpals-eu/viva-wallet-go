package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func getBalanceTransferUri(c Config, walletID string, targetWalletID string) string {
	return fmt.Sprintf("%s/api/wallets/%s/balancetransfer/%s", AppUri(c), walletID, targetWalletID)
}

type BalanceTransfer struct {
	Amount            int    `json:"amount"`
	Description       string `json:"description"`
	SaleTransactionID string `json:"saleTransactionId"`
}

type BalanceTransferResponse struct {
	DebitTransactionID  string `json:"DebitTransactionId"`
	CreditTransactionID string `json:"CreditTransactionId"`
}

// BalanceTransfer transfers money from one wallet to another.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Balance-Transfer/paths/~1api~1wallets~1{walletId}~1balancetransfer~1{targetWalletId}/post
func (c BasicAuthClient) BalanceTranfer(walletID string, targetWalletID string, payload BalanceTransfer) (*BalanceTransferResponse, error) {

	uri := getBalanceTransferUri(c.Config, walletID, targetWalletID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BalanceTransfer %s", err)
	}

	b := &BalanceTransferResponse{}
	reqErr := c.Post(uri, bytes.NewReader(data), &b)
	if reqErr != nil {
		return nil, reqErr
	}
	return b, nil
}

func getWalletsUri(c Config) string {
	return fmt.Sprintf("%s/api/wallets", AppUri(c))
}

type Wallet struct {
	IBAN         string  `json:"Iban"`
	WalletID     int     `json:"WalletId"`
	IsPrimary    bool    `json:"IsPrimary"`
	Amount       float64 `json:"Amount"`
	Available    float64 `json:"Available"`
	Overdraft    float64 `json:"Overdraft"`
	FriendlyName string  `json:"FriendlyName"`
	CurrencyCode string  `json:"CurrencyCode"`
}

// GetWallets fetches a list of wallets associated to your account.
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/#tag/Retrieve-Wallet/paths/~1api~1wallets/get
func (c BasicAuthClient) GetWallets() ([]Wallet, error) {
	uri := getWalletsUri(c.Config)

	var r []Wallet
	err := c.Get(uri, &r)
	if err != nil {
			return nil, err
	}
	return r, nil
}
