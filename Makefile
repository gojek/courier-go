.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: vet
vet:
	@go vet ./...

.PHONY: lint
lint: golangci-lint
	@$(GOLANGCI_LINT) run --timeout=10m -v

.PHONY: imports
imports: goimports
	@$(GOIMPORTS) -w -local ***REMOVED*** -d `find . -type f -name '*.go'`

.PHONY: docs
docs: godoc
	@$(GODOC) ***REMOVED*** > README.md && \
		sed -i.bak '1s/^# courier$\/# Documentation/' README.md && rm README.md.bak
	@cat REPO_README.md | cat - README.md > temp && mv temp README.md
	@$(GODOC) ***REMOVED***/metrics > metrics/README.md

## test: Run all tests
test: check test-run test-cov
test-run:
	@go test ./... -covermode=count -coverprofile=test.cov

test-cov: gocov
	@$(GOCOV) convert test.cov | $(GOCOV) report

.PHONY: check
check: fmt vet lint imports docs
	@git diff --quiet || test $$(git diff --name-only | grep -v -e 'go.mod$$' -e 'go.sum$$' | wc -l) -eq 0 || ( echo "The following changes (result of code generators and code checks) have been detected:" && git --no-pager diff && false ) # fail if Git working tree is dirty

# ========= Helpers ===========

GOLANGCI_LINT = $(shell pwd)/.bin/golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint)

GOIMPORTS = $(shell pwd)/.bin/goimports
goimports:
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@v0.1.0)

GOCOV = $(shell pwd)/.bin/gocov
gocov:
	$(call go-get-tool,$(GOCOV),github.com/axw/gocov/gocov@v1.0.0)

GODOC = $(shell pwd)/.bin/godocdown
godoc:
	$(call go-get-tool,$(GODOC),github.com/robertkrimen/godocdown/godocdown)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
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
