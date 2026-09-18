package vivawallet

import (
	"encoding/json"
	"os"
	"testing"
)

// The three status/state fields are decoded from fixtures modelled on Viva's own documented
// response examples, so that a wrong Go type is caught here rather than in production.
//
// encoding/json errors on a type mismatch, so these tests fail loudly if StatusId ever stops
// being a JSON string or StateId stops being a JSON number.

func TestGetTransactionResponseDecodesStatusID(t *testing.T) {
	fixture, err := os.ReadFile("testdata/get_transaction_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var resp GetTransactionResponse
	if err := json.Unmarshal(fixture, &resp); err != nil {
		t.Fatalf("decoding get_transaction_response.json: %v", err)
	}
	if resp.StatusID != TransactionStatusFinished {
		t.Errorf("StatusID = %q, want %q", resp.StatusID, TransactionStatusFinished)
	}
}

func TestTransactionResponseDecodesStatusID(t *testing.T) {
	fixture, err := os.ReadFile("testdata/create_transaction_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var resp TransactionResponse
	if err := json.Unmarshal(fixture, &resp); err != nil {
		t.Fatalf("decoding create_transaction_response.json: %v", err)
	}
	if resp.StatusID != TransactionStatusFinished {
		t.Errorf("StatusID = %q, want %q", resp.StatusID, TransactionStatusFinished)
	}
	if resp.CurrencyCode != "978" {
		t.Errorf("CurrencyCode = %q, want 978", resp.CurrencyCode)
	}
	if !resp.Success {
		t.Error("Success = false, want true")
	}
}

func TestGetOrderPaymentResponseDecodesStateID(t *testing.T) {
	fixture, err := os.ReadFile("testdata/get_order_payment_response.json")
	if err != nil {
		t.Fatal(err)
	}

	var resp GetOrderPaymentResponse
	if err := json.Unmarshal(fixture, &resp); err != nil {
		t.Fatalf("decoding get_order_payment_response.json: %v", err)
	}
	if resp.StateID != OrderStatePaid {
		t.Errorf("StateID = %d, want %d (Paid)", resp.StateID, OrderStatePaid)
	}
}

// TestTransactionStatusDecodesTwoCharacterDisputeCodes guards the reason TransactionStatus is a
// string type and not a single byte or rune: the dispute statuses are two characters wide.
func TestTransactionStatusDecodesTwoCharacterDisputeCodes(t *testing.T) {
	for _, want := range []TransactionStatus{
		TransactionStatusDisputeAwaitingResponse,
		TransactionStatusDisputeInProgress,
		TransactionStatusDisputeLost,
		TransactionStatusDisputeWon,
		TransactionStatusDisputeSuspected,
	} {
		body := []byte(`{"statusId":"` + string(want) + `"}`)

		var resp GetTransactionResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("decoding statusId %q: %v", want, err)
		}
		if resp.StatusID != want {
			t.Errorf("StatusID = %q, want %q", resp.StatusID, want)
		}
	}
}

// TestOrderStateRejectsQuotedStateID documents that StateId is a JSON number. Viva's OpenAPI
// document annotates it as `type: string`, but its own response example returns an unquoted 3.
// If Viva ever really starts quoting it, this test fails and flags the breaking change.
func TestOrderStateRejectsQuotedStateID(t *testing.T) {
	var resp GetOrderPaymentResponse
	if err := json.Unmarshal([]byte(`{"StateId":"3"}`), &resp); err == nil {
		t.Fatal("decoding a quoted StateId succeeded, want a type error")
	}
}
