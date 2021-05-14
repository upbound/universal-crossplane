# Upbound Universal Crossplane (UXP)

<a href="https://upbound.io/uxp">
    <img align="right" style="margin-left: 20px" src="https://raw.githubusercontent.com/upbound/universal-crossplane/main/docs/media/logo.png?token=AAC27SIXWKA3P4XMLPDFQXDAUML7I" width=200 />
</a>

Upbound Universal Crossplane (UXP) is Upbound's official enterprise-grade distribution of Crossplane. It's fully compatible with downstream Crossplane, open source, capable of connecting to Upbound Cloud for real-time dashboard visibility, and maintained by Upbound. It's the easiest way for both individual community members and enterprises to start deploying control plane architectures to production.

## Quick Start


1. Install the Upbound CLI
    ```
    curl -sL https://cli.upbound.io | sh
    ```

2. Install UXP to a Kubernetes cluster (make sure to have Kubeconfig setup with your cluster info)
    ```
    up uxp install
    ```


3. <a href="https://cloud.upbound.io/register">Create an Upbound account</a> for a free real-time dashboard for UXP

4. Connect UXP to Upbound Cloud (go here for more details)
    ```
    up cloud login --profile=<CHOOSE PROFILE NAME> --account=<USERNAME> --username=<USERNAME> --password=<PASSWORD>
    up cloud controlplane attach $NAME --profile=<YOUR PROFILE NAME> | up uxp connect -
    ```

5. View your UXP instance live by <a href="https://cloud.upbound.io/login">signing into your Upbound account</a>


< SCREENSHOT GOES HERE >

## Additional Resources
* [UXP Documentation](https://cloud.upbound.io/uxp) provides additional information about UXP and resources for developers like examples
* [Developer Guide](docs/developer-guide.md) describes how to build and run UXP locally from source
* [Community Slack](https://slack.crossplane.io) is where you can go to get all of your UXP questions answered
