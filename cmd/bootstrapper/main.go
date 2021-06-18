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

package main

import (
	"encoding/base64"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/upbound/universal-crossplane/internal/controllers/billing"
	"github.com/upbound/universal-crossplane/internal/controllers/tlssecrets"
	"github.com/upbound/universal-crossplane/internal/controllers/upboundagent"
	"github.com/upbound/universal-crossplane/internal/version"
)

// BootstrapCmd represents the "bootstrap" command
type BootstrapCmd struct {
	SyncPeriod         time.Duration `default:"10m"`
	Namespace          string        `default:"upbound-system"`
	UpboundAPIUrl      string        `default:"https://api.upbound.io"`
	UpboundTokenSecret string        `default:"upbound-control-plane-token"`
	AgentManifest      string        `name:"agent-manifest" help:"Base64 encoded Kubernetes deployment spec for upbound-agent"`
	Controllers        []string      `default:"tls-secrets" name:"controller" help:"List of controllers you want to run"`
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
	ctx.FatalIfErrorf(corev1.AddToScheme(s), "cannot add corev1 to client-go scheme")
	ctx.FatalIfErrorf(appsv1.AddToScheme(s), "cannot add appsv1 to client-go scheme")

	cfg, err := ctrl.GetConfig()
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot get config"))
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:     s,
		SyncPeriod: &cli.Bootstrap.SyncPeriod,
		Namespace:  cli.Bootstrap.Namespace,
	})
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot create manager"))

	m, err := base64.StdEncoding.DecodeString(cli.Bootstrap.AgentManifest)
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot base64 decode agent manifest"))
	ds := appsv1.DeploymentSpec{}
	err = yaml.Unmarshal(m, &ds)
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot parse agent manifest as deployment spec"))

	logger := logging.NewLogrLogger(zl.WithName("bootstrapper"))
	for _, c := range cli.Bootstrap.Controllers {
		switch c {
		case "tls-secrets":
			ctx.FatalIfErrorf(errors.Wrapf(tlssecrets.Setup(mgr, logger), "cannot start %s controller", c))
		case "upbound-agent":
			ctx.FatalIfErrorf(errors.Wrapf(upboundagent.Setup(mgr, logger, ds, cli.Bootstrap.UpboundTokenSecret), "cannot start %s controller", c))
		case "aws-marketplace":
			ctx.FatalIfErrorf(errors.Wrapf(billing.SetupAWSMarketplace(mgr, logger), "cannot setup %s controller", c))
		default:
			ctx.Errorf("unknown controller name: %s", c)
		}
	}

	logger.Info("Starting bootstrapper", "version", version.Version)
	ctx.FatalIfErrorf(errors.Wrap(mgr.Start(ctrl.SetupSignalHandler()), "cannot start controller manager"))
}
