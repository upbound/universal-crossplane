# ====================================================================================
# Setup Project

PROJECT_NAME := ubc-distro
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

# ====================================================================================
# Setup Output

S3_BUCKET ?= ubc-distro.releases
-include build/makelib/output.mk

# ====================================================================================
# Setup Kubernetes tools

HELM_VERSION=v2.17.0
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Helm

HELM_BASE_URL = https://charts.upbound.io
HELM_S3_BUCKET = upbound.charts
HELM_CHARTS_DIR = $(ROOT_DIR)/charts
HELM_CHARTS = ubc-distro
HELM_CHART_LINT_ARGS_crossplane = --set nameOverride='',imagePullSecrets=''
-include build/makelib/helm.mk

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

TMPDIR := $(shell mktemp -d)

# TODO(muvaf): we don't need to handle crds folder after this PR is merged https://github.com/crossplane/crossplane/pull/2160
crossplane:
	@$(INFO) Fetching Crossplane chart $(CROSSPLANE_TAG)
	@git clone -b $(CROSSPLANE_TAG) $(CROSSPLANE_REPO) $(TMPDIR)/crossplane
	@rm -rf $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@mkdir -p $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@cp -a $(TMPDIR)/crossplane/cluster/charts/crossplane/templates/* $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/templates/crossplane
	@cp -a $(TMPDIR)/crossplane/cluster/charts/crossplane/crds $(HELM_CHARTS_DIR)/$(PROJECT_NAME)/crds
	@$(OK) Crossplane chart has been fetched


.PHONY: crossplane submodules fallthrough