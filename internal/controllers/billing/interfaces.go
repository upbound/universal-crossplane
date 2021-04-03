package billing

import "context"

type Registerer interface {
	Register(ctx context.Context, uid string) (string, error)
	Verify(token, uid string) (bool, error)
}

func NewNopRegisterer() NopRegisterer {
	return NopRegisterer{}
}

type NopRegisterer struct{}

func (np NopRegisterer) Register(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (np NopRegisterer) Verify(token, uid string) (bool, error) {
	return true, nil
}
