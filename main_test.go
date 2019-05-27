package objectmatch

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog"
)

var (
	integration = flag.Bool("integration", false, "run integration tests")
	kubeconfig = flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube/config"), "kubernetes config to use for tests")
	kubecontext = flag.String("kubecontext", "", "kubernetes context to use in tests")
	keepnamespace = flag.Bool("keepnamespace", false, "keep the kubernetes namespace that was used for the tests")
	testContext = &IntegrationTestContext{}
)


func TestMain(m *testing.M) {

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Set("v", "3")

	flag.Parse()

	if *integration {

		config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
			&clientcmd.ConfigOverrides{CurrentContext:*kubecontext},
		).ClientConfig()
		if err != nil {
			panic(err.Error())
		}
		testContext.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		err = testContext.CreateNamespace()
		if err != nil {
			panic("Failed to setup test namespace")
		}
	}
	result := m.Run()
	if *integration {
		if !*keepnamespace {
			err := testContext.DeleteNamespace()
			if err != nil {
				panic("Failed to delete test namespace")
			}
		}
	}
	os.Exit(result)
}

type IntegrationTestContext struct {
	Client kubernetes.Interface
	Namespace string
}
func (ctx *IntegrationTestContext) CreateNamespace() error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "integration-",
		},
	}
	namespace, err := ctx.Client.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return err
	}
	ctx.Namespace = namespace.Name
	return nil
}
func (ctx *IntegrationTestContext) DeleteNamespace() error {
	err := ctx.Client.CoreV1().Namespaces().Delete(ctx.Namespace, &metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	})
	return err
}
