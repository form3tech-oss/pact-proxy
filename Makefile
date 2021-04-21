PLATFORM                 := $(shell uname)
NAME                     := $(shell basename $(CURDIR))
GOFMT_FILES              ?= $$(find ./ -name '*.go' | grep -v vendor | grep -v externalmodels)
GOTEST_DIRECTORIES       ?= $$(find ./internal/ -type f -iname "*_test.go" -exec dirname {} \; | uniq)
PACT_VERSION             ?= "1.88.45"

export GO111MODULE=on
export GOPRIVATE=github.com/form3tech-oss
export GOFLAGS=-mod=vendor

ifeq (${platform},Darwin)
PACT_FILE := "pact-${PACT_VERSION}-osx.tar.gz"
else
PACT_FILE := "pact-${PACT_VERSION}-linux-x86_64.tar.gz"
endif

.PHONY: build
build:
	@echo "==> Building..."
	@go install ./...

.PHONY: test
test: install-pact
	@echo "==> Executing tests..."
	@echo ${GOTEST_DIRECTORIES} | xargs -n1 go test --timeout 30m -v -count 1

.PHONY: goimports
goimports: install-goimports
	goimports -w $(GOFMT_FILES)

.PHONY: install-goimports
install-goimports:
	@type goimports >/dev/null 2>&1 || (cd /tmp && go get golang.org/x/tools/cmd/goimports && cd -)

.PHONY: docker-package
docker-package: build
	@echo "==> Building docker image..."
	@for f in ./cmd/*/; do \
		./scripts/docker-package.sh "$$f"; \
	done

.PHONY: vendor
vendor:
	@go mod tidy && go mod vendor && go mod verify

.PHONY: install-pact
install-pact:
	@if [ ! -d ./pact ]; then \
        echo "pact not installed, installing..."; \
        wget --quiet https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v${PACT_VERSION}/${PACT_FILE} -O /tmp/pactserver.tar.gz && tar -xzf /tmp/pactserver.tar.gz 2>/dev/null -C .; \
    fi