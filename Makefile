# ====================================================================================
# Setup Project

PROJECT_NAME := ess-plugin-vault
PROJECT_REPO := github.com/crossplane-contrib/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

S3_BUCKET ?= crossplane.releases
-include build/makelib/output.mk

# ====================================================================================
# Setup Go

# Set a sane default so that the nprocs calculation below is less noisy on the initial
# loading of this file
NPROCS ?= 1

# each of our test suites starts a kube-apiserver and running many test suites in
# parallel can lead to high CPU utilization. by default we reduce the parallelism
# to half the number of CPU cores.
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))

GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/server
GO_LDFLAGS += -X $(GO_PROJECT)/pkg/version.Version=$(VERSION)
GO_SUBDIRS += cmd pkg apis
GO111MODULE = on
-include build/makelib/golang.mk

# ====================================================================================
# Setup Images
REGISTRY_ORGS ?= xpkg.upbound.io/crossplane-contrib
IMAGES = ess-plugin-vault-server
OSBASEIMAGE = gcr.io/distroless/static:nonroot
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup Helm

USE_HELM3 = true
HELM_BASE_URL = https://charts.crossplane.io
HELM_S3_BUCKET = crossplane.charts
HELM_CHARTS = $(PROJECT_NAME)
HELM_CHART_LINT_ARGS_$(PROJECT_NAME) = --set nameOverride='',imagePullSecrets=''
-include build/makelib/k8s_tools.mk
-include build/makelib/helm.mk
# ====================================================================================
# Setup Local Dev
-include build/makelib/local.mk
# ====================================================================================
# Fallthrough should be before all other targets

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# ====================================================================================
# Targets

# run `make help` to see the targets and options

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

# NOTE(hasheddan): we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(UP)

CRD_DIR=package/crds

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/server --debug


.PHONY: reviewable submodules fallthrough run crds.clean generate

# ====================================================================================
# Special Targets

define EXTERNAL_VAULT_HELP
External Vault Targets:
    reviewable         Ensure a PR is ready for review.
    submodules         Update the submodules, such as the common build scripts.

endef
export ESS_PLUGIN_VAULT_HELP

ess-plugin-vault.help:
	@echo "$$ESS_PLUGIN_VAULT_HELP"

local-dev: local.up local.deploy.$(PROJECT_NAME)

help-special: ess-plugin-vault.help

.PHONY: ess-plugin-vault.help help-special
