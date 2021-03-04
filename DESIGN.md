# Packaging Upbound Distro

* Owner: Muvaffak Onuş (@muvaf)
* Reviewers: Upbound Engineering
* Status: Draft

## Problem Statement

The main goal of Upbound Distribution of Crossplane is to provide a well-tested
version of Crossplane and the ability to connect the cluster to Upbound Cloud.
Most of the components that will allow the cluster to be connected to Upbound Cloud
is also used by hosted Crossplane service. So, the components that will make up
this distro live in several different repositories, including open source Crossplane.

Currently, the only component that is ready to be shipped in the distro artifact
is Crossplane. However, we will add other components as they get ready in a fashion
that will allow seamless upgrades. Additionally, current open source deployments
of Crossplane should be upgraded with this distro without any disruption of service.

## Proposal

A new repository will be created to host the main Helm chart of Upbound Distribution.
It will contain all manifests that are necessary to deploy all the components
without introducing a Helm chart dependency since upgrade scenarios do not work
well with Helm's dependency management.

The directory structure will look like the following:
```
.
├── DESIGN.md
├── Makefile
├── README.md
└── charts
    └── upbound-crossplane
        ├── crossplane
        │   └── templates
        │       ├── NOTES.txt
        │       ├── _helpers.tpl
        │       ├── clusterrole.yaml
        │       ├── clusterrolebinding.yaml
        │       ├── configuration.yaml
        │       ├── deployment.yaml
        │       ├── lock.yaml
        │       ├── provider.yaml
        │       ├── rbac-manager-allowed-provider-permissions.yaml
        │       ├── rbac-manager-clusterrole.yaml
        │       ├── rbac-manager-clusterrolebinding.yaml
        │       ├── rbac-manager-deployment.yaml
        │       ├── rbac-manager-managed-clusterroles.yaml
        │       ├── rbac-manager-serviceaccount.yaml
        │       └── serviceaccount.yaml
        ├── gateway
        ├── graphql
        └── values.yaml
```

As you can see, the `crossplane/templates` folder contains exact copies of YAML
templates used in open source Crossplane. The same will be true for `gateway/templates`
and `graphql/templates` folders as well. The versions of each component will reside
in `Makefile` and there will be targets to fetch `<chart>/templates` folder from
all component repositories.

The `values.yaml` will contain the default values for all components deployed in
the Helm chart.

Additional to manifests of components, we will include a `ConfigMap` that lists the
deployed versions of all component. 

### Upgrade

There are two main scenarios for upgrading:
* From open source Crossplane with the same version
* From open source Crossplane with a different version
* From an earlier version of distro that doesn't have the additional components

#### From Open Source Crossplane

When a user deploys open source Crossplane v1.0.0 using Helm, they have a Helm release
in their cluster that manages this deployment. Since distro chart will include the
same manifests, upgrading to Distro v1.0.0 will not affect running Crossplane
services. Helm 3 allows upgrade of a release with another chart and updates the
metadata of the Helm release accordingly.

The caveat in upgrading from OSS to distro is that we need to make sure distro
doesn't accidentally downgrade the existing installation, which might cause unexpected
errors. We will guard against that scenario by documentation initially and then
have a pre-install hook in Helm that will do the necessary pre-flight checks.

In order for documentation approach to be good enough, the only mode we will
support for upgrading from a different Crossplane version is where both distro
and the existing installation share the same version. For example, if user has
Crossplane v1.0.0 but wants to upgrade to distro v1.1.0, then they will have to
upgrade to Crossplane v1.1.0 first. This way users won't have to think about the
different version pairs at all.

#### Additional Components

We plan to include additional components in the distro alongside Crossplane. Since
they will be additive, users will be able to upgrade from both open source Crossplane
and an earlier version of distro.

## Future Considerations

With this approach, we will have to make sure that variable names of different charts
originally located elsewhere do not collide. However, once other components are ready
and managed Crossplane service of Upbound starts to use the distro we'll be able to
*move* their Helm charts to this repository, so this will get easier to test.

### Monorepo

Once this is used in both self-hosted and managed scenarios, we can consider moving
the code of other components to this repository. In the end, Helm chart manifests and
code of components other than Crossplane could live in this repository. This way, we'd
be able to consolidate everything we can into one place and our final test pipelines
can run in the same repository as the one that engineers open PRs.

## Alternatives Considered

### Upbound Operator

We can have an Upbound Operator that can manage all components. However, the
components already require high privileges in the cluster, especially Crossplane
RBAC Manager, so in addition to that, we'll request users to give high privileges
to Upbound Operator as well.

Another caveat with Upbound Operator is about having many moving pieces. If we
use a management layer similar to provider-helm to manage Helm charts of the
components, then we'll bear the cost of both Helm and operator code. If it'll manage
the manfiests directly, then it's not much different then Helm itself since, at
least for now, there is not a very special logic Helm cannot handle.

The last but not the least is that we can easily go from only-Helm to operator
approach without much sacrifice, hence we can avoid paying the cost of security
concerns around the operator. Going back from operator to only-Helm incurs a higher
cost both in terms of development effor and technical feasibility.

See this [document](https://docs.google.com/document/d/1DApqQqdgAHy5lEAzUOuTIZbFsvnjuPKZnHMnQeIkCrM) for details.

### Helm Chart Dependencies

We could have one Helm chart with dependencies to other components, including
Crossplane. However, Helm dependency management is too rigid for upgrade scenarios
from a standard chart (OSS) to another one that has dependencies. Also, we lose
flexibility around variable namings that might affect more than one component.

See this [issue](https://github.com/upbound/hosted-crossplane-squad/issues/439) for details.