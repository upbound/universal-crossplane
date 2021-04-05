package main

import (
	"time"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/upbound/universal-crossplane/internal/clients/upbound"
	"github.com/upbound/universal-crossplane/internal/controllers/billing"
	"github.com/upbound/universal-crossplane/internal/controllers/tlssecrets"
	"github.com/upbound/universal-crossplane/internal/controllers/ubccerts"
	"github.com/upbound/universal-crossplane/internal/version"
)

// BootstrapCmd represents the "bootstrap" command
type BootstrapCmd struct {
	SyncPeriod    time.Duration `default:"10m"`
	Namespace     string        `default:"upbound-system"`
	UpboundAPIUrl string        `default:"https://api.upbound.io"`
	Controllers   []string      `default:"tls-secrets,ubc-certs" name:"controller" help:"List of controllers you want to run"`
}

var cli struct {
	Debug bool `help:"Enable debug mode"`

	Bootstrap BootstrapCmd `cmd:"" help:"Bootstraps Universal Crossplane"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(cli.Debug, cli.Bootstrap.Controllers)
	ctx.FatalIfErrorf(err)
}

// Run sets up and starts the bootstrapper.
func (b *BootstrapCmd) Run(debug bool, controllers []string) error {
	zl := zap.New(zap.UseDevMode(debug))
	ctrl.SetLogger(zl)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return errors.Wrap(err, "cannot get config")
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		SyncPeriod: &b.SyncPeriod,
		Namespace:  b.Namespace,
	})
	if err != nil {
		return errors.Wrap(err, "cannot create manager")
	}

	logger := logging.NewLogrLogger(zl.WithName("bootstrapper"))
	for _, c := range controllers {
		switch c {
		case "tls-secrets":
			if err := tlssecrets.Setup(mgr, logger); err != nil {
				return err
			}
		case "ubc-certs":
			if err := ubccerts.Setup(mgr, logger, upbound.NewClient(b.UpboundAPIUrl, debug)); err != nil {
				return err
			}
		case "aws-marketplace":
			if err := billing.SetupAWSMarketplace(mgr, logger); err != nil {
				return err
			}
		default:
			return errors.Errorf("unknown controller name: %s", c)
		}
	}

	logger.Info("Starting bootstrapper", "version", version.Version)
	return errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager")
}
