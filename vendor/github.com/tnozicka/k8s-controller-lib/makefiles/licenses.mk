DISALLOWED_LICENCE_TYPES ?=reciprocal,restricted,forbidden,unknown
IGNORED_PACKAGE_LICENCES ?=

verify-licences:
	$(GO) tool github.com/google/go-licenses/v2 check ./... --disallowed_types=$(DISALLOWED_LICENCE_TYPES) $(addprefix --ignore=,$(IGNORED_PACKAGE_LICENCES))
.PHONY: verify-licences
_default-verify-target: verify-licences
