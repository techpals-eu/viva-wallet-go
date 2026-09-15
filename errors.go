package vivawallet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maxErrorBody caps how much of a non-2xx response body is retained on an APIError, so that
// an HTML error page does not end up wholesale in a log line.
const maxErrorBody = 64 << 10

// APIError is returned for any non-2xx response from Viva's API. StatusCode and Body are
// always populated; the remaining fields are filled in on a best-effort basis when the body
// is a Viva-shaped JSON error.
//
// Ref: https://developer.vivawallet.com/apis-for-payments/payment-api/
type APIError struct {
	StatusCode    int    `json:"-"`
	Body          string `json:"-"`
	ErrorCode     int    `json:"ErrorCode"`
	ErrorText     string `json:"ErrorText"`
	EventID       int    `json:"EventId"`
	CorrelationID string `json:"CorrelationId"`
}

func (e *APIError) Error() string {
	if e.ErrorText != "" {
		return fmt.Sprintf("viva: status %d: %s (code %d, correlation %s)",
			e.StatusCode, e.ErrorText, e.ErrorCode, e.CorrelationID)
	}
	return fmt.Sprintf("viva: status %d: %s", e.StatusCode, e.Body)
}

// newAPIError builds an APIError from a response status and its already-read body. The body
// is decoded on a best-effort basis: responses that are plain text or HTML simply leave the
// Viva-specific fields zeroed.
func newAPIError(statusCode int, body []byte) *APIError {
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody]
	}

	apiErr := &APIError{StatusCode: statusCode, Body: string(body)}
	_ = json.Unmarshal(body, apiErr)

	return apiErr
}

// isSuccess reports whether a status code is in the 2xx range. Viva legitimately returns 201
// and 204 from some endpoints, so a bare comparison against 200 is not enough.
func isSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}

// readBody reads a response body, capped so a malformed or oversized response cannot exhaust
// memory, and closes it.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxErrorBody+1))
}
