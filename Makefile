VERSION := $(shell node -e "console.log(require('./package.json').version)" 2>/dev/null || echo "0.0.0-dev")
BINARY  := aether
LDFLAGS := -X github.com/aether-colony/aether/cmd.Version=$(VERSION)

.PHONY: build test lint clean install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/aether/

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build
