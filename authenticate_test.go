package vivawallet

import (
	"testing"
	"time"
)

func Test_HasAuthExpired_does_not_reauthenticate(t *testing.T) {
	client := vivawallet.NewOAuthClient("fake-id", "fake-secret", true)

	// 1. mock http client calls (i.e. /connect/token and /checkout/v2/orders).
	mockAuthResponse := &TokenResponse{
		AccessToken: "",
		ExpiresIn:   3600,
		Scope:       "whatever",
		TokenType:   "Bearer",
	}

	// 2. verify AuthToken returns a token

	// 3. call CreateOrderPayment

	// 4. verify HasAuthExpired does not call Authenticate
}

func Test_HasAuthExpired_reauthenticate(t *testing.T) {
	client := vivawallet.NewOAuthClient("fake-id", "fake-secret", true)

	// 1. mock http client calls (i.e. /connect/token and /checkout/v2/orders).
	mockAuthResponse := &TokenResponse{
		AccessToken: "",
		ExpiresIn:   1, // 1 second
		Scope:       "whatever",
		TokenType:   "Bearer",
	}

	// 2. Wait for the token to expire
	time.Sleep(2)

	// 3. verify AuthToken returns a real token

	// 4. make a call to create order payment

	// 5. verify HasTokenExpired calls Authenticate
}
