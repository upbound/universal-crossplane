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

### Crossplane Fork Synchronization

This document describes how to synchronize a Crossplane fork with the upstream
repository. The synchronization process is automated using a GitHub action,
which is automatically triggered daily or can be triggered manually. In case of
conflicts, you will need to manually resolve them and force-push the changes to
the fork. Ensure you have the required permissions to perform these tasks.

Follow the steps below to set up your environment and handle different synchronization scenarios:

#### Initial Setup

If starting from scratch, clone the repository and set up the required remotes and submodules:

```shell
git clone https://github.com/upbound/crossplane
cd crossplane
git remote add upstream https://github.com/crossplane/crossplane
git fetch --all
git submodule update --init
```

If the repository was already cloned, update the remotes and submodules as follows:

```shell
git checkout master
git fetch --all
git submodule update --init
# To ensure you are on the latest master, note that any local changes on master will be lost
git reset --hard origin/master 
```

Depending on the task, choose one of the scenarios below:

##### Handling Branch Sync Failure

**Ensure you have pulled the latest changes from the origin remote.**

For each branch that failed to sync (starting with `master`, if it failed as well), perform the following steps:

1. Manually sync the branch `$BRANCH` using the `./hack/uxp/fork.sh` script from the `upbound/crossplane` repository:

   ```shell
   git checkout master # ALWAYS USE THE SCRIPT FROM MASTER
   # CHANGES WILL BE LOST ON <branch> IF NOT PREVIOUSLY PUSHED
   ./hack/uxp/fork.sh sync_branch <branch> # e.g. release-1.12
   ```

   This function will fail if there are conflicts.

2. Resolve conflicts using your preferred tool, e.g., `git mergetool`.

3. Add the resolved files to the index: `git add <files>`.

4. Continue the rebase: `git rebase --continue`.

5. Repeat steps 2-4 until the rebase is complete.

6. Force-push with lease to avoid overriding changes you did not pull to the fork: `git push --force-with-lease origin <branch>`.

##### Synchronizing Up to a Specific Tag or Commit

To sync up to a specific commit or tag (even if the branch was already synced), use the following command:

```shell
git checkout master # Always use the script from master

./hack/uxp/fork.sh sync_branch <branch> <commit_or_tag> # e.g. release-1.12 v1.12.0
```

For more information on the synchronization GitHub action, refer to [xp-fork-sync-gha][xp-fork-sync-gha].

##### Break Glass Procedure

If you think you force pushed the wrong thing, DO NOT PANIC, you can almost surely recover it.

```shell
git reflog # Find the commit you want to recover
# or filter it down with: git reflog <branch>
git reset --hard <commit> # e.g. HEAD@{1}
```

See [git-reflog][git-reflog] and [undo-force-push][undo-force-push] for more information.

## Deprecated or on-demand release channels

New UXP releases can be also published to the following release channels, but
for the time being we are not proactively doing so. Below sections should only
be followed if explicitly requested:

### Operator Lifecycle Manager

OLM uses its own packaging format for its marketplaces.

1. **download bundle**: Download the bundle from the following link:
   ```
   VERSION=<release version> # including v prefix, i.e. v1.3.3-up.1
   curl -sL https://releases.upbound.io/universal-crossplane/stable/$VERSION/olm/$VERSION.tar.gz  | tar xz
   ```
1. **open pull requests**: Extract the content into a new folder named with version **without v prefix**:
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
   ```bash
   VERSION=<released-version> # excluding v prefix, i.e. 1.3.3-up.1
   curl -sL https://charts.upbound.io/stable/universal-crossplane-$VERSION.tgz | tar xz
   ```
1. **login to ECR**: You need to have an AWS user in production marketplace account
   and use the following command to login your docker client:
   ```bash
   # Note that the account ID is the one used by all products in the marketplace, not our
   # account ID.
   aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin "709825985650.dkr.ecr.us-east-1.amazonaws.com"
   ```
1. **tag & push**: Tag and push every image like the following:
   ```bash
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

[git-reflog]: https://git-scm.com/docs/git-reflog
[undo-force-push]: https://www.jvt.me/posts/2021/10/23/undo-force-push/
