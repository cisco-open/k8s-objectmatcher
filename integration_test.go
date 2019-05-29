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
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    testContext.Namespace,
				},
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    testContext.Namespace,
				},
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
		NewTestMatch("pod matches when there is a remote change on a field (Spec.Hostname) that DOES NOT EXIST in the local object",
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    testContext.Namespace,
				},
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
				},
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    testContext.Namespace,
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: testContext.Namespace,
				},
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
	}
	for _, test := range tests {
		err := testMatchOnObjectv2(test)
		if err != nil {
			t.Fatalf("Test %s failed: %s %s", test.name, err, emperror.Context(err))
		}
	}
}
