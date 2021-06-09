# Operator Lifecycle Manager Artifact

This folder contains the OLM bundle to be published in OperatorHub.

> Note: We currently manually strip off the `runAsUser` and `runAsGroup` fields
> from the Crossplane and RBAC Manager `Deployments` to work around
> https://github.com/upbound/universal-crossplane/issues/116. This should be
> handled automatically in the future. 

Every PR that is merged and version that is tagged and promoted results in the
publishing of the OLM bundle to
https://releases.upbound.io/universal-crossplane. To publish a new version to
OperatorHub, download the subdirectory named `$VERSION` in the `olm` directory
and open two separate PRs to the following directories with it copied into the
specified path:
* https://github.com/operator-framework/community-operators/tree/master/community-operators/universal-crossplane
* https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators/universal-crossplane

The bundle output is specified by the contents of the helm chart and can be
modified by altering the following files. Any changes made to these files should
be committed to the repository before tag and release of a new version.
* `Chart.yaml` is used for metadata, so make sure it contains the new version.
* `annotations.yaml.tmpl` is used as base annotations, make sure channel
  information is correct for that release.
* `clusterserviceversion.yaml.tmpl` is used as base `ClusterServiceVersion` and
  it contains metadata that cannot be extracted from `Chart.yaml`, such as
  `installModes`. Make sure new version doesn't make any changes there.

> If you would like to connect to a different Upbound endpoint, you need to
> change the default one in the `values.yaml.tmpl` in Helm chart in
> `cluster/charts`.

After all is ready, run the following command to get the current bundle
generated:
```bash
make olm.build
```

> YAML of ClusterServiceVersion is not committed to git because the image tag in
> that file is generated using the hash of the last commit.

# Testing OLM Bundle

> Full instructions are at https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md

Install `kind`, `operator-sdk` and `opm` tools.

## Publish

Here is a set of commands you can use to build a bundle, add it to a custom
`CatalogSource` and test.

Specify your image prefix to point to your personal organization in the registry:
```bash
IMAGE_PREFIX="docker.io/upbound"
```

Building a bundle:
```bash
docker build cluster/olm/bundle -f cluster/olm/bundle/Dockerfile -t "${IMAGE_PREFIX}/uxp-test:0.0.1"
docker push "${IMAGE_PREFIX}/uxp-test:0.0.1"
```

Creating a `CatalogSource` including your bundle:
```bash
opm --build-tool docker index add --bundles "${IMAGE_PREFIX}/uxp-test:0.0.1" --tag "${IMAGE_PREFIX}/uxp-test-catalog:0.0.1"
docker push "${IMAGE_PREFIX}/uxp-test-catalog:0.0.1"
```

## Consume

Create a cluster and install OLM:
```bash
kind create cluster --wait 5m
operator-sdk olm install
```

Now install your custom `CatalogSource`:
```bash
sed "s|IMAGE_PREFIX|${IMAGE_PREFIX}|g" cluster/olm/test/catalogsource.yaml | kubectl apply -f -
```

Wait for it to become ready:
```bash
kubectl get catalogsource -n olm -o yaml -w
```

The cluster has a catalog that includes our OLM bundle. Now, we will prepare the
namespace we'll deploy our bundle to. Note that the namespace has to be `upbound-system`
and it needs to be configured with a global `OperatorGroup` so that it can watch
all namespaces.
```bash
kubectl apply -f cluster/olm/test/operatorgroup.yaml
```

Now create the `Subscription`:
```bash
kubectl apply -f cluster/olm/test/subscription.yaml
```

This will install your operator to `upbound-system` namespace. You can watch
`Subscription` and `ClusterServiceVersion` CRs in that namespace.