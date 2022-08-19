PLATFORM                 := $(shell uname)
NAME                     := $(shell basename $(CURDIR))
GOFMT_FILES              ?= $$(find ./ -name '*.go' | grep -v vendor | grep -v externalmodels)
GOTEST_DIRECTORIES       ?= $$(find ./internal/ -type f -iname "*_test.go" -exec dirname {} \; | uniq)
PACT_VERSION             ?= "1.88.90"

export GOPRIVATE=github.com/form3tech-oss
export GOFLAGS=-mod=vendor

DOCKER_IMG ?= form3tech/pact-proxy

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

.PHONY: vendor
vendor:
	@go mod tidy && go mod vendor && go mod verify

.PHONY: install-pact
install-pact:
	@if [ ! -d ./pact ]; then \
        echo "pact not installed, installing..."; \
		echo ${PACT_VERSION}
		ehcho ${PACT_FILE}
        wget --quiet https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v${PACT_VERSION}/${PACT_FILE} -O /tmp/pactserver.tar.gz && tar -xzf /tmp/pactserver.tar.gz 2>/dev/null -C .; \
    fi

.PHONY: publish
publish:
	@echo "==> Building docker image..."
	docker build --build-arg APPNAME=pact-proxy -f build/package/pact-proxy/Dockerfile -t $(DOCKER_IMG):$(TRAVIS_TAG) .
	@echo "==> Logging in to the docker registry..."
	echo "$(DOCKER_PASSWORD)" | docker login -u "$(DOCKER_USERNAME)" --password-stdin
	@echo "==> Pushing built image..."
	docker push $(DOCKER_IMG):$(TRAVIS_TAG)
	docker tag $(DOCKER_IMG):$(TRAVIS_TAG) $(DOCKER_IMG):latest
	docker push $(DOCKER_IMG):latest
