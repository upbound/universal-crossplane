package bootstrap

import (
	"github.com/upbound/crossplane-distro/internal/controller/bootstrap/operations/ubccerts"

	"github.com/upbound/crossplane-distro/internal/controller/bootstrap/operations/tlssecrets"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/upbound/crossplane-distro/internal/clients/upbound"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Setup adds a controller that runs bootstrap operations
func Setup(mgr ctrl.Manager, l logging.Logger, ubcClient upbound.Client) error {
	err := tlssecrets.Setup(mgr, l)
	if err != nil {
		return err
	}
	err = ubccerts.Setup(mgr, l, ubcClient)
	if err != nil {
		return err
	}
	return nil
}
