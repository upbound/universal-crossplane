scriptdir="$( dirname "${BASH_SOURCE[0]}")"

echo "Running validation..."

login_ubc
CONTROL_PLANE_ID=$(get_control_plane_id)
export CONTROL_PLANE_ID

source "${scriptdir}/validate.sh"

if [ "${LOCALDEV_CONNECT_CLEANUP}" == "true" ]; then
  echo "Deleting self hosted control plane with id ${CONTROL_PLANE_ID}"
  delete_control_plane "${CONTROL_PLANE_ID}"
fi
