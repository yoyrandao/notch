BIN := notch
PKG := ./...

.PHONY: build test vet tidy run clean

build:
	go build -o bin/$(BIN) .

test:
	go test -race -count=1 $(PKG)

vet:
	go vet $(PKG)

tidy:
	go mod tidy

run:
	go run . $(ARGS)

clean:
	rm -rf bin/ dist/
