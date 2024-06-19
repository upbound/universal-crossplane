# Copyright 2023 The Upbound Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ifeq ($(HELM_CHARTS),)
$(error the variable HELM_CHARTS must be set prior to including helmoci.mk)
endif

# the base url where helm charts are pushed
ifeq ($(HELM_OCI_URL),)
$(error the variable HELM_OCI_URL must be set prior to including helmoci.mk)
endif

# the charts directory
HELM_CHARTS_DIR ?= $(ROOT_DIR)/cluster/charts

# the charts output directory
HELM_OUTPUT_DIR ?= $(OUTPUT_DIR)/charts

HELM_CHART_LINT_STRICT ?= true
ifeq ($(HELM_CHART_LINT_STRICT),true)
HELM_CHART_LINT_STRICT_ARG += --strict
endif

# helm home
HELM_HOME := $(abspath $(WORK_DIR)/helm)
export HELM_HOME

HELM_CACHE_HOME = $(HELM_HOME)/cache
HELM_CONFIG_HOME = $(HELM_HOME)/config
HELM_DATA_HOME = $(HELM_HOME)/data
export HELM_CACHE_HOME
export HELM_CONFIG_HOME
export HELM_DATA_HOME

# remove the leading `v` for helm chart versions
HELM_CHART_VERSION := $(VERSION:v%=%)

# ====================================================================================
# Helm Targets
$(HELM_HOME): $(HELM)
	@mkdir -p $(HELM_HOME)

$(HELM_OUTPUT_DIR):
	@mkdir -p $(HELM_OUTPUT_DIR)

define helm.chart
$(HELM_OUTPUT_DIR)/$(1)-$(HELM_CHART_VERSION).tgz: $(HELM_HOME) $(HELM_OUTPUT_DIR) $(shell find $(HELM_CHARTS_DIR)/$(1) -type f)
	@$(INFO) helm package $(1) $(HELM_CHART_VERSION)
	$(HELM) package --version $(HELM_CHART_VERSION) --app-version $(HELM_CHART_VERSION) -d $(HELM_OUTPUT_DIR) $(abspath $(HELM_CHARTS_DIR)/$(1))
	@$(OK) helm package $(1) $(HELM_CHART_VERSION)

helm.lint.$(1): $(HELM_HOME)
	@rm -rf $(abspath $(HELM_CHARTS_DIR)/$(1)/charts)
	@$(HELM) dependency update $(abspath $(HELM_CHARTS_DIR)/$(1))
	@$(HELM) lint $(abspath $(HELM_CHARTS_DIR)/$(1)) $(HELM_CHART_LINT_ARGS_$(1)) $(HELM_CHART_LINT_STRICT_ARG)

helm.lint: helm.lint.$(1)

helm.dep.$(1): $(HELM_HOME)
	@$(INFO) helm dep $(1) $(HELM_CHART_VERSION)
	@$(HELM) dependency update $(abspath $(HELM_CHARTS_DIR)/$(1))
	@$(OK) helm dep $(1) $(HELM_CHART_VERSION)

helm.dep: helm.dep.$(1)

helm.build.$(1): $(HELM_OUTPUT_DIR)/$(1)-$(HELM_CHART_VERSION).tgz

helm.build: helm.build.$(1)
endef
$(foreach p,$(HELM_CHARTS),$(eval $(call helm.chart,$(p))))

helm.clean:
	@rm -fr $(HELM_OUTPUT_DIR)

helm.env: $(HELM)
	@$(HELM) env

# ====================================================================================
# helm push to OCI registry
define oci.push
helm.push.$(1): $(HELM_HOME)
	@$(INFO) pushing helm chart $(1) to OCI registry $(HELM_OCI_URL)
	@$(HELM) push $(HELM_OUTPUT_DIR)/$(1)-$(HELM_CHART_VERSION).tgz oci://$(HELM_OCI_URL)
	@$(OK) pushing helm charts to OCI registry

helm.push: helm.push.$(1)
endef
$(foreach p,$(HELM_CHARTS),$(eval $(call oci.push,$(p))))
# ====================================================================================
# Common Targets

build.init: helm.lint
build.check: helm.dep
build.artifacts: helm.build
clean: helm.clean
lint: helm.lint
publish.artifacts: helm.push
