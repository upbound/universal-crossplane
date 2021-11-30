# Contributing

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

## Release Process

### Update Crossplane Version (skip if not necessary)

UXP bundles [Upbound maintained version of Crossplane](https://github.com/upbound/crossplane) 
which might contain **additional** features and bug fixes on top of 
corresponding upstream Crossplane version. For example, **Upbound Crossplane**
`v1.4.0-up.1` should include everything in **Crossplane** `v1.4.0` but _might_
include additional changes. All additional changes that could be included should
have been merged into master of Upstream Crossplane. This is to prevent 
diverging from Upstream Crossplane project, but we would still be able to ship 
fixes and features early, independent of Upstream Crossplane release cadence.

To update Crossplane version in UXP:

1. Prepare corresponding release branch in [Upbound Crossplane](https://github.com/upbound/crossplane)
   1. Make sure to include all changes in Upstream Crossplane version. 
   For example, if we are planning to tag `v1.4.1-up.x`, `release-1.4` branch
   should include everything in upstream Crossplane `v1.4.1`.
   2. If additional features and/or fixes to be included, cherry-pick them into
   the release branch.
2. Run the [Tag action](https://github.com/upbound/crossplane/actions/workflows/tag.yml)
in Upbound Crossplane by following the [versioning schema](VERSIONING.md). 
Please note, for both UXP and Upbound Crossplane, we use the same versioning 
schema.
3. Run the [CI action](https://github.com/upbound/crossplane/actions/workflows/ci.yml)
in Upbound Crossplane for the release branch.
4. Update the `CROSSPLANE_TAG` and `CROSSPLANE_COMMIT` in the UXP[Makefile](Makefile).
At this point you should be able to pull the following docker image:
`docker pull upbound/crossplane:[CROSSPLANE_TAG]`.

Please note our build module [converts the latter dash to dot](https://github.com/upbound/build/pull/155)
to make the version [sortable as semver](https://github.com/upbound/universal-crossplane/issues/109).
This causes [Upbound Crossplane](https://github.com/upbound/crossplane) to
produce a docker image with tag `v1.5.0-rc.0.up.1` for git tag 
`v1.5.0-rc.0-up.1`.

### Cut UXP Release

A UXP release is cut by following the steps below:

1. **branch repo**: Create a new release branch using the GitHub UI for the
   repo.
1. **release branch prep**: Make any release-specific updates on the release
   branch. The branch name should be `release-<major version>.<minor version>`,
   like `release-1.3`
1. **tag release**: Run the `Tag` action on the _release branch_ with the
   desired version (e.g. `v0.14.0`).
1. **build/publish**: Run the `CI` action on the release branch with the version
   that was just tagged.
1. **tag next pre-release**: Run the `tag` action on the main development branch
   with the `rc.0` for the next release (e.g. `v1.3.0-up.1-rc.1`).
1. **verify**: Verify all artifacts have been published successfully, perform
   sanity testing.
1. **promote**: Run the `Promote` action to promote release to desired
   channel(s).
1. **release notes**: Publish well authored and complete release notes on
   GitHub.
1. **announce**: Announce the release on Twitter, Slack, etc.

After a release is cut, we need to publish it to the following distribution channels.

### Operator Lifecycle Manager

OLM uses its own packaging format for its marketplaces.

1. **download bundle**: Download the bundle from the following link:
   ```
   VERSION=<release version>
   curl -sL https://releases.upbound.io/universal-crossplane/stable/$VERSION/olm/$VERSION.tar.gz  | tar xz
   ```
1. **open pull requests**: Extract the content into a new folder named `$VERSION`
   and open the PRs to the following directories:
    * https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main/operators/universal-crossplane
    * https://github.com/k8s-operatorhub/community-operators/tree/main/operators/universal-crossplane
1. **sanity check**: Once the PRs are merged and CI in main branch completes the
   propagation, you should see the new version in [OperatorHub product page](https://operatorhub.io/operator/universal-crossplane).

See [this page](cluster/olm/README.md) for details.

### AWS Marketplace

AWS Marketplace installations are done using usual Helm commands but the images
need to exist in given ECR repositories. Currently, due to technical limitations,
it is not possible to define IAM policies on the allocated marketplace ECR repository
so you need to push the images there manually.

1. **list images**: In order to see what images are used in a specific release,
   you can examine the Helm chart `values.yaml` file:
   ```
   VERSION=<released-version>
   curl -sL https://charts.upbound.io/stable/universal-crossplane-$VERSION.tgz | tar xz
   ```
1. **login to ECR**: You need to have an AWS user in production marketplace account
   and use the following command to login your docker client:
   ```
   MARKETPLACE_ACCOUNT_ID=123456789
   aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin "${MARKETPLACE_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com"
   ```
1. **tag & push**: Tag and push every image like the following:
   ```
   # You can check the full repository URLs from Repositories list of the product
   # in AWS Marketplace Management Portal
   docker tag upbound/crossplane:<tag-in-values.yaml> "709825985650.dkr.ecr.us-east-1.amazonaws.com/upbound/crossplane:<tag-in-values.yaml>"
   ```
   Note that all images, including the ones under crossplane DockerHub organization,
   need to be re-tagged pushed to ECR.
1. **add a new version**: Login to [AWS Marketplace](https://aws.amazon.com/marketplace/)
   and add a new version using `Request changes` menu in the UI. It will
   pre-populate most of the metadata from the last release, make sure all
   versions are updated.
1. **release version**: Using the same `Request changes`, release the version you
   created.
1. **sanity check**: After change requests are completed, visit [the product page](https://aws.amazon.com/marketplace/pp/prodview-uhc2iwi5xysoc)
   to make sure the new version is there.

### Rancher Marketplace

UXP is available as a partner chart in the Rancher Marketplace. Rancher partner charts are served from the
Rancher's [partner-charts repository](https://github.com/rancher/partner-charts). To get UXP chart updated,
we need to open a PR against the [main-source branch](https://github.com/rancher/partner-charts/tree/main-source) there.

To prepare the PR, we need to follow the workflow steps listed [here](https://github.com/rancher/partner-charts/tree/main-source#workflow).
Due to the reasons outlined in [this issue](https://github.com/upbound/universal-crossplane/issues/119), we need an
additional change in `Chart.yaml` where we convert UXP version from `x.y.z-up.t` to `x.y.z00t` in the [make changes step](https://github.com/rancher/partner-charts/tree/main-source#4-make-changes).
See [this](https://github.com/rancher/partner-charts/pull/89#discussion_r640533267) as an example.
