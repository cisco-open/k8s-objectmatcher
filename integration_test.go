/*
Copyright 2019 Banzai Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package objectmatch

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/goph/emperror"
	admregv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	v1beta12 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIntegration(t *testing.T) {

	if !*integration {
		t.Skip()
	}

	tests := []*TestItem{
		NewTestMatch("pod matches with original",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "test-container", Image: "test-image",
						},
					},
				},
			}),
		NewTestDiff("pod does not match when a slice item gets removed",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "test-container",
							Image:   "test-image",
							Command: []string{"1", "2"},
						},
					},
				},
			}).
			withLocalChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Containers[0].Command = []string{"1"}
			}),
		NewTestDiff("pod does not match when a slice item gets added",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "test-container",
							Image:   "test-image",
							Command: []string{"1", "2"},
						},
					},
				},
			}).
			withLocalChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Containers[0].Command = []string{"1", "2", "3"}
			}),
		NewTestDiff("pod does not match when a field shozuld be removed",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			}).
			withRemoteChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Containers[0].Command = []string{"1", "2", "3"}
			}),
		NewTestDiff("pod does not match when a field gets removed locally, but exists remotely",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "test-container",
							Image:   "test-image",
							Command: []string{"1", "2"},
						},
					},
				},
			}).
			withLocalChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Containers[0].Command = nil
			}),
		NewTestDiff("pod does not match when a field gets removed remotely, but exists locally",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "test-container",
							Image:   "test-image",
							Command: []string{"1", "2"},
						},
					},
				},
			}).
			withRemoteChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Containers[0].Command = nil
			}),
		NewTestDiff("pod does not match when there is a remote change on a field (Spec.Hostname) that EXISTS in the local object",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Hostname: "original",
					Containers: []v1.Container{
						{
							Name: "test-container", Image: "test-image",
						},
					},
				},
			}).
			withRemoteChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Hostname = "changed on the server"
			}),
		NewTestDiff("pod does not match when there is a local change",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "test-container", Image: "test-image",
						},
					},
				},
			}).
			withLocalChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Hostname = "changed locally"
			}),
		NewTestMatch("pod matches when there is a remote change on a field (Spec.Hostname) that DOES NOT EXIST in the local object",
			&v1.Pod{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "test-container", Image: "test-image",
						},
					},
				},
			}).
			withRemoteChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Hostname = "changed on the server"
			}),
		NewTestMatch("serviceaccount matches with original",
			&v1.ServiceAccount{
				ObjectMeta: standardObjectMeta(),
			}),
		NewTestMatch("clusterrole matches with original",
			&rbacv1.ClusterRole{
				ObjectMeta: standardObjectMeta(),
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get"},
						APIGroups: []string{"*"},
						Resources: []string{"configmaps"},
					},
				},
			}),
		NewTestMatch("role matches with original",
			&rbacv1.Role{
				ObjectMeta: standardObjectMeta(),
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get"},
						APIGroups: []string{"*"},
						Resources: []string{"configmaps"},
					},
				},
			}),
		NewTestMatch("clusterrolebinding matches with original",
			&rbacv1.ClusterRoleBinding{
				ObjectMeta: standardObjectMeta(),
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						APIGroup:  "",
						Name:      "test",
						Namespace: "test",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "test",
				},
			}),
		NewTestMatch("rolebinding matches with original",
			&rbacv1.RoleBinding{
				ObjectMeta: standardObjectMeta(),
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						APIGroup:  "",
						Name:      "test",
						Namespace: "test",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test",
				},
			}),
		NewTestMatch("service matches with original",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       "http",
							Protocol:   v1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: v1.ServiceTypeLoadBalancer,
				},
			}),
		NewTestMatch("service matches with original even if defaults are not set",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name: "http",
							Port: 80,
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: v1.ServiceTypeLoadBalancer,
				},
			}),
		NewTestMatch("service matches with original even if nodeport is set remotely",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name: "http",
							Port: 80,
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: v1.ServiceTypeLoadBalancer,
				},
			}).
			withRemoteChange(func(a interface{}) {
				b := a.(*v1.Service)
				b.Spec.Ports[0].NodePort = 32020
			}),
		NewTestMatch("service sometimes specifies nodeport locally as well",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:     "http",
							Port:     80,
							NodePort: 32020,
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: v1.ServiceTypeLoadBalancer,
				},
			}),
		NewTestMatch("configmap match",
			&v1.ConfigMap{
				ObjectMeta: standardObjectMeta(),
				Data: map[string]string{
					"test": "data",
				},
			}),
		NewTestMatch("crd match",
			&v1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tests.test.org",
				},
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "test.org",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:   "tests",
						Singular: "test",
						Kind:     "Test",
						ListKind: "Tests",
					},
					Scope: v1beta1.NamespaceScoped,
					Versions: []v1beta1.CustomResourceDefinitionVersion{
						{
							Name:    "v1",
							Served:  true,
							Storage: true,
						},
					},
				},
			}).withIgnoreVersions([]string{"v1.10"}),
		NewTestMatch("crd match for deprecated version spec",
			&v1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "btests.test.org",
				},
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "test.org",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:   "btests",
						Singular: "btest",
						Kind:     "Btest",
						ListKind: "Btests",
					},
					Scope:   v1beta1.NamespaceScoped,
					Version: "v1",
				},
			}),
		NewTestMatch("daemonset match",
			&appsv1.DaemonSet{
				ObjectMeta: standardObjectMeta(),
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metaWithLabels(map[string]string{
							"a": "b",
						}),
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container", Image: "test-image",
								},
							},
						},
					},
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
						Type: appsv1.OnDeleteDaemonSetStrategyType,
					},
				},
			}),
		NewTestMatch("deployment match",
			&appsv1.Deployment{
				ObjectMeta: standardObjectMeta(),
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metaWithLabels(map[string]string{
							"a": "b",
						}),
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "test-container", Image: "test-image",
								},
							},
						},
					},
				},
			}),
		NewTestMatch("hpa match",
			&v2beta1.HorizontalPodAutoscaler{
				ObjectMeta: standardObjectMeta(),
				Spec: v2beta1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: v2beta1.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "test",
						APIVersion: "apps/v1",
					},
					MinReplicas: int32ref(1),
					MaxReplicas: 2,
				},
			}),
		NewTestMatch("mutating webhook configuration",
			&admregv1beta1.MutatingWebhookConfiguration{
				ObjectMeta: standardObjectMeta(),
				Webhooks: []admregv1beta1.Webhook{
					{
						Name: "a.b.c",
						ClientConfig: admregv1beta1.WebhookClientConfig{
							Service: &admregv1beta1.ServiceReference{
								Name:      "test",
								Namespace: testContext.Namespace,
								Path:      strRef("/inject"),
							},
							CABundle: nil,
						},
						Rules: []admregv1beta1.RuleWithOperations{
							{
								Operations: []admregv1beta1.OperationType{
									admregv1beta1.Create,
								},
								Rule: admregv1beta1.Rule{
									Resources:   []string{"pods"},
									APIGroups:   []string{""},
									APIVersions: []string{"*"},
								},
							},
						},
					},
				},
			}),
		NewTestMatch("pdb match",
			&v1beta12.PodDisruptionBudget{
				ObjectMeta: standardObjectMeta(),
				Spec: v1beta12.PodDisruptionBudgetSpec{
					MinAvailable: intstrRef(intstr.FromInt(1)),
				},
			}),
		NewTestMatch("pvc match",
			&v1.PersistentVolumeClaim{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": resource.MustParse("1G"),
						},
					},
				},
			}),
		//NewTestMatch("unstructured match", &unstructured.Unstructured{
		//	Object: map[string]interface{}{
		//		"metadata": map[string]interface{}{
		//			"name": "value",
		//		},
		//	},
		//}).withGroupVersionResource(&schema.GroupVersionResource{
		//	Version:  "v1",
		//	Resource: "serviceaccounts",
		//}),
	}
	for _, test := range tests {
		serverVersion := os.Getenv("K8S_VERSION")
		if test.ignoreVersions != nil {
			if serverVersion == "" {
				t.Errorf("Ignore list defined as %s for %s, but server version is not set", test.ignoreVersions, test.name)
				continue
			} else {
				if versionPrefixMatch(serverVersion, test.ignoreVersions) {
					if testing.Verbose() {
						log.Printf("# skipped %s due to server version", test.name)
					}
					continue
				}
			}
		}
		if testing.Verbose() {
			log.Printf("> %s", test.name)
		}
		err := testMatchOnObjectv2(test)
		if err != nil {
			t.Errorf("Test '%s' failed: %s %s", test.name, err, emperror.Context(err))
		}
	}
}

func int32ref(x int32) *int32 {
	return &x
}

func strRef(s string) *string {
	return &s
}

func intstrRef(i intstr.IntOrString) *intstr.IntOrString {
	return &i
}

func versionPrefixMatch(s string, l []string) bool {
	for _, i := range l {
		if strings.HasPrefix(s, i) {
			return true
		}
	}
	return false
}
