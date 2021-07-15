ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

fmt:
	@$(call run-go-mod-dir,go vet ./...,"go fmt")

vet:
	@$(call run-go-mod-dir,go vet ./...,"go vet")

lint: golangci-lint
	@$(call run-go-mod-dir,$(GOLANGCI_LINT) run --timeout=10m -v,".bin/golangci-lint")

.PHONY: ci
ci: test

imports: gci
	@$(call run-go-mod-dir,$(GCI) -w -local ***REMOVED*** ./ | { grep -v -e 'skip file .*' || true; },".bin/gci")

docs: godoc
	@$(GODOC) ***REMOVED*** > README.md && \
		sed -i.bak '1s/^# courier$\/# Documentation\n/' README.md && rm README.md.bak
	@cat REPO_README.md | cat - README.md > temp && mv temp README.md
	@$(GODOC) ***REMOVED***/metrics > metrics/README.md
	@cd otelcourier && $(GODOC) ***REMOVED***/otelcourier > README.md && cd ..

## test: Run all tests
.PHONY: test
test: check test-run test-cov

test-run:
	@$(call run-go-mod-dir,go test ./... -covermode=count -coverprofile=coverage.out,"go test")

test-cov: gocov
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out | $(GOCOV) report)

.PHONY: check
check: fmt vet lint imports docs
	@git diff --quiet || test $$(git diff --name-only | grep -v -e 'go.mod$$' -e 'go.sum$$' | wc -l) -eq 0 || ( echo "The following changes (result of code generators and code checks) have been detected:" && git --no-pager diff && false ) # fail if Git working tree is dirty

# ========= Helpers ===========

GOLANGCI_LINT = $(shell pwd)/.bin/golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint)

GCI = $(shell pwd)/.bin/gci
gci:
	$(call go-get-tool,$(GCI),github.com/daixiang0/gci@v0.2.9)

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

# run-go-mod-dir runs the given $1 command in all the directories with
# a go.mod file
define run-go-mod-dir
set -e; \
for dir in $(ALL_GO_MOD_DIRS); do \
	[ -z $(2) ] || echo "$(2) $${dir}/..."; \
	cd "$${dir}" && $(1); \
done;
endef
