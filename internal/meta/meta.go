package meta

const (
	// LabelKeyManagedBy is the key for the label indicating resource is managed by bootstrapper
	LabelKeyManagedBy = "upbound.io/managed-by"
	// LabelValueManagedBy is the value for the label indicating resource is managed by bootstrapper
	LabelValueManagedBy = "bootstrapper"
	// SecretNameEntitlement is the name of the Secret that contains the tokens
	// stored for entitlement of usage of Universal Crossplane.
	SecretNameEntitlement = "upbound-entitlement"
)
