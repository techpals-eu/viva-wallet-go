# 13 — CheckoutOrder.Customer is an anonymous struct

**Severity:** Low
**File:** [../order_payments.go](../order_payments.go)

## Problem

```go
type CheckoutOrder struct {
    Amount               int64  `json:"amount"`
    CustomerTransactions string `json:"customerTrns,omitempty"`
    Customer             struct {
        Email       string `json:"email,omitempty"`
        FullName    string `json:"fullName,omitempty"`
        Phone       string `json:"phone,omitempty"`
        CountryCode string `json:"countryCode,omitempty"`
        RequestLang string `json:"requestLang,omitempty"`
    } `json:"customer,omitempty"`
    ...
}
```

The nested customer is an anonymous struct type.

## Impact

Callers cannot build a customer separately, pass one between functions, or write a helper that
returns one. The only way to populate it is field-by-field on an existing `CheckoutOrder`:

```go
o := vivawallet.CheckoutOrder{Amount: 1000}
o.Customer.Email = "a@b.com"
o.Customer.FullName = "..."
```

There is no way to express `func customerFor(u User) vivawallet.Customer`. For a type that maps
onto the caller's own user model, that is the single most natural thing to want.

Separately: `omitempty` on a struct field does nothing — structs are never empty for
`encoding/json`. An all-zero customer currently serialises as `"customer":{}`. Whether Viva minds
is untested; if it does, the field needs to be a pointer.

## Fix

```go
type Customer struct {
    Email       string `json:"email,omitempty"`
    FullName    string `json:"fullName,omitempty"`
    Phone       string `json:"phone,omitempty"`
    CountryCode string `json:"countryCode,omitempty"`
    RequestLang string `json:"requestLang,omitempty"`
}

type CheckoutOrder struct {
    Amount               int64     `json:"amount"`
    CustomerTransactions string    `json:"customerTrns,omitempty"`
    Customer             *Customer `json:"customer,omitempty"`
    ...
}
```

Making it a **pointer** both fixes the `omitempty` semantics and is the shape callers want. It is a
breaking change for anyone doing `o.Customer.Email = ...` on a zero value (nil deref), so it
belongs in the v2 batch with [09](09-context-support.md).

If shipping in v1, extract the named type but keep it a value field — non-breaking, since the field
layout and JSON are identical — and defer the pointer change to v2.

## Note

The same `Customer` shape appears in the ISV order body in the fiscalization work, so a named type
gets reused there rather than duplicated.

## Verification

- Round-trip test: a `CheckoutOrder` with no customer must not emit a `customer` key; one with a
  customer must emit exactly the populated fields.
