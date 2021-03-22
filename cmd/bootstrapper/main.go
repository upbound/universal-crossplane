package main

import (
	"fmt"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/upbound/crossplane-distro/internal/controller/bootstrap"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Context struct {
	Debug bool
}

type BootstrapCmd struct {
	SyncPeriod time.Duration `default:"5m"`
	Namespace  string        `default:"crossplane-system"`
}

var cli struct {
	Debug bool `help:"Enable debug mode"`

	Bootstrap BootstrapCmd `cmd help:"Bootstraps project uruk hai"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}

func (b *BootstrapCmd) Run(ctx *Context) error {
	fmt.Println("bootstrapping...")
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

	if err := bootstrap.Setup(mgr, logging.NewLogrLogger(zl.WithName("bootstrap"))); err != nil {
		return errors.Wrap(err, "cannot add bootstrap controller to manager")
	}

	return errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager")
}
