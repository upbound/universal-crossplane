#!/usr/bin/env bash
set -aeuo pipefail

UPBOUND_CROSSPLANE_NAMESPACE=${HELM_RELEASE_NAMESPACE}
API_TOKEN="${LOCALDEV_CONNECT_API_TOKEN}"
CONTROL_PLANE_NAME="${LOCALDEV_CONNECT_CP_NAME}"
CONTROL_PLANE_ORG="${LOCALDEV_CONNECT_CP_ORG}"
UPBOUND_PLATFORM_TOKEN_SECRET_NAME="control-plane-token-secret"

if [ ${LOCALDEV_CONNECT_TO_UBC} != "true" ]; then
  echo "LOCALDEV_CONNECT_TO_UBC is not set to true, skipping self hosted platform creation"
  return 0
fi

# TODO(hasan): replace below with up cli

_decode_base64_url() {
  local len=$((${#1} % 4))
  local result="$1"
  if [ $len -eq 2 ]; then result="$1"'=='
  elif [ $len -eq 3 ]; then result="$1"'='
  fi
  echo "$result" | tr '_-' '/+' | base64 -D
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

echo "Checking if platform ${CONTROL_PLANE_NAME} in org ${CONTROL_PLANE_ORG} already exists..."
platform_id=$(curl -s --cookie /tmp/req.cookie "https://${UPBOUND_API_ENDPOINT}/v1/namespaces/${CONTROL_PLANE_ORG}/platforms" | \
  jq -c '.platforms[] | select(.platform.name | contains("'"${CONTROL_PLANE_NAME}"'"))' | \
  jq -r .platform.id)

if [ -z "${platform_id}" ]; then
    echo "Creating a new platform with name ${CONTROL_PLANE_NAME} in org ${CONTROL_PLANE_ORG}"
    platform_id=$(curl -s --cookie /tmp/req.cookie \
      -H "Content-Type: application/json" \
      -d '{"namespace": "'"${CONTROL_PLANE_ORG}"'","name": "'"${CONTROL_PLANE_NAME}"'","description": " ", "selfHosted": true, "kubeClusterID": "'"${kube_cluster_id}"'"}' \
      "https://${UPBOUND_API_ENDPOINT}/v1/platforms" | \
      jq -r .platform.platform.id)
fi
echo "Platform created/exists with id ${platform_id}!"

echo "Creating platform token..."
platform_token=$(curl -s --cookie /tmp/req.cookie \
  -H "Content-Type: application/json" \
  -d '{"data":{"type":"tokens","attributes":{"name":"a platform token"},"relationships":{"owner":{"data":{"type":"platforms","id":"'"${platform_id}"'"}}}}}' \
  "https://${UPBOUND_API_ENDPOINT}/v1/tokens" | \
  jq -r .data.meta.jwt)

if [ -z "${platform_token}" ]; then
  echo "Token creation failed, obtained token is empty"
  exit 1
fi
echo "Platform token created!"

echo "Creating platform token secret..."
kubectl -n "${UPBOUND_CROSSPLANE_NAMESPACE}" delete secret "${UPBOUND_PLATFORM_TOKEN_SECRET_NAME}" --ignore-not-found
kubectl -n "${UPBOUND_CROSSPLANE_NAMESPACE}" create secret generic "${UPBOUND_PLATFORM_TOKEN_SECRET_NAME}" --from-literal token="${platform_token}"

echo "Success!"