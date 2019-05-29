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
	"testing"

	"github.com/goph/emperror"
	admregv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	}
	for _, test := range tests {
		err := testMatchOnObjectv2(test)
		if err != nil {
			t.Fatalf("Test %s failed: %s %s", test.name, err, emperror.Context(err))
		}
	}
}

func int32ref(x int32) *int32 {
	return &x
}

func strRef(s string) *string {
	return &s
}
