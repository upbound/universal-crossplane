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
| affinity | object | `{}` | Enable affinity for Crossplane pod. |
| args | list | `[]` | A list of additional args to be passed to Crossplane's container. |
| billing.awsMarketplace.enabled | bool | `false` | Enable AWS Marketplace billing. |
| billing.awsMarketplace.iamRoleARN | string | `"arn:aws:iam::<ACCOUNT_ID>:role/<ROLE_NAME>"` | AWS Marketplace billing IAM role ARN. |
| bootstrapper.config.args | list | `[]` | List of additional args for the bootstrapper deployment. |
| bootstrapper.config.debugMode | bool | `false` | Enable debug mode for bootstrapper. |
| bootstrapper.config.envVars | object | `{}` | List of additional environment variables for the bootstrapper deployment. EXAMPLE envVars:   sample.key: value1   ANOTHER.KEY: value2 RESULT   - name: sample_key     value: "value1"   - name: ANOTHER_KEY     value: "value2" |
| bootstrapper.image.pullPolicy | string | `"IfNotPresent"` | Bootstrapper image pull policy. |
| bootstrapper.image.repository | string | `"xpkg.upbound.io/upbound/uxp-bootstrapper"` | Bootstrapper image repository. |
| bootstrapper.image.tag | string | `""` | Bootstrapper image tag: if not set, appVersion field from Chart.yaml is used. |
| bootstrapper.resources | object | `{}` | Resources configuration for bootstrapper. |
| configuration.packages | list | `[]` | The list of Configuration packages to install together with Crossplane. |
| customAnnotations | object | `{}` | Custom annotations to add to the Crossplane deployment and pod. |
| customLabels | object | `{}` | Custom labels to add into metadata. |
| deploymentStrategy | string | `"RollingUpdate"` | The deployment strategy for the Crossplane and RBAC Manager (if enabled) pods. |
| extraEnvVarsCrossplane | object | `{}` | List of extra environment variables to set in the Crossplane deployment. Any `.` in variable names will be replaced with `_` (example: `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`). |
| extraEnvVarsRBACManager | object | `{}` | List of extra environment variables to set in the Crossplane rbac manager deployment. Any `.` in variable names will be replaced with `_` (example: `SAMPLE.KEY=value1` becomes `SAMPLE_KEY=value1`). |
| extraVolumeMountsCrossplane | object | `{}` | List of extra volumesMounts to add to Crossplane. |
| extraVolumesCrossplane | object | `{}` | List of extra Volumes to add to Crossplane. |
| hostNetwork | bool | `false` | Enable hostNetwork for Crossplane. Caution: setting it to true means Crossplane's Pod will have high privileges. |
| image.pullPolicy | string | `"IfNotPresent"` | Crossplane image pull policy used in all containers. |
| image.repository | string | `"upbound/crossplane"` | Crossplane image. |
| image.tag | string | `"v1.12.2-up.2"` | Crossplane image tag: if not set, appVersion field from Chart.yaml is used. |
| imagePullSecrets | object | `{}` | Names of image pull secrets to use. |
| leaderElection | bool | `true` | Enable leader election for Crossplane Managers pod. |
| metrics.enabled | bool | `false` | Expose Crossplane and RBAC Manager metrics endpoint. |
| nameOverride | string | `"crossplane"` |  |
| nodeSelector | object | `{}` | Enable nodeSelector for Crossplane pod. |
| packageCache.configMap | string | `""` | Name of the ConfigMap to be used as package cache. Providing a value will cause the default emptyDir volume not to be mounted. |
| packageCache.medium | string | `""` | Storage medium for package cache. `Memory` means volume will be backed by tmpfs, which can be useful for development. |
| packageCache.pvc | string | `""` | Name of the PersistentVolumeClaim to be used as the package cache. Providing a value will cause the default emptyDir volume to not be mounted. |
| packageCache.sizeLimit | string | `"20Mi"` | Size limit for package cache. If medium is `Memory` then maximum usage would be the minimum of this value the sum of all memory limits on containers in the Crossplane pod. |
| podSecurityContextCrossplane | object | `{}` | PodSecurityContext for Crossplane. |
| podSecurityContextRBACManager | object | `{}` | PodSecurityContext for RBAC Manager. |
| priorityClassName | string | `""` | Priority class name for Crossplane and RBAC Manager (if enabled) pods. |
| provider.packages | list | `[]` | The list of Provider packages to install together with Crossplane. |
| rbacManager.affinity | object | `{}` | Enable affinity for RBAC Managers pod. |
| rbacManager.args | list | `[]` | A list of additional args to be pased to the RBAC manager's container. |
| rbacManager.deploy | bool | `true` | Deploy RBAC Manager and its required roles. |
| rbacManager.leaderElection | bool | `true` | Enable leader election for RBAC Managers pod. |
| rbacManager.managementPolicy | string | `"All"` | The extent to which the RBAC manager will manage permissions:. - `All` indicates to manage all Crossplane controller and user roles. - `Basic` indicates to only manage Crossplane controller roles and the `crossplane-admin`, `crossplane-edit`, and `crossplane-view` user roles. |
| rbacManager.nodeSelector | object | `{}` | Enable nodeSelector for RBAC Managers pod. |
| rbacManager.replicas | int | `1` | The number of replicas to run for the RBAC Manager pods. |
| rbacManager.skipAggregatedClusterRoles | bool | `false` | Opt out of deploying aggregated ClusterRoles. |
| rbacManager.tolerations | list | `[]` | Enable tolerations for RBAC Managers pod. |
| registryCaBundleConfig.key | object | `{}` | Key to use from ConfigMap containing additional CA bundle for fetching from package registries. |
| registryCaBundleConfig.name | object | `{}` | Name of ConfigMap containing additional CA bundle for fetching from package registries. |
| replicas | int | `1` | The number of replicas to run for the Crossplane pods. |
| resourcesCrossplane.limits.cpu | string | `"100m"` | CPU resource limits for Crossplane. |
| resourcesCrossplane.limits.memory | string | `"512Mi"` | Memory resource limits for Crossplane. |
| resourcesCrossplane.requests.cpu | string | `"100m"` | CPU resource requests for Crossplane. |
| resourcesCrossplane.requests.memory | string | `"256Mi"` | Memory resource requests for Crossplane. |
| resourcesRBACManager.limits.cpu | string | `"100m"` | CPU resource limits for RBAC Manager. |
| resourcesRBACManager.limits.memory | string | `"512Mi"` | Memory resource limits for RBAC Manager. |
| resourcesRBACManager.requests.cpu | string | `"100m"` | CPU resource requests for RBAC Manager. |
| resourcesRBACManager.requests.memory | string | `"256Mi"` | Memory resource requests for RBAC Manager. |
| securityContextCrossplane.allowPrivilegeEscalation | bool | `false` | Allow privilege escalation for Crossplane. |
| securityContextCrossplane.readOnlyRootFilesystem | bool | `true` | ReadOnly root filesystem for Crossplane. |
| securityContextCrossplane.runAsGroup | int | `65532` | Run as group for Crossplane. |
| securityContextCrossplane.runAsUser | int | `65532` | Run as user for Crossplane. |
| securityContextRBACManager.allowPrivilegeEscalation | bool | `false` | Allow privilege escalation for RBAC Manager. |
| securityContextRBACManager.readOnlyRootFilesystem | bool | `true` | ReadOnly root filesystem for RBAC Manager. |
| securityContextRBACManager.runAsGroup | int | `65532` | Run as group for RBAC Manager. |
| securityContextRBACManager.runAsUser | int | `65532` | Run as user for RBAC Manager. |
| serviceAccount.customAnnotations | object | `{}` | Custom annotations to add to the serviceaccount of Crossplane. |
| tolerations | list | `[]` | Enable tolerations for Crossplane pod. |
| webhooks.enabled | bool | `true` | Enable webhook functionality for Crossplane as well as packages installed by Crossplane. |
| xfn.args | list | `[]` | List of additional args for the xfn container. |
| xfn.cache | object | `{"configMap":"","medium":"","pvc":"","sizeLimit":"1Gi"}` | Cache configuration for xfn. |
| xfn.enabled | bool | `false` | Enable alpha xfn sidecar container that runs Composition Functions. Note you also need to run Crossplane with --enable-composition-functions for it to call xfn. |
| xfn.extraEnvVars | object | `{}` | List of additional environment variables for the xfn container. |
| xfn.image | object | `{"pullPolicy":"IfNotPresent","repository":"upbound/xfn","tag":"v1.12.2-up.2"}` | Image for xfn: if tag is not set appVersion field from Chart.yaml is used. |
| xfn.resources | object | `{"limits":{"cpu":"2000m","memory":"2Gi"},"requests":{"cpu":"1000m","memory":"1Gi"}}` | Resources definition for xfn. |
| xfn.resources.limits.cpu | string | `"2000m"` | CPU resource limits for RBAC Manager. |
| xfn.resources.limits.memory | string | `"2Gi"` | Memory resource limits for RBAC Manager. |
| xfn.resources.requests.cpu | string | `"1000m"` | CPU resource requests for RBAC Manager. |
| xfn.resources.requests.memory | string | `"1Gi"` | Memory resource requests for RBAC Manager. |
| xfn.securityContext.allowPrivilegeEscalation | bool | `false` | Allow privilege escalation for xfn sidecar. |
| xfn.securityContext.capabilities | object | `{"add":["SETUID","SETGID"]}` | Capabilities configuration for xfn sidecar. These capabilities allow xfn sidecar to create better user namespaces. It drops them after creating a namespace. |
| xfn.securityContext.readOnlyRootFilesystem | bool | `true` | ReadOnly root filesystem for xfn sidecar. |
| xfn.securityContext.runAsGroup | int | `65532` | Run as group for xfn sidecar. |
| xfn.securityContext.runAsUser | int | `65532` | Run as user for xfn sidecar. |
| xfn.securityContext.seccompProfile | object | `{"type":"Unconfined"}` | Seccomp Profile for xfn. xfn needs the unshare syscall, which most RuntimeDefault seccomp profiles do not allow. |

