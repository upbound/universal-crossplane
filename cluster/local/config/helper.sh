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

login_ubc() {
  token_id=$(get_token_id "${LOCALDEV_CONNECT_API_TOKEN}")

  echo "Logging in with token id ${token_id}..."
  curl --cookie-jar /tmp/req.cookie \
    -H "Content-Type: application/json" \
    -d '{"id": "'"${token_id}"'","password": "'"${LOCALDEV_CONNECT_API_TOKEN}"'","remember": false}' \
    "https://${UPBOUND_API_ENDPOINT}/v1/login"
  echo "Logged in!"
}

get_control_plane_id() {
  id=$(curl -s --cookie /tmp/req.cookie "https://${UPBOUND_API_ENDPOINT}/v1/accounts/${LOCALDEV_CONNECT_CP_ORG}/controlPlanes" | \
    jq -c '.[] | select(.controlPlane.name | contains("'"${LOCALDEV_CONNECT_CP_NAME}"'"))' | \
    jq -r .controlPlane.id)
  echo "${id}"
}

create_control_plane() {
  kube_cluster_id=$(${KUBECTL} get ns kube-system -o jsonpath='{.metadata.uid}')
  id=$(curl -s --cookie /tmp/req.cookie \
      -H "Content-Type: application/json" \
      -d '{"namespace": "'"${LOCALDEV_CONNECT_CP_ORG}"'","name": "'"${LOCALDEV_CONNECT_CP_NAME}"'","description": " ", "selfHosted": true, "kubeClusterID": "'"${kube_cluster_id}"'"}' \
      "https://${UPBOUND_API_ENDPOINT}/v1/controlPlanes" | \
      jq -r .controlPlane.id)
  echo "${id}"
}

create_control_plane_token(){
  cp_id=$1
  token=$(curl -s --cookie /tmp/req.cookie \
    -H "Content-Type: application/json" \
    -d '{"data":{"type":"tokens","attributes":{"name":"a control plane token"},"relationships":{"owner":{"data":{"type":"controlPlanes","id":"'"${cp_id}"'"}}}}}' \
    "https://${UPBOUND_API_ENDPOINT}/v1/tokens" | \
    jq -r .data.meta.jwt)
  echo "${token}"
}

delete_control_plane(){
  id=$1
  curl -s --cookie /tmp/req.cookie \
    -X DELETE \
    "https://${UPBOUND_API_ENDPOINT}/v1/controlPlanes/${id}"
}