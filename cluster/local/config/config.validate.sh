echo """
=============================================================
Deployment configuration:
LOCALDEV_CONNECT_TO_UBC: ${LOCALDEV_CONNECT_TO_UBC}
LOCALDEV_CONNECT_CP_ORG: ${LOCALDEV_CONNECT_CP_ORG}
LOCALDEV_CONNECT_CP_NAME: ${LOCALDEV_CONNECT_CP_NAME}
=============================================================
"""

if [ "${LOCALDEV_CONNECT_TO_UBC}" == "true" ]; then
  if [[ -z ${LOCALDEV_CONNECT_API_TOKEN:-} ]]; then
    echo_error "LOCALDEV_CONNECT_TO_UBC is set to true but LOCALDEV_CONNECT_API_TOKEN is not set ";
  fi
  if [[ -z ${LOCALDEV_CONNECT_CP_ORG:-} ]]; then
    echo_error "LOCALDEV_CONNECT_TO_UBC is set to true but LOCALDEV_CONNECT_CP_ORG is not set ";
  fi
  if [[ -z ${LOCALDEV_CONNECT_CP_NAME:-} ]]; then
    echo_error "LOCALDEV_CONNECT_TO_UBC is set to true but LOCALDEV_CONNECT_CP_NAME is not set ";
  fi
fi
