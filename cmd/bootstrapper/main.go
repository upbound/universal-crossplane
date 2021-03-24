package main

import (
	"time"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/upbound/crossplane-distro/internal/clients/upbound"
	"github.com/upbound/crossplane-distro/internal/controller/bootstrap"
	"github.com/upbound/crossplane-distro/internal/version"
)

type Context struct {
	Debug bool
}

type BootstrapCmd struct {
	SyncPeriod    time.Duration `default:"10m"`
	Namespace     string        `default:"crossplane-system"`
	UpboundAPIUrl string        `default:"https://api.upbound.io"`
}

var cli struct {
	Debug bool `help:"Enable debug mode"`

	Bootstrap BootstrapCmd `cmd:"" help:"Bootstraps project uruk hai"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}

func (b *BootstrapCmd) Run(ctx *Context) error {
	zl := zap.New(zap.UseDevMode(ctx.Debug))
	if ctx.Debug {
		ctrl.SetLogger(zl)
	}

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

	logger := logging.NewLogrLogger(zl.WithName("bootstrap"))
	if err := bootstrap.Setup(mgr, logger, upbound.NewClient(b.UpboundAPIUrl, ctx.Debug), b.Namespace); err != nil {
		return errors.Wrap(err, "cannot add bootstrap controller to manager")
	}

	logger.Info("Starting bootstrapper", "version", version.Version)

	return errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager")
}
