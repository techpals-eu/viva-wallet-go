package vivawallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c Client) BalanceTranfer(walletID string, targetWalletID string, payload BalanceTransfer) (*BalanceTransferResponse, error) {
	auth := AuthBody(c.Config)

	uri := getBalanceTransferUri(c.Config, walletID, targetWalletID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BalanceTransfer %s", err)
	}

	req, _ := http.NewRequest("POST", uri, bytes.NewReader(data))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to tranfer money %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to tranfer money with status %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
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

type GetWalletResponse struct {
	IBAN         string  `json:"Iban"`
	WalletID     string  `json:"WalletId"`
	IsPrimary    bool    `json:"IsPrimary"`
	Amount       float64 `json:"Amount"`
	Available    float64 `json:"Available"`
	Overdraft    float64 `json:"Overdraft"`
	FriendlyName string  `json:"FriendlyName"`
	CurrencyCode int     `json:"CurrencyCode"`
}

func (c Client) RetrieveWallet() (*GetWalletResponse, error) {
	auth := AuthBody(c.Config)

	uri := getWalletsUri(c.Config)

	req, _ := http.NewRequest("GET", uri, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/json")

	resp, httpErr := c.HTTPClient.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("failed to get wallets %s", httpErr)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get wallets with status %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	r := &GetWalletResponse{}
	if jsonErr := json.Unmarshal(body, r); jsonErr != nil {
		return nil, jsonErr
	}
	return r, nil
}
