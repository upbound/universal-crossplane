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

CROSSPLANE_REPO := https://github.com/upbound/crossplane.git
# Tag corresponds to Docker image tag while commit is git-compatible signature
# for pulling. They do not always match.
CROSSPLANE_TAG := v1.5.0-rc.0.up.1
CROSSPLANE_COMMIT := v1.5.0-rc.0-up.1

BOOTSTRAPPER_TAG := $(VERSION)
AGENT_TAG := $(VERSION)
XGQL_TAG := v0.1.5

export BOOTSTRAPPER_TAG
export AGENT_TAG
export XGQL_TAG
export CROSSPLANE_TAG

# ====================================================================================
# Setup Output

S3_BUCKET ?= public-upbound.releases/$(PACKAGE_NAME)
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

OLMBUNDLE_VERSION = v0.5.2
USE_HELM3 = true
HELM_CHART_LINT_STRICT = false
CRDS_DIR=$(ROOT_DIR)/cluster/crds
OLM_DIR=$(ROOT_DIR)/cluster/olm
-include build/makelib/k8s_tools.mk

# up download and install
# TODO(hasheddan): move to build submodule when appropriate
UP_VERSION ?= v0.1.0
export UP := $(TOOLS_HOST_DIR)/up-$(UP_VERSION)
$(UP):
	@$(INFO) installing up $(UP_VERSION)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(UP) https://cli.upbound.io/stable/$(UP_VERSION)/bin/$(SAFEHOST_PLATFORM)/up || $(FAIL)
	@chmod +x $(UP)
	@$(OK) installing up $(UP)

# ====================================================================================
# Setup Helm

HELM_BASE_URL = https://charts.upbound.io
HELM_S3_BUCKET = public-upbound.charts
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

helm.prepare.universal-crossplane: crossplane
	@$(INFO) Generating values.yaml for the chart
	@cp -f $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml.tmpl $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(SED_CMD) 's|%%BOOTSTRAPPER_TAG%%|$(BOOTSTRAPPER_TAG)|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(SED_CMD) 's|%%CROSSPLANE_TAG%%|$(CROSSPLANE_TAG)|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(SED_CMD) 's|%%AGENT_TAG%%|$(AGENT_TAG)|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(SED_CMD) 's|%%XGQL_TAG%%|$(XGQL_TAG)|g' $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/values.yaml
	@$(OK) Generating values.yaml for the chart

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
	@$(SED_CMD) 's|RELEASE-NAME|$(PROJECT_NAME)|g' $(WORK_DIR)/olm.yaml
	@rm -rf $(OLM_DIR)/bundle
	@cat $(WORK_DIR)/olm.yaml | $(OLMBUNDLE) --version $(HELM_CHART_VERSION) --chart-file-path $(HELM_CHARTS_DIR)/$(PACKAGE_NAME)/Chart.yaml --extra-resources-dir $(CRDS_DIR) --output-dir $(OLM_DIR)

olm.artifacts: olm.build
	@mkdir -p $(abspath $(OUTPUT_DIR)/olm)
	@tar -czvf $(abspath $(OUTPUT_DIR)/olm/$(VERSION)).tar.gz  -C $(OLM_DIR)/bundle .

build.artifacts: olm.artifacts

generate.run: helm.prepare olm.build

local-dev: $(UP) local.up local.deploy.$(PACKAGE_NAME)

e2e.run: build local-dev local.deploy.validation
e2e.done: local.down

.PHONY: olm.build olm.artifacts crossplane submodules fallthrough
