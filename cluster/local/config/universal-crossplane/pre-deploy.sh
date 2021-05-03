#!/usr/bin/env bash
set -aeuo pipefail

UPBOUND_CROSSPLANE_NAMESPACE=${HELM_RELEASE_NAMESPACE}
API_TOKEN="${LOCALDEV_CONNECT_API_TOKEN}"
CONTROL_PLANE_NAME="${LOCALDEV_CONNECT_CP_NAME}"
CONTROL_PLANE_ORG="${LOCALDEV_CONNECT_CP_ORG}"
UPBOUND_PLATFORM_TOKEN_SECRET_NAME="upbound-control-plane-token"

if [ "${LOCALDEV_UBC_PERMISSION}" != "Edit" ] && [ "${LOCALDEV_UBC_PERMISSION}" != "View" ]; then
  echo "LOCALDEV_UBC_PERMISSION is neither Edit nor View, skipping self hosted control plane creation"
  return 0
fi

# TODO(hasan): replace below with up cli

_decode_base64_url() {
  local len=$((${#1} % 4))
  local result="$1"
  if [ $len -eq 2 ]; then result="$1"'=='
  elif [ $len -eq 3 ]; then result="$1"'='
  fi
  echo "$result" | tr '_-' '/+' | base64 -d
}

get_token_id() { _decode_base64_url $(echo -n $1 | cut -d "." -f ${2:-2}) | jq -r .jti; }

token_id=$(get_token_id "${API_TOKEN}")
kube_cluster_id=$(kubectl get ns kube-system -o jsonpath='{.metadata.uid}')

echo "Logging in with token id ${token_id}..."
curl --cookie-jar /tmp/req.cookie \
  -H "Content-Type: application/json" \
  -d '{"id": "'"${token_id}"'","password": "'"${API_TOKEN}"'","remember": false}' \
  "https://${UPBOUND_API_ENDPOINT}/v1/login"
echo "Logged in!"

echo "Checking if control plane ${CONTROL_PLANE_NAME} in org ${CONTROL_PLANE_ORG} already exists..."
cp_id=$(curl -s --cookie /tmp/req.cookie "https://${UPBOUND_API_ENDPOINT}/v1/namespaces/${CONTROL_PLANE_ORG}/controlPlanes" | \
  jq -c '.[] | select(.controlPlane.name | contains("'"${CONTROL_PLANE_NAME}"'"))' | \
  jq -r .controlPlane.id)

if [ -z "${cp_id}" ]; then
    echo "Creating a new control plane with name ${CONTROL_PLANE_NAME} in org ${CONTROL_PLANE_ORG}"
    cp_id=$(curl -s --cookie /tmp/req.cookie \
      -H "Content-Type: application/json" \
      -d '{"namespace": "'"${CONTROL_PLANE_ORG}"'","name": "'"${CONTROL_PLANE_NAME}"'","description": " ", "selfHosted": true, "kubeClusterID": "'"${kube_cluster_id}"'"}' \
      "https://${UPBOUND_API_ENDPOINT}/v1/controlPlanes" | \
      jq -r .controlPlane.id)
fi
echo "Platform created/exists with id ${cp_id}!"

echo "Creating control plane token..."
cp_token=$(curl -s --cookie /tmp/req.cookie \
  -H "Content-Type: application/json" \
  -d '{"data":{"type":"tokens","attributes":{"name":"a control plane token"},"relationships":{"owner":{"data":{"type":"controlPlanes","id":"'"${cp_id}"'"}}}}}' \
  "https://${UPBOUND_API_ENDPOINT}/v1/tokens" | \
  jq -r .data.meta.jwt)

if [ -z "${cp_token}" ]; then
  echo "Token creation failed, obtained token is empty"
  exit 1
fi
echo "Platform token created!"

echo "Creating control plane token secret..."
kubectl -n "${UPBOUND_CROSSPLANE_NAMESPACE}" delete secret "${UPBOUND_PLATFORM_TOKEN_SECRET_NAME}" --ignore-not-found
kubectl -n "${UPBOUND_CROSSPLANE_NAMESPACE}" create secret generic "${UPBOUND_PLATFORM_TOKEN_SECRET_NAME}" --from-literal token="${cp_token}"

echo "Success!"