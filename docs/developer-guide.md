## Developer Guide

### Local Development

To spin up a local development environment with locally built artifacts, run:

```
make build local-dev
```

You can override default local development configuration by overriding environment
variables [here](https://github.com/upbound/universal-crossplane/blob/main/cluster/local/config/config.env).

For example, the following will enable connecting your local development environment to Upbound Cloud:

```
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

To run validation test including Upbound Cloud connectivity, run:

```
export LOCALDEV_CONNECT_CP_ORG=<YOUR_UBC_ORG>
export LOCALDEV_CONNECT_API_TOKEN=<YOUR_ACCESS_TOKEN>

make e2e.run
```