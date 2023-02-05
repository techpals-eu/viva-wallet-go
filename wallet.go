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

func (c BasicAuthClient) BalanceTranfer(walletID string, targetWalletID string, payload BalanceTransfer) (*BalanceTransferResponse, error) {

	uri := getBalanceTransferUri(c.Config, walletID, targetWalletID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BalanceTransfer %s", err)
	}

	body, bodyErr := c.post(uri, bytes.NewReader(data))
	if bodyErr != nil {
		return nil, bodyErr
	}

	b := &BalanceTransferResponse{}
	if jsonErr := json.Unmarshal(body, b); jsonErr != nil {
		return nil, jsonErr
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

func (c BasicAuthClient) GetWallets() ([]Wallet, error) {
	uri := getWalletsUri(c.Config)

	body, bodyErr := c.get(uri)
	if bodyErr != nil {
		return nil, bodyErr
	}

	var r []Wallet
	if jsonErr := json.Unmarshal(body, &r); jsonErr != nil {
		return nil, jsonErr
	}
	return r, nil
}
