ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
DOCS_EXCLUDE := "benchmark"

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
	@$(GODOC) --repository.url "https://***REMOVED***/-" \
		--repository.default-branch master --repository.path / \
		--output '{{.Dir}}/README.md' ./...
	@cat REPO_README.md | cat - README.md > temp && mv temp README.md

.PHONY: update-proto
update-proto:
	@$(MAKE) -C webhook/grpc update-proto

.PHONY: generate
generate:
	@$(call run-go-mod-dir,go generate ./...,"go generate")

.PHONY: gomod.tidy
gomod.tidy:
	@$(call run-go-mod-dir,go mod tidy,"go mod tidy")

## test: Run all tests
.PHONY: test
test: check test-run test-cov

test-run:
	@$(call run-go-mod-dir,go test ./... -covermode=count -coverprofile=coverage.out,"go test")

test-cov: gocov
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out | $(GOCOV) report)

.PHONY: check
check: fmt vet lint imports generate docs
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

GODOC = $(shell pwd)/.bin/gomarkdoc
godoc:
	$(call go-get-tool,$(GODOC),github.com/princjef/gomarkdoc/cmd/gomarkdoc)

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

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
