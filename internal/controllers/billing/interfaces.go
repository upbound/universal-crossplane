package billing

import "context"

// Registerer can register usage of crossplane-distro with idempotent calls.
type Registerer interface {
	Register(ctx context.Context, namespace, uid string) (string, error)
	Verify(token, uid string) (bool, error)
}

// NewNopRegisterer returns a Registerer that does nothing.
func NewNopRegisterer() NopRegisterer {
	return NopRegisterer{}
}

// NopRegisterer implements Registerer and does nothing.
type NopRegisterer struct{}

// Register does nothing.
func (np NopRegisterer) Register(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// Verify does nothing.
func (np NopRegisterer) Verify(_, _ string) (bool, error) {
	return true, nil
}
