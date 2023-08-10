# universal-crossplane

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![AppVersion: 0.0.1](https://img.shields.io/badge/AppVersion-0.0.1-informational?style=flat-square)

Upbound Universal Crossplane (UXP) is Upbound's official enterprise-grade
distribution of Crossplane. It's fully compatible with upstream Crossplane,
open source, capable of connecting to Upbound Cloud for real-time dashboard
visibility, and maintained by Upbound. It's the easiest way for both
individual community members and enterprises to build their production control
planes.

**Homepage:** <https://upbound.io>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Upbound Inc. | <info@upbound.io> |  |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Add `affinities` to the Crossplane pod deployment. |
| args | list | `[]` | Add custom arguments to the Crossplane pod. |
| billing.awsMarketplace.enabled | bool | `false` | Enable AWS Marketplace billing. |
| billing.awsMarketplace.iamRoleARN | string | `"arn:aws:iam::<ACCOUNT_ID>:role/<ROLE_NAME>"` | AWS Marketplace billing IAM role ARN. |
| bootstrapper.config.args | list | `[]` | List of additional args for the bootstrapper deployment. |
| bootstrapper.config.debugMode | bool | `false` | Enable debug mode for bootstrapper. |
| bootstrapper.config.envVars | object | `{}` | List of additional environment variables for the bootstrapper deployment. EXAMPLE envVars:   sample.key: value1   ANOTHER.KEY: value2 RESULT   - name: sample_key     value: "value1"   - name: ANOTHER_KEY     value: "value2" |
| bootstrapper.image.pullPolicy | string | `"IfNotPresent"` | Bootstrapper image pull policy. |
| bootstrapper.image.repository | string | `"xpkg.upbound.io/upbound/uxp-bootstrapper"` | Bootstrapper image repository. |
| bootstrapper.image.tag | string | `""` | Bootstrapper image tag: if not set, appVersion field from Chart.yaml is used. |
| bootstrapper.resources | object | `{}` | Resources configuration for bootstrapper. |
| configuration.packages | list | `[]` | A list of Configuration packages to install. |
| customAnnotations | object | `{}` | Add custom `annotations` to the Crossplane pod deployment. |
| customLabels | object | `{}` | Add custom `labels` to the Crossplane pod deployment. |
| deploymentStrategy | string | `"RollingUpdate"` | The deployment strategy for the Crossplane and RBAC Manager pods. |
| extraEnvVarsCrossplane | object | `{}` | Add custom environmental variables to the Crossplane pod deployment. Replaces any `.` in a variable name with `_`. For example, `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`. |
| extraEnvVarsRBACManager | object | `{}` | Add custom environmental variables to the RBAC Manager pod deployment. Replaces any `.` in a variable name with `_`. For example, `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`. |
| extraVolumeMountsCrossplane | object | `{}` | Add custom `volumeMounts` to the Crossplane pod. |
| extraVolumesCrossplane | object | `{}` | Add custom `volumes` to the Crossplane pod. |
| hostNetwork | bool | `false` | Enable `hostNetwork` for the Crossplane deployment. Caution: enabling `hostNetwork`` grants the Crossplane Pod access to the host network namespace. |
| image.pullPolicy | string | `"IfNotPresent"` | The image pull policy used for Crossplane and RBAC Manager pods. |
| image.repository | string | `"upbound/crossplane"` | Repository for the Crossplane pod image. |
| image.tag | string | `"v1.13.2-up.1"` | The Crossplane image tag. Defaults to the value of `appVersion` in Chart.yaml. |
| imagePullSecrets | object | `{}` | The imagePullSecret names to add to the Crossplane ServiceAccount. |
| leaderElection | bool | `true` | Enable [leader election](https://docs.crossplane.io/latest/concepts/pods/#leader-election) for the Crossplane pod. |
| metrics.enabled | bool | `false` | Enable Prometheus path, port and scrape annotations and expose port 8080 for both the Crossplane and RBAC Manager pods. |
| nameOverride | string | `"crossplane"` |  |
| nodeSelector | object | `{}` | Add `nodeSelectors` to the Crossplane pod deployment. |
| packageCache.configMap | string | `""` | The name of a ConfigMap to use as the package cache. Disables the default package cache `emptyDir` Volume. |
| packageCache.medium | string | `""` | Set to `Memory` to hold the package cache in a RAM-backed file system. Useful for Crossplane development. |
| packageCache.pvc | string | `""` | The name of a PersistentVolumeClaim to use as the package cache. Disables the default package cache `emptyDir` Volume. |
| packageCache.sizeLimit | string | `"20Mi"` | The size limit for the package cache. If medium is `Memory` the `sizeLimit` can't exceed Node memory. |
| podSecurityContextCrossplane | object | `{}` | Add a custom `securityContext` to the Crossplane pod. |
| podSecurityContextRBACManager | object | `{}` | Add a custom `securityContext` to the RBAC Manager pod. |
| priorityClassName | string | `""` | The PriorityClass name to apply to the Crossplane and RBAC Manager pods. |
| provider.packages | list | `[]` | A list of Provider packages to install. |
| rbacManager.affinity | object | `{}` | Add `affinities` to the RBAC Manager pod deployment. |
| rbacManager.args | list | `[]` | Add custom arguments to the RBAC Manager pod. |
| rbacManager.deploy | bool | `true` | Deploy the RBAC Manager pod and its required roles. |
| rbacManager.leaderElection | bool | `true` | Enable [leader election](https://docs.crossplane.io/latest/concepts/pods/#leader-election) for the RBAC Manager pod. |
| rbacManager.managementPolicy | string | `"Basic"` | Defines the Roles and ClusterRoles the RBAC Manager creates and manages. - A policy of `Basic` creates and binds Roles only for the Crossplane ServiceAccount, Provider ServiceAccounts and creates Crossplane ClusterRoles. - A policy of `All` includes all the `Basic` settings and also creates Crossplane Roles in all namespaces. - Read the Crossplane docs for more information on the [RBAC Roles and ClusterRoles](https://docs.crossplane.io/latest/concepts/pods/#crossplane-clusterroles) |
| rbacManager.nodeSelector | object | `{}` | Add `nodeSelectors` to the RBAC Manager pod deployment. |
| rbacManager.replicas | int | `1` | The number of RBAC Manager pod `replicas` to deploy. |
| rbacManager.skipAggregatedClusterRoles | bool | `false` | Don't install aggregated Crossplane ClusterRoles. |
| rbacManager.tolerations | list | `[]` | Add `tolerations` to the RBAC Manager pod deployment. |
| registryCaBundleConfig.key | string | `""` | The ConfigMap key containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates. |
| registryCaBundleConfig.name | string | `""` | The ConfigMap name containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates. |
| replicas | int | `1` | The number of Crossplane pod `replicas` to deploy. |
| resourcesCrossplane.limits.cpu | string | `"100m"` | CPU resource limits for the Crossplane pod. |
| resourcesCrossplane.limits.memory | string | `"512Mi"` | Memory resource limits for the Crossplane pod. |
| resourcesCrossplane.requests.cpu | string | `"100m"` | CPU resource requests for the Crossplane pod. |
| resourcesCrossplane.requests.memory | string | `"256Mi"` | Memory resource requests for the Crossplane pod. |
| resourcesRBACManager.limits.cpu | string | `"100m"` | CPU resource limits for the RBAC Manager pod. |
| resourcesRBACManager.limits.memory | string | `"512Mi"` | Memory resource limits for the RBAC Manager pod. |
| resourcesRBACManager.requests.cpu | string | `"100m"` | CPU resource requests for the RBAC Manager pod. |
| resourcesRBACManager.requests.memory | string | `"256Mi"` | Memory resource requests for the RBAC Manager pod. |
| securityContextCrossplane.allowPrivilegeEscalation | bool | `false` | Enable `allowPrivilegeEscalation` for the Crossplane pod. |
| securityContextCrossplane.readOnlyRootFilesystem | bool | `true` | Set the Crossplane pod root file system as read-only. |
| securityContextCrossplane.runAsGroup | int | `65532` | The group ID used by the Crossplane pod. |
| securityContextCrossplane.runAsUser | int | `65532` | The user ID used by the Crossplane pod. |
| securityContextRBACManager.allowPrivilegeEscalation | bool | `false` | Enable `allowPrivilegeEscalation` for the RBAC Manager pod. |
| securityContextRBACManager.readOnlyRootFilesystem | bool | `true` | Set the RBAC Manager pod root file system as read-only. |
| securityContextRBACManager.runAsGroup | int | `65532` | The group ID used by the RBAC Manager pod. |
| securityContextRBACManager.runAsUser | int | `65532` | The user ID used by the RBAC Manager pod. |
| serviceAccount.customAnnotations | object | `{}` | Add custom `annotations` to the Crossplane ServiceAccount. |
| tolerations | list | `[]` | Add `tolerations` to the Crossplane pod deployment. |
| webhooks.enabled | bool | `true` | Enable webhooks for Crossplane and installed Provider packages. |
| xfn.args | list | `[]` | Add custom arguments to the Composite functions runner container. |
| xfn.cache.configMap | string | `""` | The name of a ConfigMap to use as the Composite function runner package cache. Disables the default Composite function runner package cache `emptyDir` Volume. |
| xfn.cache.medium | string | `""` | Set to `Memory` to hold the Composite function runner package cache in a RAM-backed file system. Useful for Crossplane development. |
| xfn.cache.pvc | string | `""` | The name of a PersistentVolumeClaim to use as the Composite function runner package cache. Disables the default Composite function runner package cache `emptyDir` Volume. |
| xfn.cache.sizeLimit | string | `"1Gi"` | The size limit for the Composite function runner package cache. If medium is `Memory` the `sizeLimit` can't exceed Node memory. |
| xfn.enabled | bool | `false` | Enable the alpha Composition functions (`xfn`) sidecar container. Also requires Crossplane `args` value `--enable-composition-functions` set. |
| xfn.extraEnvVars | object | `{}` | Add custom environmental variables to the Composite function runner container. Replaces any `.` in a variable name with `_`. For example, `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`. |
| xfn.image.pullPolicy | string | `"IfNotPresent"` | Composite function runner container image pull policy. |
| xfn.image.repository | string | `"upbound/xfn"` | Composite function runner container image. |
| xfn.image.tag | string | `"v1.13.2-up.1"` | Composite function runner container image tag. Defaults to the value of `appVersion` in Chart.yaml. |
| xfn.resources.limits.cpu | string | `"2000m"` | CPU resource limits for the Composite function runner container. |
| xfn.resources.limits.memory | string | `"2Gi"` | Memory resource limits for the Composite function runner container. |
| xfn.resources.requests.cpu | string | `"1000m"` | CPU resource requests for the Composite function runner container. |
| xfn.resources.requests.memory | string | `"1Gi"` | Memory resource requests for the Composite function runner container. |
| xfn.securityContext.allowPrivilegeEscalation | bool | `false` | Enable `allowPrivilegeEscalation` for the Composite function runner container. |
| xfn.securityContext.capabilities.add | list | `["SETUID","SETGID"]` | Set Linux capabilities for the Composite function runner container. The default values allow the container to create an unprivileged user namespace for running Composite function containers. |
| xfn.securityContext.readOnlyRootFilesystem | bool | `true` | Set the Composite function runner container root file system as read-only. |
| xfn.securityContext.runAsGroup | int | `65532` | The group ID used by the Composite function runner container. |
| xfn.securityContext.runAsUser | int | `65532` | The user ID used by the Composite function runner container. |
| xfn.securityContext.seccompProfile.type | string | `"Unconfined"` | Apply a `seccompProfile` to the Composite function runner container. The default value allows the Composite function runner container permissions to use the `unshare` syscall. |

