package controller

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/upbound/crossplane-distro/internal/clients/upbound"
	"github.com/upbound/crossplane-distro/internal/controller/tlssecrets"
	"github.com/upbound/crossplane-distro/internal/controller/ubccerts"
)

// Setup creates controllers that runs bootstrap operations
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
