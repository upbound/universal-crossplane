# Operator Lifecycle Manager Artifact

This folder contains the OLM bundle to be published in OperatorHub.

What you need to do when you'd like to release a new version is the following:
* `Chart.yaml` is used for metadata, so make sure it contains the new version.
* `annotations.yaml.tmpl` is used as base annotations, make sure channel information
  is correct for that release.
* `clusterserviceversion.yaml.tmpl` is used as base `ClusterServiceVersion` and it
  contains metadata that cannot be extracted from `Chart.yaml`, such as `installModes`.
  Make sure new version doesn't make any changes there.

> If you would like to connect to a different Upbound endpoint, you need to change
> the default one in the `values.yaml.tmpl` in Helm chart in `cluster/charts`.

After all is ready, run the following command to get the current bundle generated:
```bash
make olm
```

> YAML of ClusterServiceVersion is not committed to git because the image tag
> in that file is generated using the hash of the last commit.

A new folder will be created here named with the version number. After making sure
it all looks good, open PRs to the following targets to publish it:
* https://github.com/operator-framework/community-operators/tree/master/community-operators
* https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators


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
cd <version directory>
docker build -t "${IMAGE_PREFIX}/uxp-test:0.0.1" .
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
```
cd <root of the repo>
sed "s|IMAGE_PREFIX|${IMAGE_PREFIX}|g" cluster/olm/test/catalogsource.yaml | kubectl apply -f -
```

Wait for it to become ready:
```
kubectl get catalogsource -n olm -o yaml -w
```

The cluster has a catalog that includes our OLM bundle. Now, we will prepare the
namespace we'll deploy our bundle to. Note that the namespace has to be `upbound-system`
and it needs to be configured with a global `OperatorGroup` so that it can watch
all namespaces.
```
kubectl apply -f cluster/olm/test/operatorgroup.yaml
```

Now create the `Subscription`:
```
kubectl apply -f cluster/olm/test/subscription.yaml
```

This will install your operator to `operators` namespace. You can watch `Subscription`
and `InstallPlan` CRs in that namespace.