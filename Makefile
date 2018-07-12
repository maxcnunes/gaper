OS		:= $(shell uname -s)
TEST_PACKAGES	:= $(shell go list ./... | grep -v cmd)
COVER_PACKAGES	:= $(shell go list ./... | grep -v cmd | paste -sd "," -)
LINTER		:= $(shell command -v gometalinter 2> /dev/null)

.PHONY: setup

setup:
ifeq ($(OS), Darwin)
	brew install dep
else
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
endif
	dep ensure -vendor-only
ifndef LINTER
	@echo "Installing linter"
	@go get -u github.com/alecthomas/gometalinter
	@gometalinter --install
endif

build:
	@go build -o ./gaper cmd/gaper/main.go

## lint: Validate golang code
lint:
	@gometalinter \
		--deadline=120s \
		--line-length=120 \
		--enable-all \
		--vendor ./...

test:
	@go test -p=1 -coverpkg $(COVER_PACKAGES) \
		-covermode=atomic -coverprofile=coverage.out $(TEST_PACKAGES)

cover: test
	@go tool cover -html=coverage.out

fmt:
	@find . -name '*.go' -not -wholename './vendor/*' | \
		while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
