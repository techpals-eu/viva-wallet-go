package vivawallet

// TransactionStatus is the status of a transaction, returned as the `statusId` / `StatusId`
// field by the retrieve, create and cancel transaction calls and by the payment webhooks.
//
// It is a string rather than a single character: the dispute statuses are two characters
// ("MA", "MI", "ML", "MW", "MS").
//
// Ref: https://developer.viva.com/integration-reference/response-codes/
type TransactionStatus string

const (
	// TransactionStatusFinished means the transaction completed successfully (PAYMENT SUCCESSFUL).
	TransactionStatusFinished TransactionStatus = "F"
	// TransactionStatusActive means the transaction is in progress (PAYMENT PENDING).
	TransactionStatusActive TransactionStatus = "A"
	// TransactionStatusCaptured refers to an original pre-auth transaction that has now been
	// captured. The capture itself is a separate transaction with status
	// TransactionStatusFinished.
	TransactionStatusCaptured TransactionStatus = "C"
	// TransactionStatusError means the transaction did not complete (PAYMENT UNSUCCESSFUL).
	TransactionStatusError TransactionStatus = "E"
	// TransactionStatusRefunded means the transaction was fully or partially refunded.
	TransactionStatusRefunded TransactionStatus = "R"
	// TransactionStatusCancelled means the transaction was cancelled by the merchant.
	TransactionStatusCancelled TransactionStatus = "X"
	// TransactionStatusDisputed means the cardholder has disputed the transaction with the
	// issuing bank.
	TransactionStatusDisputed TransactionStatus = "M"
	// TransactionStatusDisputeAwaitingResponse means the dispute is awaiting a response.
	TransactionStatusDisputeAwaitingResponse TransactionStatus = "MA"
	// TransactionStatusDisputeInProgress means the dispute is in progress.
	TransactionStatusDisputeInProgress TransactionStatus = "MI"
	// TransactionStatusDisputeLost means a disputed transaction has been refunded.
	TransactionStatusDisputeLost TransactionStatus = "ML"
	// TransactionStatusDisputeWon means the dispute was won.
	TransactionStatusDisputeWon TransactionStatus = "MW"
	// TransactionStatusDisputeSuspected means a dispute is suspected.
	TransactionStatusDisputeSuspected TransactionStatus = "MS"
)

// OrderState is the state of a payment order, returned as the `StateId` field by the
// retrieve order call.
//
// Note that this is a JSON number, unlike TransactionStatus. Viva's OpenAPI document
// annotates StateId as `type: string`, but its own response example returns an unquoted
// `3` and the documented values are the bare numbers below, so the annotation is a
// documentation bug.
//
// Ref: https://developer.viva.com/apis-for-payments/payment-api/
type OrderState int

const (
	// OrderStatePending means the order has not been paid yet.
	OrderStatePending OrderState = 0
	// OrderStateExpired means the order expired before it was paid.
	OrderStateExpired OrderState = 1
	// OrderStateCanceled means the merchant cancelled the order before it was paid.
	OrderStateCanceled OrderState = 2
	// OrderStatePaid means the order has been paid. An order that was paid and then had its
	// transaction cancelled stays in this state, since the cancellation is a separate
	// transaction against the same order code.
	OrderStatePaid OrderState = 3
)
