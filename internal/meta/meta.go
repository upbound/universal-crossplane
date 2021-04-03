package meta

const (
	// LabelKeyManagedBy is the key for the label indicating resource is managed by bootstrapper
	LabelKeyManagedBy = "upbound.io/managed-by"
	// LabelValueManagedBy is the value for the label indicating resource is managed by bootstrapper
	LabelValueManagedBy = "bootstrapper"
	// SecretNameControlPlaneToken is the name of the Secret that contains control plane token
	// the cluster is connected to.
	SecretNameControlPlaneToken = "upbound-control-plane-token"
	// SecretKeyControlPlaneToken is the key whose value is the control plane token
	// in control plane Secret.
	SecretKeyControlPlaneToken = "token"
)
