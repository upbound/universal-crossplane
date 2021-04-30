# Upbound Distribution of Crossplane

Upbound releases a distribution of Crossplane that can connect to Upbound Cloud
and let users manage their Crossplane resources through the UI.

This repository contains the Helm chart and necessary manifests to deploy all
components of Upbound Distribution.

## Local Development

To spin up a local development environment with locally built artifacts, run:

```
make build local-dev
```

You can override default local development configuration by overriding environment
values [here](https://github.com/upbound/universal-crossplane/blob/main/cluster/local/config/config.env).

For example, following will enable connecting your local development environment to Upbound Cloud:

```
export LOCALDEV_CONNECT_TO_UBC=true
export LOCALDEV_CONNECT_CP_ORG=<YOUR_UBC_ORG>
export LOCALDEV_CONNECT_API_TOKEN=<YOUR_ACCESS_TOKEN>

make build local-dev
```

### Cleanup

To clean up local dev environment, first delete self hosted control plane (if connected) from Upbound Cloud Console
and then run:

```
make local.down
```

### Validation

To run validation tests locally, run:

```
make e2e.run
```