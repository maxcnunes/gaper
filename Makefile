OS		:= $(shell uname -s)
TEST_PACKAGES	:= $(shell go list ./... | grep -v cmd)
COVER_PACKAGES	:= $(shell go list ./... | grep -v cmd | paste -sd "," -)
LINTER		:= $(shell command -v gometalinter 2> /dev/null)

build:
	@go build -o ./gaper cmd/gaper/main.go

## lint: Validate golang code
# Install it following this doc https://golangci-lint.run/usage/install/#local-installation,
# please use the same version from .github/workflows/workflow.yml.
lint:
	@golangci-lint run

test:
	@go test -p=1 -coverpkg $(COVER_PACKAGES) \
		-covermode=atomic -coverprofile=coverage.out $(TEST_PACKAGES)

cover: test
	@go tool cover -html=coverage.out

fmt:
	@find . -name '*.go' -not -wholename './vendor/*' | \
		while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
