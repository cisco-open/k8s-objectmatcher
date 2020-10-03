module github.com/banzaicloud/k8s-objectmatcher/tests

go 1.13

require (
	emperror.dev/errors v0.8.0
	github.com/banzaicloud/k8s-objectmatcher v1.4.1
	k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog/v2 v2.3.0
)

replace github.com/banzaicloud/k8s-objectmatcher => ../
