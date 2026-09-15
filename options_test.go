package vivawallet

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// recordingTransport records that it was used and returns a canned empty response.
type recordingTransport struct {
	calls int
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.calls++
	return httptest.NewRecorder().Result(), nil
}

func TestWithHTTPClientIsUsed(t *testing.T) {
	rt := &recordingTransport{}
	hc := &http.Client{Transport: rt}

	oauth := NewOAuth("id", "secret", true, WithHTTPClient(hc))
	if oauth.Client != hc {
		t.Fatalf("NewOAuth did not use the supplied http.Client")
	}
	if err := oauth.Get("https://example.invalid/x", &struct{}{}); err != nil {
		t.Fatalf("Get returned %v", err)
	}

	basic := NewBasicAuth("merchant", "key", true, WithHTTPClient(hc))
	if basic.Client != hc {
		t.Fatalf("NewBasicAuth did not use the supplied http.Client")
	}
	if err := basic.Get("https://example.invalid/x", &struct{}{}); err != nil {
		t.Fatalf("Get returned %v", err)
	}

	if rt.calls != 2 {
		t.Fatalf("transport recorded %d calls, want 2", rt.calls)
	}
}

func TestClientsDoNotShareHTTPClient(t *testing.T) {
	a := NewOAuth("id", "secret", true)
	b := NewOAuth("id", "secret", true)
	if a.Client == b.Client {
		t.Fatal("two OAuthClients share the same *http.Client")
	}

	c := NewBasicAuth("merchant", "key", true)
	d := NewBasicAuth("merchant", "key", true)
	if c.Client == d.Client {
		t.Fatal("two BasicAuthClients share the same *http.Client")
	}
	if a.Client == c.Client {
		t.Fatal("an OAuthClient and a BasicAuthClient share the same *http.Client")
	}
}

func TestDefaultTimeoutAndWithTimeout(t *testing.T) {
	if got := NewOAuth("id", "secret", true).Client.Timeout; got != defaultTimeout {
		t.Fatalf("default timeout = %v, want %v", got, defaultTimeout)
	}

	want := 5 * time.Second
	if got := NewOAuth("id", "secret", true, WithTimeout(want)).Client.Timeout; got != want {
		t.Fatalf("WithTimeout gave %v, want %v", got, want)
	}
	if got := NewBasicAuth("merchant", "key", true, WithTimeout(want)).Client.Timeout; got != want {
		t.Fatalf("WithTimeout gave %v, want %v", got, want)
	}
}

func TestWithTimeoutIgnoredWhenHTTPClientSupplied(t *testing.T) {
	hc := &http.Client{Timeout: time.Minute}
	c := NewOAuth("id", "secret", true, WithTimeout(time.Second), WithHTTPClient(hc))
	if c.Client.Timeout != time.Minute {
		t.Fatalf("WithHTTPClient client timeout was overwritten: %v", c.Client.Timeout)
	}
}

func TestWithBaseURLs(t *testing.T) {
	c := NewOAuth("id", "secret", true, WithBaseURLs("http://api", "http://app", "http://auth"))

	if got := ApiUri(c.Config); got != "http://api" {
		t.Errorf("ApiUri = %q, want http://api", got)
	}
	if got := AppUri(c.Config); got != "http://app" {
		t.Errorf("AppUri = %q, want http://app", got)
	}
	if got := c.tokenEndpoint(); got != "http://auth/connect/token" {
		t.Errorf("tokenEndpoint = %q, want http://auth/connect/token", got)
	}
}

func TestWithBaseURLsEmptyStringsKeepDefaults(t *testing.T) {
	c := NewOAuth("id", "secret", true, WithBaseURLs("http://api", "", ""))

	if got := ApiUri(c.Config); got != "http://api" {
		t.Errorf("ApiUri = %q, want http://api", got)
	}
	if got := AppUri(c.Config); got != "https://demo.vivapayments.com" {
		t.Errorf("AppUri = %q, want the demo default", got)
	}
	if got := c.authUri(); got != "https://demo-accounts.vivapayments.com" {
		t.Errorf("authUri = %q, want the demo default", got)
	}
}

func TestZeroValueOptionsAreIgnored(t *testing.T) {
	c := NewOAuth("id", "secret", true, WithHTTPClient(nil), WithTimeout(0), nil)

	if c.Client == nil {
		t.Fatal("WithHTTPClient(nil) left the client without an http.Client")
	}
	if c.Client.Timeout != defaultTimeout {
		t.Fatalf("WithTimeout(0) changed the timeout to %v", c.Client.Timeout)
	}
}

func TestOptionsAppliedInOrder(t *testing.T) {
	first := &http.Client{}
	second := &http.Client{}

	c := NewOAuth("id", "secret", true, WithHTTPClient(first), WithHTTPClient(second))
	if c.Client != second {
		t.Fatal("the later WithHTTPClient did not win")
	}
}
