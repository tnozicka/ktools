GOLANGCI-LINT :=$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint

verify-golangci-lint:
	$(GOLANGCI-LINT) run ./...
.PHONY: verify-golangci-lint
_default-verify-target: verify-golangci-lint


update-golangci-lint:
	$(GOLANGCI-LINT) run ./... --fix
.PHONY: update-golangci-lint
_default-update-target: update-golangci-lint
