module github.com/upbound/universal-crossplane

go 1.16

require (
	github.com/alecthomas/kong v0.2.16
	github.com/crossplane/crossplane-runtime v0.13.1-0.20210329140831-9f8833ba7151
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-resty/resty/v2 v2.5.0
	github.com/golang/mock v1.4.1
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jarcoal/httpmock v1.0.8
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/echo-contrib v0.9.0
	github.com/labstack/echo/v4 v4.2.2
	github.com/labstack/gommon v0.3.0
	github.com/nats-io/jwt v1.2.2
	github.com/nats-io/nats.go v1.10.1-0.20210330225420-a0b1f60162f8
	github.com/nats-io/nkeys v0.3.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/gomega v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.4.1 // indirect
	github.com/upbound/nats-proxy v0.1.4
	go.opencensus.io v0.22.5
	golang.org/x/tools v0.0.0-20200916195026-c9a70fc28ce3 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/controller-runtime v0.8.0
)
