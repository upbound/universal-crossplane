#!/usr/bin/env bash
set -aeuo pipefail

if [ "${LOCALDEV_UBC_PERMISSION}" != "edit" ] && [ "${LOCALDEV_UBC_PERMISSION}" != "view" ]; then
  echo "LOCALDEV_UBC_PERMISSION is neither edit nor view, skipping self hosted control plane creation"
  return 0
fi

login_ubc

echo "Checking if control plane ${LOCALDEV_CONNECT_CP_NAME} in org ${LOCALDEV_CONNECT_CP_ORG} already exists..."
cp_id=$(get_control_plane_id)

if [ -z "${cp_id}" ]; then
  echo "Creating a new control plane with name ${LOCALDEV_CONNECT_CP_NAME} in org ${LOCALDEV_CONNECT_CP_ORG}"
  cp_id=$(create_control_plane)
  echo "Platform created with id ${cp_id}!"
else
  echo "Platform exists with id ${cp_id}!"
fi



echo "Creating control plane token..."
cp_token=$(create_control_plane_token "${cp_id}")

if [ -z "${cp_token}" ]; then
  echo "Token creation failed, obtained token is empty"
  exit 1
fi
echo "Platform token created!"

echo "Creating control plane token secret..."
platform_token_secret="upbound-control-plane-token"
${KUBECTL} -n "${HELM_RELEASE_NAMESPACE}" delete secret "${platform_token_secret}" --ignore-not-found
${KUBECTL} -n "${HELM_RELEASE_NAMESPACE}" create secret generic "${platform_token_secret}" --from-literal token="${cp_token}"

echo "Success!"