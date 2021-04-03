package controllers

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/upbound/universal-crossplane/internal/clients/upbound"
	"github.com/upbound/universal-crossplane/internal/controllers/tlssecrets"
	"github.com/upbound/universal-crossplane/internal/controllers/ubccerts"
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
