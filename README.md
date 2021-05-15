# Upbound Universal Crossplane (UXP)

<a href="https://upbound.io/uxp">
    <img align="right" style="margin-left: 20px" src="docs/media/logo.png" width=200 />
</a>

Upbound Universal Crossplane (UXP) is [Upbound's][upbound] official
enterprise-grade distribution of [Crossplane][crossplane]. It's fully compatible
with upstream Crossplane, open source, capable of connecting to Upbound Cloud
for real-time dashboard visibility, and maintained by Upbound. It's the easiest
way for both individual community members and enterprises to build their
production control planes.

## Quick Start

1. Install the [Upbound CLI][upbound-cli].

   ```console
   curl -sL https://cli.upbound.io | sh
   ```

2. Install UXP to a Kubernetes cluster.

   ```console
   # Make sure your ~/.kube/config file points to your cluster
   up uxp install
   ```

3. [Create an Upbound account][create-account] for a free dashboard for UXP.

4. Connect UXP to Upbound Cloud.

   ```console
   # The name of your new UXP control plane.
   UXP_NAME=mycrossplane

   # Your Upbound Cloud user.
   UP_USER=myuser

   # Write your Upbound Cloud password to a file.
   vim password

   cat password | up cloud login --username=${UP_USER} --password=-
   up cloud controlplane attach ${UXP_NAME} | up uxp connect -
   ```

5. Manage your UXP control plane by [signing in][login] to your Upbound account.

![UXP in Upbound Cloud](docs/media/uxp-in-ubc.png)

## Additional Resources

- The [UXP Documentation][uxp-documentation] provides additional information
  about UXP and resources for developers, like examples.
- The [developer guide][developer-guide] describes how to build and run UXP
  locally from source.
- [UXP Slack][uxp-slack] is where you can go to get all of your UXP questions
  answered.

[upbound]: https://upbound.io
[crossplane]: https://crossplane.io/
[upbound-cli]: https://github.com/upbound/up
[create-account]: https://cloud.upbound.io/register
[login]: https://cloud.upbound.io/login
[uxp-documentation]: https://cloud.upbound.io/uxp
[developer-guide]: docs/developer-guide.md
[uxp-slack]: https://crossplane.slack.com/archives/uxp
