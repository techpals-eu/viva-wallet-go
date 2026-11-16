all: build-example test vet lint

build-example:
	go build -o examp ./example/main.go

clean-example:
	rm -rf examp

lint:
	staticcheck

test:
	go test -race ./...

# test-verbose names every test as it runs. -count=1 bypasses the test cache, which
# otherwise reports "(cached)" and prints nothing on a re-run.
test-verbose:
	go test -race -count=1 -v ./...

# example runs the demo program against the live Viva demo API. It requires:
#   VIVA_CLIENT_ID, VIVA_CLIENT_SECRET, VIVA_MERCHANT_ID, VIVA_API_KEY
example:
	go run ./example

# fiscal-demo runs one step of the Greek fiscalisation flow against the Viva DEMO API, e.g.
#   make fiscal-demo STEP=zero        (start | zero | cash | terminal; cash and terminal need AA=n)
# Credentials come from the environment; see example/fiscal/main.go for the variables.
fiscal-demo:
	go run ./example/fiscal -step $(STEP) $(if $(AA),-aa $(AA))

vet:
	go vet ./...

.PHONY: clean-example test test-verbose example fiscal-demo
