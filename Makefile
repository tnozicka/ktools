include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/make.mk
include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/go.mk
include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/golangci.mk
include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/codegen.mk
include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/links.mk
include ./vendor/github.com/tnozicka/k8s-controller-lib/makefiles/licenses.mk

all: build
.PHONY: all

# Force locale to sort files the same way for recursive traversals in all environments.
diff :=LC_COLLATE=C diff

GO_BUILD_PACKAGES :=./cmd/...

build: _default-build-target
.PHONY: build

update: _default-update-target
.PHONY: update

clean-generated: _default-clean-generated
.PHONY: clean-generated

verify-generated: tmpdir:=$(shell mktemp -d)
verify-generated:
	find ./ \
	-maxdepth 1 \
	-not -name '\.git' \
	-not -name 'vendor' \
	-exec cp -a -t '$(tmpdir)' {} +

	+@$(MAKE) -C '$(tmpdir)' clean clean-generated

	+@$(MAKE) -C '$(tmpdir)' update

	$(diff) -q -r --no-dereference --exclude='\.*' ./  '$(tmpdir)/'

	$(RM) -r '$(tmpdir)'
.PHONY: verify-generated

verify: _default-verify-target verify-generated
.PHONY: verify

test: _default-test-target
.PHONY: test

clean: _default-clean-target
clean:
.PHONY: clean
