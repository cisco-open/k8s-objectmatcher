// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tests

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	"emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	integration   = flag.Bool("integration", false, "run integration tests")
	kubeconfig    = flag.String("kubeconfig", "", "kubernetes config to use for tests")
	kubecontext   = flag.String("kubecontext", "", "kubernetes context to use in tests")
	keepnamespace = flag.Bool("keepnamespace", false, "keep the kubernetes namespace that was used for the tests")
	failonerror   = flag.Bool("failonerror", false, "fail on error to be able to debug invalid state")
	testContext   = &IntegrationTestContext{}
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Verbose() {
		klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
		klog.InitFlags(klogFlags)
		err := klogFlags.Set("v", "3")
		if err != nil {
			fmt.Printf("Failed to set log level, moving on")
		}
	}

	if *integration {
		err := testContext.Setup()
		if err != nil {
			panic("Failed to setup test namespace: " + err.Error())
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
	Client           kubernetes.Interface
	DynamicClient    dynamic.Interface
	ExtensionsClient apiextension.Interface
	Namespace        string
}

func (ctx *IntegrationTestContext) CreateNamespace() error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "integration-",
		},
	}
	namespace, err := ctx.Client.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	ctx.Namespace = namespace.Name
	return nil
}

func (ctx *IntegrationTestContext) Setup() error {
	kubeconfigOverride := os.Getenv("KUBECONFIG")
	if *kubeconfig != "" {
		kubeconfigOverride = *kubeconfig
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigOverride},
		&clientcmd.ConfigOverrides{CurrentContext: *kubecontext},
	).ClientConfig()
	if err != nil {
		return err
	}
	ctx.Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create kubernetes client")
	}
	ctx.DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create dynamic client")
	}
	ctx.ExtensionsClient, err = apiextension.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create apiextensions client")
	}
	err = testContext.CreateNamespace()
	if err != nil {
		return errors.Wrap(err, "Failed to create test namespace")
	}
	return err
}

func (ctx *IntegrationTestContext) DeleteNamespace() error {
	err := ctx.Client.CoreV1().Namespaces().Delete(context.Background(), ctx.Namespace, metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	})
	return err
}

type TestItem struct {
	name           string
	object         metav1.Object
	shouldMatch    bool
	gvr            *schema.GroupVersionResource
	remoteChange   func(interface{})
	localChange    func(interface{})
	ignoreVersions []string
}

func NewTestMatch(name string, object metav1.Object) *TestItem {
	return &TestItem{
		name:        name,
		object:      object,
		shouldMatch: true,
	}
}
func NewTestDiff(name string, object metav1.Object) *TestItem {
	return &TestItem{
		name:        name,
		object:      object,
		shouldMatch: false,
	}
}

func (t *TestItem) withRemoteChange(extender func(interface{})) *TestItem {
	t.remoteChange = extender
	return t
}

func (t *TestItem) withLocalChange(extender func(interface{})) *TestItem {
	t.localChange = extender
	return t
}

func (t *TestItem) withGroupVersionResource(gvr *schema.GroupVersionResource) *TestItem {
	t.gvr = gvr
	return t
}

func (t *TestItem) withIgnoreVersions(v []string) *TestItem {
	t.ignoreVersions = v
	return t
}

