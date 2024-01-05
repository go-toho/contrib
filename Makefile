SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

GO_TEST_CMD = go test
GO_TEST_PKGS ?= ./...

default: help

ifeq ($(CI),true)
$(info Running in a CI environment, verbose mode is disabled)
else
VERBOSE="true"
endif

# include per-user customization after all variables are defined
-include Makefile.local

.PHONY: tidy
tidy: GO_VERSION=$(shell go mod edit -json | sed -En 's/"Go": "([^"]*).*/\1/p' | tr -d '[:blank:]')
tidy: TIDY_CMD=go mod tidy -compat=$(GO_VERSION)
tidy: ## Tidy Go modules
	@$(TIDY_CMD)

.PHONY: test
test: ## Run the test suite and/or any other tests
	$(if $(ENABLE_RACE),GORACE="strip_path_prefix=$(GOPATH)/src") $(GO_TEST_CMD) \
		$(if $(ENABLE_RACE),-race) $(if $(VERBOSE),-v) \
		-cover \
		-coverprofile=coverage.out \
		-covermode=atomic \
		-timeout=15m \
		$(GO_TEST_PKGS)

.PHONY: coverage
coverage: ## Open a web browser displaying coverage
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Remove build artifacts
	@rm -f $(if $(VERBOSE),-v) coverage.out

HELP_FORMAT="    \033[36m%-15s\033[0m %s\n"
.PHONY: help
help: ## Display this usage information
	@echo "Valid targets:"
	@echo $(MAKEFILE_LIST) | \
		xargs grep -E '^[^ ]+:.*?## .*$$' -h | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""

FORCE:
