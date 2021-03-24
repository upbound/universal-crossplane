module github.com/upbound/crossplane-distro

go 1.16

require (
	github.com/alecthomas/kong v0.2.16
	github.com/crossplane/crossplane v1.1.0 // indirect
	github.com/crossplane/crossplane-runtime v0.13.0
	github.com/go-resty/resty/v2 v2.5.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/controller-runtime v0.8.0
)
