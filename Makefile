BINARY  := shortcuts
VERSION ?= dev
LDFLAGS := -s -w -X github.com/ejoja/shortcuts/internal/buildinfo.Version=$(VERSION)

.PHONY: build tidy test vet fmt e2e clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

tidy:
	go mod tidy

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

e2e: build
	./scripts/e2e.sh

clean:
	rm -f $(BINARY)
