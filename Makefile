# Project specific variables
COVER_FILE ?= coverage.out

# Main targets
.PHONY: test
test: ## Run unit tests
	@go test ./... -coverprofile=$(COVER_FILE)
	@go tool cover -func=$(COVER_FILE) | grep ^total

$(COVER_FILE):
	$(MAKE) test

.PHONY: cover
cover: $(COVER_FILE) ## Output coverage in human readable form in html
	go tool cover -html=$(COVER_FILE)
	rm -f $(COVER_FILE)

.PHONY: lint
lint: ## Check the project with lint
	golangci-lint run -v --fix

.PHONY: check
check: lint test ## Check project with static checks and unit tests

.PHONY: deps
deps: ## Manage go mod dependencies, beautify go.mod and go.sum files
	go mod tidy

.PHONY: clean
clean: ## Clean the project from built files
	rm -f $(COVER_FILE)

.PHONY: help
help: ## Print this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
