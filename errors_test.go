package vivawallet

import "testing"

func TestIsSuccess(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{status: 200, want: true},
		{status: 201, want: true},
		{status: 204, want: true},
		{status: 299, want: true},
		{status: 199, want: false},
		{status: 300, want: false},
		{status: 400, want: false},
		{status: 500, want: false},
	}

	for _, tt := range tests {
		if got := isSuccess(tt.status); got != tt.want {
			t.Errorf("isSuccess(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestNewAPIError(t *testing.T) {
	body := []byte(`{"ErrorCode":1013,"ErrorText":"Invalid TransactionId","EventId":123,"CorrelationId":"abc"}`)

	err := newAPIError(400, body)

	if err.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", err.StatusCode)
	}
	if err.ErrorCode != 1013 {
		t.Errorf("ErrorCode = %d, want 1013", err.ErrorCode)
	}
	if err.ErrorText != "Invalid TransactionId" {
		t.Errorf("ErrorText = %q, want Invalid TransactionId", err.ErrorText)
	}
	if err.CorrelationID != "abc" {
		t.Errorf("CorrelationID = %q, want abc", err.CorrelationID)
	}

	want := "viva: status 400: Invalid TransactionId (code 1013, correlation abc)"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestNewAPIErrorNonJSONBody(t *testing.T) {
	err := newAPIError(502, []byte("<html>Bad Gateway</html>"))

	if err.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", err.StatusCode)
	}
	if err.ErrorText != "" {
		t.Errorf("ErrorText = %q, want empty for a non-JSON body", err.ErrorText)
	}

	want := "viva: status 502: <html>Bad Gateway</html>"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestNewAPIErrorTruncatesOversizedBody(t *testing.T) {
	body := make([]byte, maxErrorBody+100)
	for i := range body {
		body[i] = 'a'
	}

	err := newAPIError(500, body)

	if len(err.Body) != maxErrorBody {
		t.Errorf("len(Body) = %d, want %d", len(err.Body), maxErrorBody)
	}
}
