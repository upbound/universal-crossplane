# Upbound Universal Crossplane (UXP)

<style>
    .numberCircle {
        border-radius: 50%;
        width: 22px;
        height: 22px;
        padding: 4px;
        background: #6553C0;
        border: 2px solid #6553C0;
        color: #fff;
        text-align: center;
        display: inline-block;
        margin-right: 10px;
        margin-bottom:5px;

        font: 20px Arial, sans-serif;
    }
</style>

<a href="https://upbound.io/uxp">
    <img align="right" style="margin-left: 20px" src="https://raw.githubusercontent.com/upbound/universal-crossplane/main/docs/media/logo.png?token=AAC27SIXWKA3P4XMLPDFQXDAUML7I" width=200 />
</a>

Upbound Universal Crossplane (UXP) is Upbound's official enterprise-grade distribution of Crossplane. It's fully compatible with downstream Crossplane, open source, capable of connecting to Upbound Cloud for real-time dashboard visibility, and maintained by Upbound. It's the easiest way for both individual community members and enterprises to start deploying control plane architectures to production.

## Quick Start


<div><div class="numberCircle">1</div>Install the Upbound CLI</div>

```
curl -sL https://cli.upbound.io | sh
```


<div><div class="numberCircle">2</div>Install UXP to a Kubernetes cluster (make sure to have Kubeconfig setup with your cluster info)</div>

```
up uxp install
```

<br/>

<div><div class="numberCircle">3</div><a href="https://cloud.upbound.io/register">Create an Upbound account</a> for a free real-time dashboard</div>

<br/>

<div><div class="numberCircle">4</div>Connect UXP to Upbound Cloud (go here for more details)</div>

```
up cloud login --profile=<CHOOSE PROFILE NAME> --account=<USERNAME> --username=<USERNAME> --password=<PASSWORD>
up cloud controlplane attach $NAME --profile=<YOUR PROFILE NAME> | up uxp connect -
```

<br/>

<div><div class="numberCircle">2</div>View your UXP instance live by <a href="https://cloud.upbound.io/login">signing into your Upbound account</a></div>


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