// Copyright 2021 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains the entrypoint for the bootstrapper.
package main

import (
	"fmt"
	"time"

	"github.com/alecthomas/kong"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/upbound/universal-crossplane/internal/controllers/billing"
	"github.com/upbound/universal-crossplane/internal/version"
)

// BootstrapCmd represents the "bootstrap" command.
type BootstrapCmd struct {
	SyncPeriod  time.Duration `default:"10m"`
	Namespace   string        `default:"upbound-system"`
	Controllers []string      `default:"aws-marketplace" help:"List of controllers you want to run" name:"controller"`
	MetricsPort int           `default:"8085"            help:"Port for metrics server."`
}

var cli struct { //nolint:gochecknoglobals // CLI definition.
	Debug bool `help:"Enable debug mode"`

	Bootstrap BootstrapCmd `cmd:"" help:"Bootstraps Universal Crossplane" name:"start"`
}

func main() {
	ctx := kong.Parse(&cli)
	zl := zap.New(zap.UseDevMode(cli.Debug))
	ctrl.SetLogger(zl)
	s := runtime.NewScheme()
	ctx.FatalIfErrorf(corev1.AddToScheme(s), "cannot add corev1 to client-go scheme")
	ctx.FatalIfErrorf(appsv1.AddToScheme(s), "cannot add appsv1 to client-go scheme")

	cfg, err := ctrl.GetConfig()
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot get config"))
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             s,
		SyncPeriod:         &cli.Bootstrap.SyncPeriod,
		Namespace:          cli.Bootstrap.Namespace,
		MetricsBindAddress: fmt.Sprintf(":%d", cli.Bootstrap.MetricsPort),
	})
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot create manager"))

	logger := logging.NewLogrLogger(zl.WithName("bootstrapper"))
	for _, c := range cli.Bootstrap.Controllers {
		switch c {
		case "aws-marketplace":
			ctx.FatalIfErrorf(errors.Wrapf(billing.SetupAWSMarketplace(mgr, logger), "cannot setup %s controller", c))
		default:
			ctx.Errorf("unknown controller name: %s", c)
		}
	}

	logger.Info("Starting bootstrapper", "version", version.Version)
	ctx.FatalIfErrorf(errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager"))
}
