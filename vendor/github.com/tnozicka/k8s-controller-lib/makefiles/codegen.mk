CODEGEN_PKG :=k8s.io/code-generator
CODEGEN_HEADER_FILE ?=./hack/boilerplate.go.txt
DEEPCOPY_GEN :=$(GO) tool $(CODEGEN_PKG)/cmd/deepcopy-gen

update-deepcopy:
	$(DEEPCOPY_GEN) --go-header-file='$(CODEGEN_HEADER_FILE)' ./...
.PHONY: update-deepcopy

update-codegen: update-deepcopy
.PHONY: update-codegen

clean-codegen:
	find ./ -name '*.generated.deepcopy.go' -delete
.PHONY: clean-codegen

_default-update-target: update-codegen
_default-clean-generated: clean-codegen
