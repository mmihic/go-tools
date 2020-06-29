SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
SHELL=/bin/bash -o pipefail

org_package_root          := github.com/chronosphereio
tools_bin_path            := $(abspath ./_tools/bin)
vendor_prefix             := vendor

BUILD                     := $(abspath ./bin)
BUILD_MOD_VENDOR          ?= true
VENDOR                    := $(repo_package)/$(vendor_prefix)
GO_BUILD_TAGS_LIST        :=
GO_BUILD_COMMON_ENV       ?= CGO_ENABLED=0
GO_PATH                   := ${GOPATH}
LOCAL_IS_DARWIN           := $(shell (uname | grep -i darwin >/dev/null) && echo -n true || echo -n false)

# LD Flags
GIT_REVISION              := $(shell git rev-parse --short HEAD)
GIT_BRANCH                := $(shell git rev-parse --abbrev-ref HEAD)
GIT_VERSION               := $(shell git describe --tags --abbrev=0 2>/dev/null || echo unknown)
BUILD_DATE                := $(shell date -u  +"%Y-%m-%dT%H:%M:%SZ") # Use RFC-3339 date format
BUILD_TS_UNIX             := $(shell date '+%s') # second since epoch

TOOLS :=   \
	pkgalign

# START_RULES general
.PHONY: setup
setup:
	mkdir -p $(BUILD)

.PHONY: all
all: tools
	@echo Made all successfully
# END_RULES general

.PHONY: tools
tools: $(TOOLS)

define TOOL_RULES

.PHONY: $(TOOL)
$(TOOL): setup
	@echo "--- Building $(TOOL)"
	go build -tags "$(GO_BUILD_TAGS_LIST)" -o $(BUILD)/$(TOOL) ./cmd/$(TOOL)/main/.

.PHONY: install-$(TOOL)
install-$(TOOL): $(TOOL)
	cp $(BUILD)/$(TOOL) $(HOME)/bin
endef

$(foreach TOOL,$(TOOLS),$(eval $(TOOL_RULES)))

# START_RULES tools

.PHONY: install-tools
install-tools:
	@echo "--- :golang: Installing tools"
	GOBIN=$(tools_bin_path) go install github.com/fossas/fossa-cli/cmd/fossa
	GOBIN=$(tools_bin_path) go install github.com/golangci/golangci-lint/cmd/golangci-lint
	GOBIN=$(tools_bin_path) go install github.com/axw/gocov/gocov

# END_RULES tools

.PHONY: test
test: export GO_BUILD_TAGS = $(GO_BUILD_TAGS_LIST)
test: test-base

.PHONY: test-cover
test-cover: test
	$(tools_bin_path)/gocov convert cover.out | $(tools_bin_path)/gocov report > coverage_report.out

.PHONY: go-mod-tidy
go-mod-tidy:
	@echo "--- :golang: tidying modules"
	go mod tidy

.PHONY: lint
lint: export GO_BUILD_TAGS = $(GO_BUILD_TAGS_LIST)
lint: install-tools
	@echo "--- :golang: Running linters"
	$(tools_bin_path)/golangci-lint run

.PHONY: install



.DEFAULT_GOAL := all
