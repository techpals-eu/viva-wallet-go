// Command fiscal runs the Greek fiscalisation flow against Viva's DEMO environment, one step
// at a time (step 8 of plans/25-greek-fiscalisation.md). It never talks to production.
//
// Credentials come from the environment, never from flags or files:
//
//	VIVA_RESELLER_ID, VIVA_RESELLER_API_KEY   ISV reseller credentials
//	VIVA_ISV_CLIENT_ID, VIVA_ISV_CLIENT_SECRET ISV OAuth client
//	VIVA_MERCHANT_ID                           the merchant to act for
//	VIVA_MERCHANT_VAT_ID                       the merchant's ΑΦΜ (cash and terminal steps)
//	VIVA_TERMINAL_ID                           the terminal's TID (terminal step only)
//
// Steps, in the order to run them:
//
//	go run ./example/fiscal -step start                  # once per merchant: Queue-Start receipt
//	go run ./example/fiscal -step zero                   # health check: is the queue ready?
//	go run ./example/fiscal -step cash -aa 1             # 11.1 cash receipt via receipts:sign
//	go run ./example/fiscal -step terminal -aa 2         # card sale + receipt on the terminal
//
// AA must increase per series on every run: Viva rejects or misfiles a reused number. API
// errors are printed with their full body, which is what plan 25 still needs to learn.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/techpals-eu/viva-wallet-go"
)

const demo = true // this program must never run against production

const (
	isvScope = "urn:viva:payments:core:api:isv urn:viva:payments:core:api:redirectcheckout"
	ecrScope = "urn:viva:payments:ecr:api"
)

func main() {
	step := flag.String("step", "zero", "start | zero | cash | terminal")
	series := flag.String("series", "DEMO", "receipt series")
	aa := flag.Int64("aa", 0, "receipt number within the series (cash, terminal); must increase per run")
	amount := flag.Int64("amount", 124, "receipt total in cents, 24% VAT included")
	flag.Parse()

	if err := run(*step, *series, *aa, *amount); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printAPIError(err)
		os.Exit(1)
	}
}

func run(step, series string, aa, amount int64) error {
	env, err := readEnv(step)
	if err != nil {
		return err
	}
	isv := vivawallet.NewISV(env["VIVA_RESELLER_ID"], env["VIVA_RESELLER_API_KEY"],
		env["VIVA_ISV_CLIENT_ID"], env["VIVA_ISV_CLIENT_SECRET"], demo, vivawallet.WithScope(isvScope))
	merchantID := env["VIVA_MERCHANT_ID"]

	switch step {
	case "start":
		res, err := isv.StartFiscalQueue(merchantID)
		return report(isv.Config, res, "", err)

	case "zero":
		res, err := isv.ZeroReceipt(merchantID)
		return report(isv.Config, res, "", err)

	case "cash":
		receipt, err := demoReceipt(env["VIVA_MERCHANT_VAT_ID"], series, aa, amount, vivawallet.GRPaymentCash).Sale()
		if err != nil {
			return err
		}
		sessionID, err := vivawallet.NewFiscalSessionID()
		if err != nil {
			return err
		}
		fmt.Println("sessionId (rerun with the same id and receipt to retry safely):", sessionID)
		printJSON("request", receipt)
		res, err := isv.SignReceipt(merchantID, sessionID, receipt)
		return report(isv.Config, res, receipt.FTReceiptCaseData.GR.HashPayload, err)

	case "terminal":
		ecr := vivawallet.NewISV(env["VIVA_RESELLER_ID"], env["VIVA_RESELLER_API_KEY"],
			env["VIVA_ISV_CLIENT_ID"], env["VIVA_ISV_CLIENT_SECRET"], demo, vivawallet.WithScope(ecrScope))
		return terminalSale(ecr, env, series, aa, amount)
	}
	return fmt.Errorf("unknown step %q", step)
}

func readEnv(step string) (map[string]string, error) {
	need := []string{"VIVA_RESELLER_ID", "VIVA_RESELLER_API_KEY", "VIVA_ISV_CLIENT_ID", "VIVA_ISV_CLIENT_SECRET", "VIVA_MERCHANT_ID"}
	if step == "cash" || step == "terminal" {
		need = append(need, "VIVA_MERCHANT_VAT_ID")
	}
	if step == "terminal" {
		need = append(need, "VIVA_TERMINAL_ID")
	}
	env := map[string]string{}
	var missing []string
	for _, k := range need {
		if env[k] = os.Getenv(k); env[k] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("set %v", missing)
	}
	return env, nil
}

