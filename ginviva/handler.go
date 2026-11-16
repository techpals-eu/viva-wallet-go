// Package ginviva serves Viva webhooks from a Gin application.
//
// It is a separate module from the vivawallet SDK so that the SDK itself stays free of
// dependencies: everything here is a thin adapter over vivawallet.ParseWebhook and
// GetWebhookVerificationKey, which non-Gin applications can use directly.
package ginviva

import (
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	vivawallet "github.com/techpals-eu/viva-wallet-go"
)

// defaultMaxBodyBytes caps a webhook body when Config.MaxBodyBytes is unset. The documented
// payloads are a few kilobytes.
const defaultMaxBodyBytes = 1 << 20

// KeySource fetches the webhook verification key. *vivawallet.BasicAuthClient serves
// merchant-level webhooks and *vivawallet.ISVClient serves ISV-level ones.
type KeySource interface {
	GetWebhookVerificationKey() (*vivawallet.WebhookVerificationKey, error)
}

// Config holds the callbacks for the events an application handles. Every callback is
// optional.
//
// A callback runs before Viva gets its response, so return only once the event is durably
// handled. Returning an error answers 500, and Viva redelivers the event later. Viva can
// deliver an event more than once, including after a success, so deduplicate on
// WebhookEvent.MessageID; the context carries the event under EventKey.
type Config struct {
	OnPaymentCreated                   func(*gin.Context, *vivawallet.TransactionEventData) error
	OnReversalCreated                  func(*gin.Context, *vivawallet.TransactionEventData) error
	OnTransactionFailed                func(*gin.Context, *vivawallet.TransactionEventData) error
	OnAccountConnected                 func(*gin.Context, *vivawallet.AccountConnectedData) error
	OnAccountVerificationStatusChanged func(*gin.Context, *vivawallet.AccountVerificationStatusChangedData) error

	// OnUnhandled is called for an event with no callback above, including event types this
	// SDK does not know. Such events are acknowledged with 200 either way: failing them
	// would only make Viva redeliver what nothing will handle.
	OnUnhandled func(*gin.Context, *vivawallet.WebhookEvent)

	// MaxBodyBytes caps the size of a webhook body. Zero means 1 MiB.
	MaxBodyBytes int64
}

// EventKey is the gin.Context key under which Handle stores the *vivawallet.WebhookEvent
// being delivered, for callbacks that need the envelope, e.g. its MessageID.
const EventKey = "ginviva.event"

// Handler serves both halves of a Viva webhook endpoint: the GET verification handshake and
// the POST event delivery. Both must answer on the same URL, so register it on one route:
//
//	h := ginviva.NewHandler(client, ginviva.Config{
//	    OnPaymentCreated: func(c *gin.Context, d *vivawallet.TransactionEventData) error { ... },
//	})
//	r.Any("/webhooks/viva", h.Handle)
type Handler struct {
	keys KeySource
	cfg  Config

	// mu guards key and is held across the fetch, so concurrent verification GETs share one
	// call to Viva.
	mu  sync.Mutex
	key string
}

// NewHandler returns a Handler that answers verification with the key from keys and
// dispatches events to the callbacks in cfg.
func NewHandler(keys KeySource, cfg Config) *Handler {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	return &Handler{keys: keys, cfg: cfg}
}

// Handle is the gin.HandlerFunc for the webhook route.
func (h *Handler) Handle(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.serveKey(c)
	case http.MethodPost:
		h.serveEvent(c)
	default:
		c.Header("Allow", "GET, POST")
		c.AbortWithStatus(http.StatusMethodNotAllowed)
	}
}

// serveKey answers Viva's verification GET. The key is fetched once and cached: Viva
// re-verifies periodically, and fetching per probe would spend an authenticated API call
// on each.
func (h *Handler) serveKey(c *gin.Context) {
	key, err := h.verificationKey()
	if err != nil {
		_ = c.Error(err)
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}

	// Lowercase "key", as in Viva's own example endpoint for both merchants and ISVs.
	c.JSON(http.StatusOK, vivawallet.WebhookVerificationKey{Key: key})
}

func (h *Handler) verificationKey() (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.key != "" {
		return h.key, nil
	}

	k, err := h.keys.GetWebhookVerificationKey()
	if err != nil {
		return "", err
	}
	h.key = k.Key

	return h.key, nil
}

// serveEvent answers an event delivery. Viva redelivers on any non-2xx, once an hour for a
// day, so the status is chosen by whether redelivery can help: 500 when a callback failed,
// 400 for a body that will never parse, 200 otherwise.
func (h *Handler) serveEvent(c *gin.Context) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, h.cfg.MaxBodyBytes))
	if err != nil {
		_ = c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	event, err := vivawallet.ParseWebhook(body)
	if err != nil {
		_ = c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Set(EventKey, event)

	var handled bool
	switch event.EventTypeID {
	case vivawallet.EventTransactionPaymentCreated:
		handled, err = dispatch(c, event.TransactionPaymentCreated, h.cfg.OnPaymentCreated)
	case vivawallet.EventTransactionReversalCreated:
		handled, err = dispatch(c, event.TransactionReversalCreated, h.cfg.OnReversalCreated)
	case vivawallet.EventTransactionFailed:
		handled, err = dispatch(c, event.TransactionFailed, h.cfg.OnTransactionFailed)
	case vivawallet.EventAccountConnected:
		handled, err = dispatch(c, event.AccountConnected, h.cfg.OnAccountConnected)
	case vivawallet.EventAccountVerificationStatusChanged:
		handled, err = dispatch(c, event.AccountVerificationStatusChanged, h.cfg.OnAccountVerificationStatusChanged)
	}
	if err != nil {
		// dispatch has already set the status.
		_ = c.Error(err)
		return
	}

	if !handled && h.cfg.OnUnhandled != nil {
		h.cfg.OnUnhandled(c, event)
	}
	c.Status(http.StatusOK)
}

// dispatch decodes the event data and runs callback on it. It reports false when there is
// no callback, and sets the response status itself when it returns an error.
func dispatch[T any](c *gin.Context, decode func() (*T, error), callback func(*gin.Context, *T) error) (bool, error) {
	if callback == nil {
		return false, nil
	}

	data, err := decode()
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return true, err
	}

	if err := callback(c, data); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return true, err
	}

	return true, nil
}
