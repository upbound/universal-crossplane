scriptdir="$( dirname "${BASH_SOURCE[0]}")"

echo "Running validation..."

if [ -n "${LOCALDEV_CONNECT_API_TOKEN}" ]; then
  CONTROL_PLANE_ID=$("${UP}" cloud xp list --profile=uxp-e2e | awk '$1 == "'"${LOCALDEV_CONNECT_CP_NAME}"'" {print $2}')
  export CONTROL_PLANE_ID
fi

source "${scriptdir}/validate.sh"

if [ "${LOCALDEV_CONNECT_CLEANUP}" == "true" ]; then
  echo "Deleting self hosted control plane with id ${CONTROL_PLANE_ID}"
  "${UP}" cloud xp delete --profile=uxp-e2e "${CONTROL_PLANE_ID}"
fi