// demoReceipt is one line of goods at 24% VAT, paid in full with pp.
func demoReceipt(vatID, series string, aa, amount int64, pp vivawallet.GRPaymentType) vivawallet.GRReceipt {
	vat := amount - amount*100/124 // VAT included in a 24% gross amount
	return vivawallet.GRReceipt{
		MerchantVATID: vatID,
		Series:        series,
		AA:            aa,
		Reference:     fmt.Sprintf("demo-%s-%d", series, aa),
		Moment:        time.Now(),
		Lines: []vivawallet.ChargeItem{{
			Amount: amount, Description: "Demo item", Quantity: 100, VATRate: 2400, VATAmount: vat,
			FTChargeItemCase: vivawallet.GRChargeItemCase(vivawallet.GRVATNatureUsual,
				vivawallet.GRServiceGoods, vivawallet.GRVATNormal, 0),
		}},
		Payments: []vivawallet.PayItem{{
			Amount: amount, Description: "Payment", FTPayItemCase: vivawallet.GRPayItemCase(pp, 0),
		}},
	}
}

func terminalSale(ecr *vivawallet.ISVClient, env map[string]string, series string, aa, amount int64) error {
	receipt, err := demoReceipt(env["VIVA_MERCHANT_VAT_ID"], series, aa, amount, vivawallet.GRPaymentEFTPOS).Sale()
	if err != nil {
		return err
	}
	sessionID, err := vivawallet.NewSessionID()
	if err != nil {
		return err
	}
	sale := vivawallet.ECRSaleRequest{
		SessionID:             sessionID,
		TerminalID:            env["VIVA_TERMINAL_ID"],
		CashRegisterID:        "fiscal-demo",
		Amount:                amount,
		CurrencyCode:          vivawallet.ECRCurrencyEUR,
		MerchantReference:     receipt.CBReceiptReference,
		CustomerTrns:          "Fiscal demo",
		ShowTransactionResult: true,
		ShowReceipt:           true,
		ISVDetails:            &vivawallet.ECRISVDetails{TerminalMerchantID: env["VIVA_MERCHANT_ID"]},
		FiscalisationData:     &receipt,
	}
	printJSON("sale request", sale)
	if err := ecr.Sale(sale); err != nil {
		return err
	}
	fmt.Println("sent; pay on the terminal. polling every 3s for up to 3 minutes (Ctrl-C stops)")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("gave up waiting for session %s: %w", sessionID, ctx.Err())
		case <-tick.C:
		}
		session, err := ecr.GetSession(sessionID)
		if err != nil {
			fmt.Println("  poll:", err) // e.g. not found while the terminal has not answered yet
			continue
		}
		if session.TransactionID == "" && session.Message == "" && !session.Success {
			fmt.Println("  poll: waiting")
			continue
		}
		printJSON("session", session)
		if session.FiscalisationDetails == nil {
			fmt.Println("no fiscalisationDetails: is fiscalisation enabled for this merchant and terminal?")
			return nil
		}
		return report(ecr.Config, session.FiscalisationDetails, receipt.FTReceiptCaseData.GR.HashPayload, nil)
	}
}

// report prints a fiscal result the way a receipt would show it.
func report(c vivawallet.Config, res *vivawallet.FiscalResult, hashPayload string, err error) error {
	if err != nil {
		return err
	}
	printJSON("result", res)
	fmt.Printf("state %s  ready=%v  error=%v  fail=%v  daily-closing-due=%v\n",
		res.State, res.State.Ready(), res.State.Error(), res.State.Fail(), res.State.DailyClosingDue())
	if note := res.Transmission().String(); note != "" {
		fmt.Println("PRINT:", note)
	}
	if mark := res.Mark(); mark != "" {
		fmt.Println("MARK:", mark)
	}
	for _, s := range res.Signatures {
		fmt.Printf("  [%d] %s\n      %s\n", s.Format, s.Caption, s.Data)
	}
	if hashPayload != "" {
		fmt.Println("HashPayload:", hashPayload)
		fmt.Println("QR:", vivawallet.GRReceiptQRURL(c, hashPayload))
	}
	return nil
}

// printAPIError prints everything Viva sent back, so the shape of fiscal errors can be
// recorded in plan 25.
func printAPIError(err error) {
	var apiErr *vivawallet.APIError
	if !errors.As(err, &apiErr) {
		return
	}
	fmt.Fprintf(os.Stderr, "HTTP %d, body:\n%s\n", apiErr.StatusCode, apiErr.Body)
	fmt.Fprintf(os.Stderr, "IsFiscalQueueNotStarted=%v IsFiscalisationNotEnabled=%v\n",
		vivawallet.IsFiscalQueueNotStarted(err), vivawallet.IsFiscalisationNotEnabled(err))
}

func printJSON(label string, v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Printf("--- %s\n%s\n", label, b)
}
