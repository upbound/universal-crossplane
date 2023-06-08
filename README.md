# Upbound Universal Crossplane (UXP)

<a href="https://upbound.io/uxp">
    <img align="right" style="margin-left: 20px" src="docs/media/logo.png" width=200 />
</a>

Upbound Universal Crossplane (UXP) is [Upbound's][upbound] official
enterprise-grade distribution of [Crossplane][crossplane]. It's fully compatible
with upstream Crossplane, open source, and maintained by Upbound. It's the easiest
way for both individual community members and enterprises to build their
production control planes.

## Quick Start

1. Install the [Upbound CLI][upbound-cli].

   ```console
   curl -sL https://cli.upbound.io | sh
   ```
   
    To install with Homebrew:
    ```console
    brew install upbound/tap/up
    ```

2. Install UXP to a Kubernetes cluster.

   ```console
   # Make sure your ~/.kube/config file points to your cluster
   up uxp install
   ```

### Installation With Helm 3

Helm requires the use of `--devel` flag for versions with suffixes, like
`v1.2.1-up.3`. But Helm repository we use is the stable repository so use of that
flag is only a workaround, you will always get the latest stable version of UXP.

1. Create the namespace to install UXP.

   ```console
   kubectl create namespace upbound-system
   ```

1. Add `upbound-stable` chart repository.

   ```console
   helm repo add upbound-stable https://charts.upbound.io/stable && helm repo update
   ```

1. Install the latest stable version of UXP.

   ```console
   helm install uxp --namespace upbound-system upbound-stable/universal-crossplane --devel
   ```

### Upgrade from upstream Crossplane

In order to upgrade from upstream Crossplane, the target UXP version has to match
the Crossplane version until the `-up.N` suffix. For example, you can upgrade from
Crossplane `v1.2.1` only to a UXP version that looks like `v1.2.1-up.N` but not to
a `v1.3.0-up.N`. It'd need to be upgraded to upstream Crossplane `v1.3.0` and then
UXP `v1.3.0-up.N`.

#### Using up CLI

   ```console
   # Assuming it is installed in "crossplane-system" with release name "crossplane".
   up uxp upgrade -n crossplane-system
   ```

If you'd like to upgrade to a specific version, run the following:

   ```console
   # Assuming it is installed in "crossplane-system" with release name "crossplane".
   up uxp upgrade vX.Y.Z-up.N -n crossplane-system
   ```

#### Using Helm 3

   ```console
   # Assuming it is installed in "crossplane-system" with release name "crossplane".
   helm upgrade crossplane --namespace crossplane-system upbound-stable/universal-crossplane --devel
   ```

If you'd like to upgrade to a specific version, run the following:

   ```console
   # Assuming it is installed in "crossplane-system" with release name "crossplane".
   helm upgrade crossplane --namespace crossplane-system upbound-stable/universal-crossplane --devel --version vX.Y.Z-up.N
   ```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Releases

After each minor Crossplane release, a corresponding patched and hardened
version of Universal Crossplane will be released after 2 weeks at the latest.

After the minor release of UXP, we will update that version with UXP-specific
patches by incrementing `-up.X` suffix as well as upstream patches by incrementing
the patch version to the corresponding number.

An example timeframe would be like the following:
* Crossplane `v1.5.0` is released.
* 2 weeks bake period.
* The latest version in `release-1.5` is now `v1.5.2`
* The first release of UXP for v1.5 would be `v1.5.2-up.1`.
  * We take the latest patched version at the end of 2 weeks, not `v1.5.0-up.1`
    for example, if there is a patch release.
* Crossplane `v1.5.3` is released after the initial 2 weeks bake period.
* UXP `v1.5.3-up.1` will be released immediately to accommodate the fix coming
  with the patch version.

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
[uxp-documentation]: https://cloud.upbound.io/docs/uxp
[developer-guide]: developer-guide.md
[uxp-slack]: https://crossplane.slack.com/archives/upbound
