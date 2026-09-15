package vivawallet

import "testing"

func TestApiUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo-api.vivapayments.com"},
		{name: "production", demo: false, want: "https://api.vivapayments.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApiUri(Config{Demo: tt.demo})
			if got != tt.want {
				t.Errorf("ApiUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com"},
		{name: "production", demo: false, want: "https://www.vivapayments.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppUri(Config{Demo: tt.demo})
			if got != tt.want {
				t.Errorf("AppUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOAuthClientAuthUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo-accounts.vivapayments.com"},
		{name: "production", demo: false, want: "https://accounts.vivapayments.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OAuthClient{Config: Config{Demo: tt.demo}}
			got := c.authUri()
			if got != tt.want {
				t.Errorf("authUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTokenEndpoint(t *testing.T) {
	c := &OAuthClient{Config: Config{Demo: true}}
	want := "https://demo-accounts.vivapayments.com/connect/token"
	if got := c.tokenEndpoint(); got != want {
		t.Errorf("tokenEndpoint() = %q, want %q", got, want)
	}
}

func TestGetTransactionUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo-api.vivapayments.com/checkout/v2/transactions/trx-1"},
		{name: "production", demo: false, want: "https://api.vivapayments.com/checkout/v2/transactions/trx-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTransactionUri(Config{Demo: tt.demo}, "trx-1")
			if got != tt.want {
				t.Errorf("getTransactionUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCreateTransactionUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/transactions/trx-1"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/transactions/trx-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCreateTransactionUri(Config{Demo: tt.demo}, "trx-1")
			if got != tt.want {
				t.Errorf("getCreateTransactionUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCancelTransactionUri(t *testing.T) {
	tests := []struct {
		name       string
		demo       bool
		sourceCode string
		want       string
	}{
		{
			name: "demo without source code",
			demo: true,
			want: "https://demo.vivapayments.com/api/transactions/trx-1?amount=100",
		},
		{
			name:       "production with source code",
			demo:       false,
			sourceCode: "src 1",
			want:       "https://www.vivapayments.com/api/transactions/trx-1?amount=100&sourceCode=src+1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCancelTransactionUri(Config{Demo: tt.demo}, "trx-1", 100, tt.sourceCode)
			if got != tt.want {
				t.Errorf("getCancelTransactionUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCancelPartialAuthUri(t *testing.T) {
	tests := []struct {
		name       string
		demo       bool
		sourceCode string
		want       string
	}{
		{
			name: "demo without source code",
			demo: true,
			want: "https://demo-api.vivapayments.com/acquiring/v1/transactions/trx-1?amount=100",
		},
		{
			name:       "production with source code",
			demo:       false,
			sourceCode: "src 1",
			want:       "https://api.vivapayments.com/acquiring/v1/transactions/trx-1?amount=100&sourceCode=src+1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCancelPartialAuthUri(Config{Demo: tt.demo}, "trx-1", 100, tt.sourceCode)
			if got != tt.want {
				t.Errorf("getCancelPartialAuthUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckoutOrderUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo-api.vivapayments.com/checkout/v2/orders"},
		{name: "production", demo: false, want: "https://api.vivapayments.com/checkout/v2/orders"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkoutOrderUri(Config{Demo: tt.demo})
			if got != tt.want {
				t.Errorf("checkoutOrderUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateOrderUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/orders/123"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/orders/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateOrderUri(Config{Demo: tt.demo}, 123)
			if got != tt.want {
				t.Errorf("updateOrderUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetOrderPaymentUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/orders/123"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/orders/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOrderPaymentUri(Config{Demo: tt.demo}, 123)
			if got != tt.want {
				t.Errorf("getOrderPaymentUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeleteOrderPaymentUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/orders/123"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/orders/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deleteOrderPaymentUri(Config{Demo: tt.demo}, 123)
			if got != tt.want {
				t.Errorf("deleteOrderPaymentUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBalanceTransferUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/wallets/w1/balancetransfer/w2"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/wallets/w1/balancetransfer/w2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBalanceTransferUri(Config{Demo: tt.demo}, "w1", "w2")
			if got != tt.want {
				t.Errorf("getBalanceTransferUri() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetWalletsUri(t *testing.T) {
	tests := []struct {
		name string
		demo bool
		want string
	}{
		{name: "demo", demo: true, want: "https://demo.vivapayments.com/api/wallets"},
		{name: "production", demo: false, want: "https://www.vivapayments.com/api/wallets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getWalletsUri(Config{Demo: tt.demo})
			if got != tt.want {
				t.Errorf("getWalletsUri() = %q, want %q", got, tt.want)
			}
		})
	}
}