func testMatchOnObject(testItem *TestItem, ignoreField string) error {
	var existing metav1.Object
	var err error
	opts := []patch.CalculateOption{
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		patch.IgnorePDBSelector(),
		patch.IgnoreField(ignoreField),
	}

	newObject := testItem.object
	err = patch.DefaultAnnotator.SetLastAppliedAnnotation(newObject.(runtime.Object))
	if err != nil {
		return err
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	}
	switch newObject.(type) {
	default:
		return errors.WithDetails(errors.New("Unsupported type"), "type", reflect.TypeOf(newObject), "object", newObject)
	case *rbacv1.ClusterRole:
		existing, err = testContext.Client.RbacV1().ClusterRoles().Create(context.Background(), newObject.(*rbacv1.ClusterRole), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.RbacV1().ClusterRoles().Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *rbacv1.Role:
		existing, err = testContext.Client.RbacV1().Roles(newObject.GetNamespace()).Create(context.Background(), newObject.(*rbacv1.Role), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.RbacV1().Roles(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *rbacv1.ClusterRoleBinding:
		existing, err = testContext.Client.RbacV1().ClusterRoleBindings().Create(context.Background(), newObject.(*rbacv1.ClusterRoleBinding), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.RbacV1().ClusterRoleBindings().Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *rbacv1.RoleBinding:
		existing, err = testContext.Client.RbacV1().RoleBindings(newObject.GetNamespace()).Create(context.Background(), newObject.(*rbacv1.RoleBinding), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.RbacV1().RoleBindings(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.Pod:
		existing, err = testContext.Client.CoreV1().Pods(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.Pod), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().Pods(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.Service:
		existing, err = testContext.Client.CoreV1().Services(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.Service), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().Services(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.ConfigMap:
		existing, err = testContext.Client.CoreV1().ConfigMaps(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.ConfigMap), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().ConfigMaps(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.Secret:
		existing, err = testContext.Client.CoreV1().Secrets(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.Secret), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().Secrets(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1beta1.CustomResourceDefinition:
		existing, err = testContext.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.Background(), newObject.(*v1beta1.CustomResourceDefinition), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *appsv1.DaemonSet:
		existing, err = testContext.Client.AppsV1().DaemonSets(newObject.GetNamespace()).Create(context.Background(), newObject.(*appsv1.DaemonSet), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.AppsV1().DaemonSets(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *appsv1.Deployment:
		existing, err = testContext.Client.AppsV1().Deployments(newObject.GetNamespace()).Create(context.Background(), newObject.(*appsv1.Deployment), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.AppsV1().Deployments(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v2beta1.HorizontalPodAutoscaler:
		existing, err = testContext.Client.AutoscalingV2beta1().HorizontalPodAutoscalers(newObject.GetNamespace()).Create(context.Background(), newObject.(*v2beta1.HorizontalPodAutoscaler), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.AutoscalingV2beta1().HorizontalPodAutoscalers(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *admissionregistrationv1beta1.MutatingWebhookConfiguration:
		existing, err = testContext.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), newObject.(*admissionregistrationv1beta1.MutatingWebhookConfiguration), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *policyv1beta1.PodDisruptionBudget:
		existing, err = testContext.Client.PolicyV1beta1().PodDisruptionBudgets(newObject.GetNamespace()).Create(context.Background(), newObject.(*policyv1beta1.PodDisruptionBudget), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.PolicyV1beta1().PodDisruptionBudgets(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
		// In case of PodDisruptionBudget resources `patchStrategy:"replace"` results in a constant diff from kubernetes version 1.21
		// The IgnorePDBSelector CalculateOption can only help in this situation if either the current or modified object contains gvk information, this is why it's set here explicitly
		te, _ := meta.TypeAccessor(existing)
		te.SetKind("PodDisruptionBudget")
		te.SetAPIVersion("policy/v1beta1")
	case *v1.PersistentVolumeClaim:
		existing, err = testContext.Client.CoreV1().PersistentVolumeClaims(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.PersistentVolumeClaim), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().PersistentVolumeClaims(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.ServiceAccount:
		existing, err = testContext.Client.CoreV1().ServiceAccounts(newObject.GetNamespace()).Create(context.Background(), newObject.(*v1.ServiceAccount), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().ServiceAccounts(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *unstructured.Unstructured:
		existing, err = testContext.DynamicClient.Resource(*testItem.gvr).Namespace(testContext.Namespace).
			Create(context.Background(), newObject.(*unstructured.Unstructured), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.DynamicClient.Resource(*testItem.gvr).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *v1.Node:
		existing, err = testContext.Client.CoreV1().Nodes().Create(context.Background(), newObject.(*v1.Node), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.CoreV1().Nodes().Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	case *appsv1.StatefulSet:
		existing, err = testContext.Client.AppsV1().StatefulSets(newObject.GetNamespace()).Create(context.Background(), newObject.(*appsv1.StatefulSet), metav1.CreateOptions{})
		if err != nil {
			return errors.WrapWithDetails(err, "failed to create object", "object", newObject)
		}
		defer func() {
			err = testContext.Client.AppsV1().StatefulSets(newObject.GetNamespace()).Delete(context.Background(), existing.GetName(), deleteOptions)
			if err != nil {
				log.Printf("Failed to remove object %s %+v", existing.GetName(), err)
			}
		}()
	}

	if testItem.remoteChange != nil {
		testItem.remoteChange(existing)
	}

	if testItem.localChange != nil {
		testItem.localChange(newObject)
	}

	patchResult, err := patch.DefaultPatchMaker.Calculate(existing.(runtime.Object), newObject.(runtime.Object), opts...)
	if err != nil {
		return err
	}

	matched := patchResult.IsEmpty()

	if testItem.shouldMatch && !matched {
		return errors.WithDetails(errors.New("Objects did not match"), "patch", patchResult)
	}

	if !testItem.shouldMatch && matched {
		return errors.WithDetails(errors.New("Objects matched although they should not"), "patch", patchResult)
	}

	return nil
}

func standardObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: "test-",
		Namespace:    testContext.Namespace,
	}
}

func metaWithLabels(labels map[string]string) metav1.ObjectMeta {
	meta := standardObjectMeta()
	meta.Labels = labels
	return meta
}
