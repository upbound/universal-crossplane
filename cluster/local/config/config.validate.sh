echo """
=============================================================
Deployment configuration:
LOCALDEV_UBC_PERMISSION: ${LOCALDEV_UBC_PERMISSION}
LOCALDEV_CONNECT_CP_ORG: ${LOCALDEV_CONNECT_CP_ORG}
LOCALDEV_CONNECT_CP_NAME: ${LOCALDEV_CONNECT_CP_NAME}
LOCALDEV_CONNECT_CLEANUP: ${LOCALDEV_CONNECT_CLEANUP}
=============================================================
"""

if [ "${LOCALDEV_UBC_PERMISSION}" == "edit" ] || [ "${LOCALDEV_UBC_PERMISSION}" == "view" ]; then
  if [[ -z ${LOCALDEV_CONNECT_API_TOKEN:-} ]]; then
    echo_error "LOCALDEV_UBC_PERMISSION is set to ${LOCALDEV_UBC_PERMISSION} but LOCALDEV_CONNECT_API_TOKEN is not set ";
  fi
  if [[ -z ${LOCALDEV_CONNECT_CP_ORG:-} ]]; then
    echo_error "LOCALDEV_UBC_PERMISSION is set to ${LOCALDEV_UBC_PERMISSION} but LOCALDEV_CONNECT_CP_ORG is not set ";
  fi
  if [[ -z ${LOCALDEV_CONNECT_CP_NAME:-} ]]; then
    echo_error "LOCALDEV_UBC_PERMISSION is set to ${LOCALDEV_UBC_PERMISSION} but LOCALDEV_CONNECT_CP_NAME is not set ";
  fi
fi
