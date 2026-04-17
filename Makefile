VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build clean

build:
	mkdir -p bin
	cd scanner && go build -ldflags "$(LDFLAGS)" -o ../bin/shrikeops ./cmd/cli/main.go

clean:
	rm -rf bin/
