GIT ?=git
GO_BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT_SHA = $(shell git rev-parse --short HEAD)
GIT_VERSION = $(shell git describe --long --tags --match='v*' --dirty 2>/dev/null || echo "v0.0.0")
GIT_TREE_STATE ?=$(shell ( ( [ ! -d ".git/" ] || $(GIT) diff --quiet ) && echo 'clean' ) || echo 'dirty')
GIT_REF ?= $(shell git rev-parse --abbrev-ref HEAD)

GO ?=go
goos :=$(shell $(GO) env GOOS)$(if $(filter $(.SHELLSTATUS),0),,$(error "can't evaluate GOOS"))
goarch :=$(shell $(GO) env GOARCH)$(if $(filter $(.SHELLSTATUS),0),,$(error "can't evaluate GOARCH"))
GO_PACKAGE ?=$(shell $(GO) list -m -f '{{ .Path }}')$(if $(filter $(.SHELLSTATUS),0),,$(error "can't detect go package"))
GO_BUILD_PACKAGES ?=./cmd/...
GO_BUILD_PACKAGES_EXPANDED ?=$(shell $(GO) list $(GO_BUILD_PACKAGES))$(if $(filter $(.SHELLSTATUS),0),,$(error "can't list build packages"))
GO_BUILD_FLAGS ?=
GO_TEST_PACKAGES ?=$(shell $(GO) list -find -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./...)$(if $(filter $(.SHELLSTATUS),0),,$(error "can't find go test files"))
GO_TEST_FLAGS ?=-race
GO_TEST_EXTRA_FLAGS ?=
GO_LD_EXTRA_FLAGS ?=
define version-ldflags
-X "$(1).GitCommit=$(GIT_COMMIT_SHA)" \
-X "$(1).GitRef=$(GIT_REF)" \
-X "$(1).Version=$(GIT_VERSION)" \
-X $(1).gitTreeState="$(GIT_TREE_STATE)" \
-X "$(1).BuildDate=$(GO_BUILD_DATE)"
endef
GO_LD_FLAGS ?= -ldflags '-s -w $(strip $(call version-ldflags,$(GO_PACKAGE)/pkg/version) $(GO_LD_EXTRA_FLAGS))'

GOFMT ?=gofmt
GOFMT_FLAGS ?=-s -l

# $1 - package name
define build-package
	$(info Compiling $(notdir $(1)) for $(goos)/$(goarch)...)
	$(strip CGO_ENABLED=0 $(GO) build $(GO_BUILD_FLAGS) $(GO_LD_FLAGS) $(1))

endef

build-go:
	$(if $(strip $(GO_BUILD_PACKAGES_EXPANDED)),,$(error no packages to build: GO_BUILD_PACKAGES_EXPANDED var is empty))
	$(foreach package,$(GO_BUILD_PACKAGES_EXPANDED),$(call build-package,$(package)))
.PHONY: build-go
_default-build-target: build-go

# $1 - cmd
define exec-on-every-go-file
	find ./ -not \( -path '\\.*' -o -path '*/vendor/*' \) -name '*.go' -exec $(1) {} +
endef

update-gofmt:
	$(info Running $(GOFMT) $(GOFMT_FLAGS) -w)
	$(call exec-on-every-go-file,$(GOFMT) $(GOFMT_FLAGS) -w)
.PHONY: update-gofmt
_default-update-target: update-gofmt

update-gofix:
	go fix ./...
.PHONY: update-gofix
_default-update-target: update-gofix

update-deps-go:
	$(GO) mod tidy
	$(GO) mod vendor
.PHONY: update-deps-go
_default-update-target: update-deps-go

verify-govet:
	$(GO) vet ./...
.PHONY: verify-govet
_default-verify-target: verify-govet

verify-gofmt:
	$(info Running $(GOFMT) $(GOFMT_FLAGS))
	@output=$$( $(call exec-on-every-go-file,$(GOFMT) $(GOFMT_FLAGS)) ); \
	if [ -n "$${output}" ]; then \
		echo "$@ failed - please run \`make update-gofmt\` to fix following files:" > /dev/stderr; \
		echo "$${output}" > /dev/stderr; \
		exit 1; \
	else \
		echo "gofmt successfully verified" > /dev/stderr; \
	fi;
.PHONY: verify-gofmt
_default-verify-target: verify-gofmt

test-unit-go:
	$(strip $(GO) test $(GO_TEST_FLAGS) $(GO_TEST_EXTRA_FLAGS) $(GO_TEST_PACKAGES))
.PHONY: test-unit-go
_default-test-target: test-unit-go
_default-test-unit-target: test-unit-go

# $1 - package name
define go-clean-package
	if [[ -f '$(1)' ]]; then $(RM) '$(1)'; fi

endef

clean-binary-go:
	$(foreach package,$(GO_BUILD_PACKAGES_EXPANDED),$(call go-clean-package,$(package)))
.PHONY: clean-binary-go
_default-clean-target: clean-binary-go
