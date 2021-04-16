package main

import (
	"time"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	Bootstrap BootstrapCmd `cmd:"" name:"start" help:"Bootstraps Universal Crossplane"`
}

func main() {
	ctx := kong.Parse(&cli)
	zl := zap.New(zap.UseDevMode(cli.Debug))
	ctrl.SetLogger(zl)
	s := runtime.NewScheme()
	ctx.FatalIfErrorf(corev1.AddToScheme(s), "cannot add client-go scheme")

	cfg, err := ctrl.GetConfig()
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot get config"))
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:     s,
		SyncPeriod: &cli.Bootstrap.SyncPeriod,
		Namespace:  cli.Bootstrap.Namespace,
	})
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot create manager"))

	logger := logging.NewLogrLogger(zl.WithName("bootstrapper"))
	for _, c := range cli.Bootstrap.Controllers {
		switch c {
		case "tls-secrets":
			ctx.FatalIfErrorf(errors.Wrapf(tlssecrets.Setup(mgr, logger), "cannot start %s controller", c))
		case "ubc-certs":
			ctx.FatalIfErrorf(errors.Wrapf(ubccerts.Setup(mgr, logger, upbound.NewClient(cli.Bootstrap.UpboundAPIUrl, cli.Debug)), "cannot setup %s controller", c))
		case "aws-marketplace":
			ctx.FatalIfErrorf(errors.Wrapf(billing.SetupAWSMarketplace(mgr, logger), "cannot setup %s controller", c))
		default:
			ctx.Errorf("unknown controller name: %s", c)
		}
	}

	logger.Info("Starting bootstrapper", "version", version.Version)
	ctx.FatalIfErrorf(errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager"))
}
