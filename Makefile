# ====================================================================================
# Setup Project

PROJECT_NAME := universal-crossplane
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64

PACKAGE_NAME := universal-crossplane

EKS_ADDON_REGISTRY := 709825985650.dkr.ecr.us-east-1.amazonaws.com

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Versions

CROSSPLANE_REPO := https://github.com/upbound/crossplane.git
# Tag corresponds to Docker image tag while commit is git-compatible signature
# for pulling. They do not always match.
CROSSPLANE_TAG := v1.20.0-up.1
CROSSPLANE_COMMIT := v1.20.0-up.1

export CROSSPLANE_TAG

# ====================================================================================
# Setup Output

S3_BUCKET ?= public-upbound.releases/$(PACKAGE_NAME)
-include build/makelib/output.mk

# ====================================================================================
# Setup Go

GOLANGCILINT_VERSION = 1.64.8

GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/bootstrapper
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal
GO111MODULE = on
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

UP_VERSION = v0.14.0
UP_CHANNEL = stable

OLMBUNDLE_VERSION = v0.5.2
OLM_DIR=$(ROOT_DIR)/cluster/olm

USE_HELM3 = true
HELM_CHART_LINT_STRICT = false
CRDS_DIR=$(ROOT_DIR)/cluster/crds

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Helm

HELM_BASE_URL = https://charts.upbound.io
HELM_OCI_URL = xpkg.upbound.io/upbound
HELM_S3_BUCKET = public-upbound.charts
HELM_CHARTS = $(PACKAGE_NAME)
HELM_CHART_LINT_ARGS_$(PACKAGE_NAME) = --set nameOverride='',imagePullSecrets=''
HELM_DOCS_ENABLED = true
HELM_VALUES_TEMPLATE_SKIPPED = true
-include build/makelib/helm.mk
-include makelib/helmoci.mk

# ====================================================================================
# Setup Images
# Due to the way that the shared build logic works, images should
# all be in folders at the same level (no additional levels of nesting).

REGISTRY_ORGS ?= xpkg.upbound.io/upbound
IMAGES = uxp-bootstrapper
-include build/makelib/imagelight.mk

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

crossplane:
	@$(INFO) Fetching Crossplane chart $(CROSSPLANE_TAG)
	@rm -rf $(WORK_DIR)/crossplane
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
	@rm -f $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@cp -a $(WORK_DIR)/crossplane/cluster/charts/crossplane/values.yaml $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@# Note(turkenh): Using sed to replace the repository and tag values in the values.yaml of the upstream chart
	@# with the ones we want to use for the UXP chart. We also append the uxp-values.yaml to the values.yaml for UXP
	@# specific values.
	@# This is more like an interim solution until we need more differences between the upstream and UXP charts.
	@$(SED_CMD) 's|repository: crossplane/crossplane|repository: upbound/crossplane|g' '$(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml'
	@$(SED_CMD) 's|repository: crossplane/xfn|repository: upbound/xfn|g' '$(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml'
	@$(SED_CMD) 's|tag: ""|tag: "$(CROSSPLANE_TAG)"|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@cat $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/uxp-values.yaml >> $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(OK) Crossplane chart has been fetched

get-versions:
	@echo CROSSPLANE_VERSION=$(CROSSPLANE_TAG)
	@echo HELM_CHART_VERSION=$(HELM_CHART_VERSION)

eksaddon.chart: crossplane
	@$(INFO) Generating values.yaml for the EKS Add-on chart
	@$(SED_CMD) 's|repository: upbound/crossplane|repository: $(EKS_ADDON_REGISTRY)/upbound/crossplane|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(SED_CMD) 's|repository: xpkg.upbound.io/upbound/uxp-bootstrapper|repository: $(EKS_ADDON_REGISTRY)/upbound/uxp-bootstrapper|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@perl -i -0pe 's|rbacManager:\n  deploy: true|rbacManager:\n  deploy: false|' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(OK) Generating values.yaml for the EKS Add-on chart

# We have to give a static namespace for OLM bundle because it does not interpret
# and change the namespace of the subjects of ClusterRoleBindings to the namespace
# where the operator is deployed. See https://github.com/operator-framework/operator-lifecycle-manager/issues/1361
# and https://github.com/operator-framework/operator-lifecycle-manager/issues/2039

olm.build: $(HELM) $(OLMBUNDLE)
	@$(INFO) Generating OLM bundle
	@$(HELM) -n upbound-system template $(HELM_CHARTS_DIR)/$(PACKAGE_NAME) \
		--set upbound.controlPlane.permission=edit \
		--set securityContextCrossplane.runAsUser=null \
		--set securityContextCrossplane.runAsGroup=null \
		--set securityContextRBACManager.runAsUser=null \
		--set securityContextRBACManager.runAsGroup=null > $(WORK_DIR)/olm.yaml
	@$(SED_CMD) 's|release-name|$(PROJECT_NAME)|g' $(WORK_DIR)/olm.yaml
	@rm -rf $(OLM_DIR)/bundle
	@cat $(WORK_DIR)/olm.yaml | $(OLMBUNDLE) --version $(HELM_CHART_VERSION) --chart-file-path $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/Chart.yaml --extra-resources-dir $(CRDS_DIR) --output-dir $(OLM_DIR)

olm.artifacts: olm.build
	@mkdir -p $(abspath $(OUTPUT_DIR)/olm)
	@tar -czvf $(abspath $(OUTPUT_DIR)/olm/$(VERSION)).tar.gz  -C $(OLM_DIR)/bundle .

build.artifacts: olm.artifacts

generate.init: crossplane olm.build

local-dev: $(UP) local.up local.deploy.$(PACKAGE_NAME)

e2e.run: build local-dev
e2e.done: local.down

.PHONY: olm.build olm.artifacts crossplane submodules fallthrough
