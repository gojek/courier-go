ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
LOCAL_GO_BIN_DIR := $(PROJECT_DIR)/.bin
BIN_DIR := $(if $(LOCAL_GO_BIN_DIR),$(LOCAL_GO_BIN_DIR),$(GOPATH)/bin)

fmt:
	@$(call run-go-mod-dir,go vet ./...,"go fmt")

vet:
	@$(call run-go-mod-dir,go vet ./...,"go vet")

lint: golangci-lint
	@$(call run-go-mod-dir,$(GOLANGCI_LINT) run --timeout=10m -v,".bin/golangci-lint")

.PHONY: ci
ci: test test-cov test-xml

imports: gci
	@$(call run-go-mod-dir,$(GCI_BIN) write --skip-generated -s standard -s default -s "prefix(github.com/gojek)" github.com/gojek . | { grep -v -e 'skip file .*' || true; },".bin/gci")

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
check: fmt vet lint imports docs
	@git diff --quiet || test $$(git diff --name-only | grep -v -e 'go.mod$$' -e 'go.sum$$' | wc -l) -eq 0 || ( echo "The following changes (result of code generators and code checks) have been detected:" && git --no-pager diff && false ) # fail if Git working tree is dirty

docs: godoc
	@$(GODOC) --repository.default-branch main --repository.path / \
		--output './docs/docs/sdk/{{.ImportPath}}.md' ./...
	@mv ./docs/docs/sdk/.md ./docs/docs/sdk/SDK.md
	@mv ./docs/docs/sdk/xds.md ./docs/docs/sdk/xds/xDS.md

# ========= Helpers ===========

golangci-lint:
	$(call install-if-needed,GOLANGCI_LINT,github.com/golangci/golangci-lint/cmd/golangci-lint,v1.53.3)

gci:
	$(call install-if-needed,GCI_BIN,github.com/daixiang0/gci,v0.10.1)

godoc:
	$(call install-if-needed,GODOC,github.com/ajatprabha/gomarkdoc/cmd/gomarkdoc,master)

GOCOV = $(shell pwd)/.bin/gocov
gocov:
	$(call install-if-needed,GOCOV,github.com/axw/gocov/gocov,v1.0.0)

gocov-xml:
	$(call install-if-needed,GOCOVXML,github.com/AlekSi/gocov-xml,v1.0.0)

is-available = $(if $(wildcard $(LOCAL_GO_BIN_DIR)/$(1)),$(LOCAL_GO_BIN_DIR)/$(1),$(if $(shell command -v $(1) 2> /dev/null),yes,no))

define install-if-needed
	@if [ ! -f "$(BIN_DIR)/$(notdir $(2))" ]; then \
    	echo "Installing $(2)@$(3) in $(BIN_DIR)" ;\
    	set -e ;\
    	TMP_DIR=$$(mktemp -d) ;\
    	cd $$TMP_DIR ;\
    	go mod init tmp ;\
    	go get $(2)@$(3) ;\
    	go build -o $(BIN_DIR)/$(notdir $(2)) $(2);\
    	rm -rf $$TMP_DIR ;\
	fi
	$(eval $1 := $(BIN_DIR)/$(notdir $(2)))
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
