# Upbound Universal Crossplane (UXP)

Upbound Universal Crossplane (UXP) is Upbound's official enterprise-grade distribution of Crossplane. It's fully compatible with downstream Crossplane, open source, capable of connecting to Upbound Cloud for real-time dashboard visibility, and maintained by Upbound. It's the easiest way for both individual community members and enterprises to start deploying control plane architectures to production.

## Quick Start
#### Install the Upbound CLI
`curl -sL https://cli.upbound.io | sh`
#### Install UXP to a Kubernetes cluster (make sure to have Kubeconfig setup with your cluster info)
`up uxp install`
#### [Create an Upbound account](https://cloud.upbound.io/register) for a free real-time dashboard
#### Connect UXP to Upbound Cloud (go here for more details)
```
up cloud login --profile=<CHOOSE PROFILE NAME> --account=<USERNAME> --username=<USERNAME> --password=<PASSWORD>
up cloud controlplane attach $NAME --profile=<YOUR PROFILE NAME> | up uxp connect -
```

#### View your UXP instance live by [signing into your Upbound account](https://cloud.upbound.io/login)

< SCREENSHOT GOES HERE >

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
export LOCALDEV_UBC_PERMISSION=edit
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
