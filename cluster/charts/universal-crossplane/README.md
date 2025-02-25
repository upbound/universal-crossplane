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
| dnsPolicy | string | `""` | Specify the `dnsPolicy` to be used by the Crossplane pod. |
| extraEnvVarsCrossplane | object | `{}` | Add custom environmental variables to the Crossplane pod deployment. Replaces any `.` in a variable name with `_`. For example, `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`. |
| extraEnvVarsRBACManager | object | `{}` | Add custom environmental variables to the RBAC Manager pod deployment. Replaces any `.` in a variable name with `_`. For example, `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`. |
| extraObjects | list | `[]` | To add arbitrary Kubernetes Objects during a Helm Install |
| extraVolumeMountsCrossplane | object | `{}` | Add custom `volumeMounts` to the Crossplane pod. |
| extraVolumesCrossplane | object | `{}` | Add custom `volumes` to the Crossplane pod. |
| function.packages | list | `[]` | A list of Function packages to install |
| hostNetwork | bool | `false` | Enable `hostNetwork` for the Crossplane deployment. Caution: enabling `hostNetwork` grants the Crossplane Pod access to the host network namespace. Consider setting `dnsPolicy` to `ClusterFirstWithHostNet`. |
| image.pullPolicy | string | `"IfNotPresent"` | The image pull policy used for Crossplane and RBAC Manager pods. |
| image.repository | string | `"xpkg.upbound.io/upbound/crossplane"` | Repository for the Crossplane pod image. |
| image.tag | string | `"v1.18.3-up.1"` | The Crossplane image tag. Defaults to the value of `appVersion` in `Chart.yaml`. |
| imagePullSecrets | list | `[]` | The imagePullSecret names to add to the Crossplane ServiceAccount. |
| leaderElection | bool | `true` | Enable [leader election](https://docs.crossplane.io/latest/concepts/pods/#leader-election) for the Crossplane pod. |
| metrics.enabled | bool | `false` | Enable Prometheus path, port and scrape annotations and expose port 8080 for both the Crossplane and RBAC Manager pods. |
| nameOverride | string | `"crossplane"` |  |
| nodeSelector | object | `{}` | Add `nodeSelectors` to the Crossplane pod deployment. |
| packageCache.configMap | string | `""` | The name of a ConfigMap to use as the package cache. Disables the default package cache `emptyDir` Volume. |
| packageCache.medium | string | `""` | Set to `Memory` to hold the package cache in a RAM backed file system. Useful for Crossplane development. |
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
| rbacManager.nodeSelector | object | `{}` | Add `nodeSelectors` to the RBAC Manager pod deployment. |
| rbacManager.replicas | int | `1` | The number of RBAC Manager pod `replicas` to deploy. |
| rbacManager.revisionHistoryLimit | string | `nil` | The number of RBAC Manager ReplicaSets to retain. |
| rbacManager.skipAggregatedClusterRoles | bool | `false` | Don't install aggregated Crossplane ClusterRoles. |
| rbacManager.tolerations | list | `[]` | Add `tolerations` to the RBAC Manager pod deployment. |
| rbacManager.topologySpreadConstraints | list | `[]` | Add `topologySpreadConstraints` to the RBAC Manager pod deployment. |
| registryCaBundleConfig.key | string | `""` | The ConfigMap key containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates. |
| registryCaBundleConfig.name | string | `""` | The ConfigMap name containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates. |
| replicas | int | `1` | The number of Crossplane pod `replicas` to deploy. |
| resourcesCrossplane.limits.cpu | string | `"500m"` | CPU resource limits for the Crossplane pod. |
| resourcesCrossplane.limits.memory | string | `"1024Mi"` | Memory resource limits for the Crossplane pod. |
| resourcesCrossplane.requests.cpu | string | `"100m"` | CPU resource requests for the Crossplane pod. |
| resourcesCrossplane.requests.memory | string | `"256Mi"` | Memory resource requests for the Crossplane pod. |
| resourcesRBACManager.limits.cpu | string | `"100m"` | CPU resource limits for the RBAC Manager pod. |
| resourcesRBACManager.limits.memory | string | `"512Mi"` | Memory resource limits for the RBAC Manager pod. |
| resourcesRBACManager.requests.cpu | string | `"100m"` | CPU resource requests for the RBAC Manager pod. |
| resourcesRBACManager.requests.memory | string | `"256Mi"` | Memory resource requests for the RBAC Manager pod. |
| revisionHistoryLimit | string | `nil` | The number of Crossplane ReplicaSets to retain. |
| securityContextCrossplane.allowPrivilegeEscalation | bool | `false` | Enable `allowPrivilegeEscalation` for the Crossplane pod. |
| securityContextCrossplane.readOnlyRootFilesystem | bool | `true` | Set the Crossplane pod root file system as read-only. |
| securityContextCrossplane.runAsGroup | int | `65532` | The group ID used by the Crossplane pod. |
| securityContextCrossplane.runAsUser | int | `65532` | The user ID used by the Crossplane pod. |
| securityContextRBACManager.allowPrivilegeEscalation | bool | `false` | Enable `allowPrivilegeEscalation` for the RBAC Manager pod. |
| securityContextRBACManager.readOnlyRootFilesystem | bool | `true` | Set the RBAC Manager pod root file system as read-only. |
| securityContextRBACManager.runAsGroup | int | `65532` | The group ID used by the RBAC Manager pod. |
| securityContextRBACManager.runAsUser | int | `65532` | The user ID used by the RBAC Manager pod. |
| service.customAnnotations | object | `{}` | Configure annotations on the service object. Only enabled when webhooks.enabled = true |
| serviceAccount.customAnnotations | object | `{}` | Add custom `annotations` to the Crossplane ServiceAccount. |
| tolerations | list | `[]` | Add `tolerations` to the Crossplane pod deployment. |
| topologySpreadConstraints | list | `[]` | Add `topologySpreadConstraints` to the Crossplane pod deployment. |
| webhooks.enabled | bool | `true` | Enable webhooks for Crossplane and installed Provider packages. |

