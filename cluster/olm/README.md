# Operator Lifecycle Manager Artifact

This folder contains the OLM bundle to be published in OperatorHub.

What you need to do when you'd like to release a new version is the following:
* `Chart.yaml` is used for metadata, so make sure it contains the new version.
* `annotations.yaml.tmpl` is used as base annotations, make sure channel information
  is correct for that release.
* `clusterserviceversion.yaml.tmpl` is used as base `ClusterServiceVersion` and it
  contains metadata that cannot be extracted from `Chart.yaml`, such as `installModes`.
  Make sure new version doesn't make any changes there.

After all is ready, run the following command:
```bash
make olm
```

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

Specify your image prefix:
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
sed "s/IMAGE_PREFIX/${IMAGE_PREFIX}/g" cluster/olm/test/catalogsource.yaml | kubectl create -f -
```

Wait for it to become ready:
```
kubectl get catalogsource -n olm -w
```

Now create the `Subscription`:
```
kubectl create -f cluster/olm/test/subscription.yaml
```

This will install your operator to `operators` namespace. You can watch `Subscription`
and `InstallPlan` CRs in that namespace.