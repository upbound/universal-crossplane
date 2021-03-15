# ====================================================================================
# Setup Project

PROJECT_NAME := project-uruk-hai
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Charts

CROSSPLANE_REPO := https://github.com/crossplane/crossplane.git
CROSSPLANE_TAG := v1.1.0

GATEWAY_TAG := v0.25.0-alpha1.54.g74ad71e
GRAPHQL_TAG := v0.25.0-alpha1.26.g63124dc-version-hack-1

export CROSSPLANE_TAG
export GATEWAY_TAG
export GRAPHQL_TAG

# ====================================================================================
# Setup Output

S3_BUCKET ?= project-uruk-hai.releases
-include build/makelib/output.mk

# ====================================================================================
# Setup Kubernetes tools
USE_HELM3 = true
HELM_CHART_LINT_STRICT = false
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Helm

HELM_BASE_URL = https://charts.upbound.io
HELM_S3_BUCKET = upbound.charts
HELM_CHARTS = project-uruk-hai
HELM_CHART_LINT_ARGS_project-uruk-hai = --set nameOverride='',imagePullSecrets=''
-include build/makelib/helm.mk

# ====================================================================================
# Setup Local Dev
-include build/makelib/local.mk

local-dev: local.up local.deploy.project-uruk-hai
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
	@rm -rf $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@mkdir -p $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/templates/* $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/crds/* $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/crds
	@$(OK) Crossplane chart has been fetched

generate-chart: crossplane
	@$(INFO) Generating values.yaml for the chart
	@cp -f $(HELM_CHARTS_DIR)/project-uruk-hai/values.yaml.tmpl $(HELM_CHARTS_DIR)/project-uruk-hai/values.yaml
	@cd $(HELM_CHARTS_DIR)/project-uruk-hai && $(SED_CMD) 's|%%CROSSPLANE_TAG%%|$(CROSSPLANE_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/project-uruk-hai && $(SED_CMD) 's|%%GATEWAY_TAG%%|$(GATEWAY_TAG)|g' values.yaml
	@cd $(HELM_CHARTS_DIR)/project-uruk-hai && $(SED_CMD) 's|%%GRAPHQL_TAG%%|$(GRAPHQL_TAG)|g' values.yaml
	@$(OK) Generating values.yaml for the chart

helm.prepare: generate-chart

reviewable: generate-chart lint

.PHONY: generate-chart crossplane submodules fallthrough reviewable