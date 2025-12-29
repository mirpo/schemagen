VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

.PHONY: install build test test-regression test-all update-expected lint lint-fix lint-testdata coverage clean

install:
	go mod download

clean:
	rm -f schemagen

build: clean
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT}" -o schemagen ./main.go

test:
	go clean -testcache
	go test ./... -v -race -count=1 -coverprofile=coverage.out

test-regression: build
	./scripts/verify-regression.sh

test-all: lint test test-regression

update-expected: build
	./scripts/update-expected.sh

lint:
	golangci-lint run --verbose

lint-fix:
	golangci-lint run --verbose --fix
	go mod tidy

lint-testdata:
	./scripts/lint-testdata.sh

coverage: test
	go tool cover -html=coverage.out
