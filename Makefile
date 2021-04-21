.DEFAULT_GOAL := default

platform := $(shell uname)
pact_version := "1.88.45"
go_test_analyser_version := "0.8.0"

ifeq (${platform},Darwin)
pact_filename := "pact-${pact_version}-osx.tar.gz"
else
pact_filename := "pact-${pact_version}-linux-x86_64.tar.gz"
endif

GOFMT_FILES?=$$(find ./ -name '*.go')

default: build test

build:
	@find ./cmd/* -maxdepth 1 -type d -exec go install -mod=vendor "{}" \;

install-goimports:
	@if [ ! -f ./goimports ]; then \
		go get golang.org/x/tools/cmd/goimports; \
	fi

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

goimports:
	goimports -w $(GOFMT_FILES)

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

vendor-sync:
	@go mod vendor

docker-package: build
	@bash -c "source ./scripts/travis_release.sh && \
		TAG=\$${TRAVIS_TAG} $(CURDIR)/scripts/docker-package.sh ./cmd/pact-proxy"

docker-publish: docker-package
	@bash -c "$(CURDIR)/scripts/docker-publish.sh ./cmd/sepadd-gateway";
	@bash -c "$(CURDIR)/scripts/docker-publish.sh ./cmd/fake-step2-dd";

publish: docker-publish

install-pact-go:
	@if [ ! -d ./pact ]; then \
		echo "pact not installed, installing..."; \
		wget https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v${pact_version}/${pact_filename} -O /tmp/pactserver.tar.gz && tar -xzf /tmp/pactserver.tar.gz 2>/dev/null -C .; \
	fi

lint:
	@echo "go lint ."
	@golint $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Lint found errors in the source code. Please check the reported errors"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

checks: vet lint errcheck


