package operations

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

const (
	name = "TLSSecretGeneration"
)

type TLSSecretOperation struct {
}

func NewTLSSecretOperation() TLSSecretOperation {
	return TLSSecretOperation{}
}

func (T TLSSecretOperation) GetName() string {
	return name
}

func (T TLSSecretOperation) Run(ctx context.Context, log logging.Logger, config map[string][]byte) error {
	log.Info("Running ", "operation", name)

	return nil
}
