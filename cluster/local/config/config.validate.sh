echo """
=============================================================
Deployment configuration:
LOCALDEV_UBC_PERMISSION: ${LOCALDEV_UBC_PERMISSION}
LOCALDEV_CONNECT_CP_ORG: ${LOCALDEV_CONNECT_CP_ORG}
LOCALDEV_CONNECT_CP_NAME: ${LOCALDEV_CONNECT_CP_NAME}
LOCALDEV_CONNECT_CLEANUP: ${LOCALDEV_CONNECT_CLEANUP}
=============================================================
"""

if [ -n "${LOCALDEV_CONNECT_API_TOKEN}" ]; then
  if [[ -z ${LOCALDEV_CONNECT_CP_ORG:-} ]]; then
    echo_error "LOCALDEV_CONNECT_API_TOKEN is set but LOCALDEV_CONNECT_CP_ORG is not set ";
  fi
  if [[ -z ${LOCALDEV_CONNECT_CP_NAME:-} ]]; then
    echo_error "LOCALDEV_CONNECT_API_TOKEN is set but LOCALDEV_CONNECT_CP_NAME is not set ";
  fi
fi
