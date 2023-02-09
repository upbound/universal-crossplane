WAIT_DEPLOYMENT="${KUBECTL} wait deployment -n ${HELM_RELEASE_NAMESPACE} --for=condition=available --timeout 60s"

echo_info "Verify all deployments are ready!"
${WAIT_DEPLOYMENT} crossplane
${WAIT_DEPLOYMENT} crossplane-rbac-manager
echo_info "Successfully validated all deployments!"
