all: build-example test vet lint

build-example:
	go build -o examp ./example/main.go

clean-example:
	rm -rf examp

lint:
	staticcheck

test:
	go run example/main.go -race ./...

vet:
	go vet ./...

.PHONY: clean-example
