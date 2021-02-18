
help: ## Show this help
	@echo "Help"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-20s\033[93m %s\n", $$1, $$2}'

.PHONY: default
default: help

.PHONY: gen
gen:  ## Generates all.
	./scripts/gogen.sh

.PHONY: deps
deps:  ## Fixes the dependencies
	./scripts/deps.sh

.PHONY: unit-test
unit-test:  ## Runs unit test.
	./scripts/check/unit-test.sh

.PHONY: test
test: unit-test  ## Runs unit test.

.PHONY: check
check:  ## Runs checks in CI environment (without docker).
	./scripts/check/check.sh

.PHONY: integration-test
integration-test: ## Runs integraton test in CI environment (without docker).
	./scripts/check/integration-test.sh
