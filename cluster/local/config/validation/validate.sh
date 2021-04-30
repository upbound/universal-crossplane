#!/usr/bin/env bash
set -aeuo pipefail

WAIT_DEPLOYMENT="${KUBECTL} wait deployment -n ${HELM_RELEASE_NAMESPACE} --for=condition=available --timeout 60s"

echo_info "Verify all deployments are ready!"
${WAIT_DEPLOYMENT} crossplane
${WAIT_DEPLOYMENT} crossplane-rbac-manager
${WAIT_DEPLOYMENT} upbound-bootstrapper
${WAIT_DEPLOYMENT} crossplane-graphql
${WAIT_DEPLOYMENT} upbound-agent
${WAIT_DEPLOYMENT} xgql
echo_info "Successfully validated all deployments!"

if [ ${LOCALDEV_CONNECT_TO_UBC} != "true" ]; then
  echo_info "LOCALDEV_CONNECT_TO_UBC is not set to true, skipping validating Upbound Cloud connectivity"
  return 0
fi

echo_info "Validating connectivity to Upbound Cloud..."
cp_kubeconfig=/tmp/cp_kubeconfig
touch "${cp_kubeconfig}"
CP_KUBECTL="${KUBECTL} --kubeconfig ${cp_kubeconfig}"
${CP_KUBECTL} config set-cluster "self-hosted-test" --server="https://${UPBOUND_PROXY_ENDPOINT}/env/${CONTROL_PLANE_ID}"
${CP_KUBECTL} config set-credentials "crossplane" --token="${LOCALDEV_CONNECT_API_TOKEN}"
${CP_KUBECTL} config set-context "self-hosted-test" --cluster="self-hosted-test" --user="crossplane"
${CP_KUBECTL} config use-context "self-hosted-test"

echo_info "Validating \"kubectl\" queries work over Upbound Cloud..."
${CP_KUBECTL} get ns
validation_namespace="uxp-validation"
${CP_KUBECTL} delete ns "${validation_namespace}" --ignore-not-found
${CP_KUBECTL} create ns "${validation_namespace}"

${KUBECTL} get ns "${validation_namespace}"
echo_info "Successfully validated \"kubectl\" queries over Upbound Cloud!"

echo_info "Validating \"xqgl\" queries work over Upbound Cloud..."
# shellcheck disable=SC2089
query='query {
  kubernetesResources(
    apiVersion: \"v1\"
    kind: \"Namespace\"
  ) {
    totalCount
    nodes {
      metadata {
        name
      }
    }
  }
}'
# shellcheck disable=SC2090,SC2116,SC2086
query="$(echo $query)"   # the query should be oneliner, without newlines
xgql_response=$(curl -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${LOCALDEV_CONNECT_API_TOKEN}" \
  -X POST -d "{ \"query\": \"$query\"}" \
  "https://${UPBOUND_PROXY_ENDPOINT}/env/${CONTROL_PLANE_ID}"/query)

echo_info "XGQL response:"
echo "${xgql_response}" | json_pp

echo_info "Checking if xgql response contains validation namespace..."
echo "${xgql_response}" | grep -o "\"name\":\"${validation_namespace}\""
echo_info "Successfully validated \"xgql\" queries over Upbound Cloud!"


echo_info "Successfully validated UXP connectivity!"