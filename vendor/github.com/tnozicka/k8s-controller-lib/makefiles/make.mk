SHELL :=/usr/bin/env bash -euEo pipefail -O inherit_errexit

# TMPDIR changes the default location of temporary file for `mktemp` to avoid writing to much data into RAM based tmpfs on some systems.
TMPDIR ?=/var/tmp
export TMPDIR

define run-make
	+@$(MAKE) -C '$(1)' $(2)

endef

_default-build-target:
.PHONY: _default-build-target

_default-test-target:
.PHONY: _default-test-target

_default-test-unit-target:
.PHONY: _default-test-unit-target

_default-update-target:
.PHONY: _default-update-target

_default-verify-target:
.PHONY: _default-verify-target
