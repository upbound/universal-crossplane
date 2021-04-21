# ====================================================================================
# Setup Project

PROJECT_NAME := universal-crossplane
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64

PACKAGE_NAME := universal-crossplane

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Versions

CROSSPLANE_REPO := https://github.com/crossplane/crossplane.git
CROSSPLANE_TAG := v1.2.0-rc.0-133-g2e227c16

BOOTSTRAPPER_TAG := $(VERSION)
AGENT_TAG := $(VERSION)
GRAPHQL_TAG := v0.25.0-alpha1.41.g119b42a
XGQL_TAG := v0.0.0-123.gadb10e6

export BOOTSTRAPPER_TAG
export AGENT_TAG
export GRAPHQL_TAG
export XGQL_TAG
export CROSSPLANE_TAG

# ====================================================================================
# Setup Output

S3_BUCKET ?= upbound.releases/$(PACKAGE_NAME)
-include build/makelib/output.mk

# ====================================================================================
# Setup Go

GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/bootstrapper $(GO_PROJECT)/cmd/upbound-agent
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
IMAGES = uxp-bootstrapper upbound-agent
-include build/makelib/image.mk

# ====================================================================================
# Setup Local Dev
-include build/makelib/local.mk
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

GITCP_CMD?=git -C $(WORK_DIR)/crossplane
CROSSPLANE_COMMIT := $(shell echo $(CROSSPLANE_TAG) | sed -E 's/(.*)\./\1-/' | sed -E 's/(.*)\./\1-/')

crossplane:
	@$(INFO) Fetching Crossplane chart $(CROSSPLANE_TAG)
	@mkdir -p $(WORK_DIR)/crossplane
	@$(GITCP_CMD) init
	@$(GITCP_CMD) remote add origin $(CROSSPLANE_REPO) 2>/dev/null || true
	@$(GITCP_CMD) fetch origin
	@$(GITCP_CMD) checkout $(CROSSPLANE_COMMIT)
	@mkdir -p $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane
	@rm -f $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane/*
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/templates/* $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/templates/crossplane
	@rm -f $(CRDS_DIR)/*
	@cp -a $(WORK_DIR)/crossplane/cluster/crds/* $(CRDS_DIR)
	@$(OK) Crossplane chart has been fetched

generate-chart: crossplane
	@$(INFO) Generating values.yaml for the chart
	@cp -f $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml.tmpl $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%BOOTSTRAPPER_TAG%%|$(BOOTSTRAPPER_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%CROSSPLANE_TAG%%|$(CROSSPLANE_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%AGENT_TAG%%|$(AGENT_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%GRAPHQL_TAG%%|$(GRAPHQL_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) && $(SED_CMD) 's|%%XGQL_TAG%%|$(XGQL_TAG)|g' values.yaml
	@$(OK) Generating values.yaml for the chart

# We have to give a static namespace for OLM bundle because it does not interpret
# and change the namespace of the subjects of ClusterRoleBindings to the namespace
# where the operator is deployed. See https://github.com/operator-framework/operator-lifecycle-manager/issues/1361
# and https://github.com/operator-framework/operator-lifecycle-manager/issues/2039

olm: $(HELM) $(OLMBUNDLE) generate-chart
	@$(INFO) Generating OLM bundle
	@$(HELM) -n upbound-system template $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) --set upbound.controlPlane.connect=true > $(WORK_DIR)/olm.yaml
	@$(SED_CMD) 's|RELEASE-NAME|$(PROJECT_NAME)|g' $(WORK_DIR)/olm.yaml
	@rm -rf $(OLM_DIR)/bundle
	@cat $(WORK_DIR)/olm.yaml | $(OLMBUNDLE) --version $(HELM_CHART_VERSION) --chart-file-path $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/Chart.yaml --extra-resources-dir $(CRDS_DIR) --output-dir $(OLM_DIR)

helm.prepare: generate-chart

# Ensure a PR is ready for review.
reviewable: helm.prepare lint

# Ensure branch is clean.
check-diff: reviewable
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || $(FAIL)
	@$(OK) branch is clean

local-dev: local.up local.deploy.$(PACKAGE_NAME)

.PHONY: generate-chart crossplane submodules fallthrough reviewable