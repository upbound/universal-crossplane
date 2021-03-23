package operations

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UBCCertsFetcher struct {
	client    client.Client
	namespace string
}

func NewUBCCertsFetcher(c client.Client, namespace string) *UBCCertsFetcher {
	return &UBCCertsFetcher{
		client:    c,
		namespace: namespace,
	}
}

func (u *UBCCertsFetcher) Run(ctx context.Context, log logging.Logger, config map[string][]byte) error {
	log.Debug("Running UBCCertsFetcher")

	return nil
}
