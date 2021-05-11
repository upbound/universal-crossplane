#!/usr/bin/env bash
set -aeuo pipefail

if [ "${LOCALDEV_UBC_PERMISSION}" != "edit" ] && [ "${LOCALDEV_UBC_PERMISSION}" != "view" ]; then
  echo "LOCALDEV_UBC_PERMISSION is neither edit nor view, skipping self hosted control plane creation"
  return 0
fi

echo "Logging in to Upbound Cloud..."
"${UP}" cloud login --profile=uxp-e2e -t "${LOCALDEV_CONNECT_API_TOKEN}" -a ${LOCALDEV_CONNECT_CP_ORG}

echo "Creating and connecting to self-hosted control plane..."
"${UP}" cloud xp attach --profile=uxp-e2e "${LOCALDEV_CONNECT_CP_NAME}" | ${UP} uxp connect -

echo "Success!"