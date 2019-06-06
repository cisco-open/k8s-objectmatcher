module github.com/banzaicloud/k8s-objectmatcher

go 1.12

require (
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/goph/emperror v0.17.1
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20190528154508-67ef80593b24
	k8s.io/apiextensions-apiserver v0.0.0-20190426053235-842c4571cde0
	k8s.io/apimachinery v0.0.0-20190528154326-e59c2fb0a8e5
	k8s.io/apiserver v0.0.0-20190528155802-e15d7878a7c8 // indirect
	k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
	k8s.io/klog v0.3.1
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20190529001817-6999998975a7 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20181126151915-b503174bad59
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181126155829-0cd23ebeb688
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181126123746-eddba98df674
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20181126153457-92fdef3a232a
	k8s.io/client-go => k8s.io/client-go v0.0.0-20181126152608-d082d5923d3c
	k8s.io/kubernetes => k8s.io/kubernetes v1.12.3
)
