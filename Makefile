ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

fmt:
	@$(call run-go-mod-dir,go vet ./...,"go fmt")

vet:
	@$(call run-go-mod-dir,go vet ./...,"go vet")

lint: golangci-lint
	@$(call run-go-mod-dir,$(GOLANGCI_LINT) run --timeout=10m -v,".bin/golangci-lint")

.PHONY: ci
ci: test test-cov test-xml

imports: gci
	@$(call run-go-mod-dir,$(GCI) -w -local github.com/gojek ./ | { grep -v -e 'skip file .*' || true; },".bin/gci")

.PHONY: gomod.tidy
gomod.tidy:
	@$(call run-go-mod-dir,go mod tidy,"go mod tidy")

## test: Run all tests
.PHONY: test
test: check test-run

test-run:
	@$(call run-go-mod-dir,go test -race -covermode=atomic -coverprofile=coverage.out ./...,"go test")

test-cov: gocov
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out > coverage.json)
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out | $(GOCOV) report)

test-xml: test-cov gocov-xml
	@jq -n '{ Packages: [ inputs.Packages ] | add }' $(shell find . -type f -name 'coverage.json' | sort) | $(GOCOVXML) > coverage.xml

.PHONY: check
check: fmt vet lint imports
	@git diff --quiet || test $$(git diff --name-only | grep -v -e 'go.mod$$' -e 'go.sum$$' | wc -l) -eq 0 || ( echo "The following changes (result of code generators and code checks) have been detected:" && git --no-pager diff && false ) # fail if Git working tree is dirty

# ========= Helpers ===========

GOLANGCI_LINT = $(shell pwd)/.bin/golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2)

GCI = $(shell pwd)/.bin/gci
gci:
	$(call go-get-tool,$(GCI),github.com/daixiang0/gci@v0.2.9)

GOCOV = $(shell pwd)/.bin/gocov
gocov:
	$(call go-get-tool,$(GOCOV),github.com/axw/gocov/gocov@v1.0.0)

GOCOVXML = $(shell pwd)/.bin/gocov-xml
gocov-xml:
	$(call go-get-tool,$(GOCOVXML),github.com/AlekSi/gocov-xml@v1.0.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/.bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# run-go-mod-dir runs the given $1 command in all the directories with
# a go.mod file
define run-go-mod-dir
set -e; \
for dir in $(ALL_GO_MOD_DIRS); do \
	[ -z $(2) ] || echo "$(2) $${dir}/..."; \
	cd "$(PROJECT_DIR)/$${dir}" && $(1); \
done;
endef
