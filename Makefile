# ====================================================================================
# Setup Project

PROJECT_NAME := crossplane-distro
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

PACKAGE_NAME := upbound-universal-crossplane

BOOTSTRAPPER_TAG := $(VERSION)
AGENT_TAG := v0.25.0-alpha1.76.g1ef3599
GRAPHQL_TAG := v0.25.0-alpha1.39.gcf9772d

export CROSSPLANE_TAG
export AGENT_TAG
export GRAPHQL_TAG

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Charts
CROSSPLANE_REPO := https://github.com/crossplane/crossplane.git
CROSSPLANE_TAG := v1.1.0

# ====================================================================================
# Setup Output

S3_BUCKET ?= $(PACKAGE_NAME).releases
-include build/makelib/output.mk

# ====================================================================================
# Setup Go

GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/bootstrapper
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal
GO111MODULE = on
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

USE_HELM3 = true
HELM_CHART_LINT_STRICT = false
CRDS_DIR=$(ROOT_DIR)/cluster/crds
OLM_DIR=$(ROOT_DIR)/cluster/olm
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Helm

HELM_BASE_URL = https://charts.upbound.io
HELM_S3_BUCKET = upbound.charts
HELM_CHARTS = $(PACKAGE_NAME)
HELM_CHART_LINT_ARGS_$(PACKAGE_NAME) = --set nameOverride='',imagePullSecrets=''
-include build/makelib/helm.mk

# ====================================================================================
# Setup Images
# Due to the way that the shared build logic works, images should
# all be in folders at the same level (no additional levels of nesting).

DOCKER_REGISTRY = upbound
IMAGES = uxp-bootstrapper
OSBASEIMAGE = gcr.io/distroless/static:nonroot
-include build/makelib/image.mk

# ====================================================================================
# Setup Local Dev
-include build/makelib/local.mk

local-dev: local.up local.deploy.$(PACKAGE_NAME)
# ====================================================================================
# Targets

# run `make help` to see the targets and options

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# TODO(muvaf): we don't need to handle crds folder after this PR is merged https://github.com/crossplane/crossplane/pull/2160
crossplane:
	@$(INFO) Fetching Crossplane chart $(CROSSPLANE_TAG)
	@mkdir -p $(WORK_DIR)/crossplane
	@git -C $(WORK_DIR)/crossplane init
	@git -C $(WORK_DIR)/crossplane remote add origin $(CROSSPLANE_REPO) 2>/dev/null || true
	@git -C $(WORK_DIR)/crossplane fetch origin refs/tags/$(CROSSPLANE_TAG):refs/tags/$(CROSSPLANE_TAG)
	@git -C $(WORK_DIR)/crossplane checkout $(CROSSPLANE_TAG)
	@rm -rf $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane
	@mkdir -p $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/templates/* $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/crds/* $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/crds
	@$(OK) Crossplane chart has been fetched

generate-chart: crossplane
	@$(INFO) Generating values.yaml for the chart
	@cp -f $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml.tmpl $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%BOOTSTRAPPER_TAG%%|$(BOOTSTRAPPER_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%CROSSPLANE_TAG%%|$(CROSSPLANE_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%AGENT_TAG%%|$(AGENT_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%GRAPHQL_TAG%%|$(GRAPHQL_TAG)|g' values.yaml
	@$(OK) Generating values.yaml for the chart

olm-bundle: $(HELM) $(OLMBUNDLE) generate-chart
	@$(INFO) Generating OLM bundle
	@$(HELM) -n upbound-system template $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) | $(OLMBUNDLE) --chart-file-path $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/Chart.yaml --extra-resources-dir $(CRDS_DIR) --output-dir $(OLM_DIR)

helm.prepare: generate-chart

reviewable: helm.prepare lint

.PHONY: generate-chart crossplane submodules fallthrough reviewable