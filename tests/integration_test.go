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
	"log"
	"os"
	"strings"
	"testing"

	"emperror.dev/errors"
	admregv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	v1beta12 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
					Volumes: []v1.Volume{
						{
							Name: "empty",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
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
		NewTestDiff("pod does not match when a field should be removed only if it existed before",
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
		// This case did not work with the legacy version
		NewTestDiff("pod does not match if we try to remove a field",
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
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
								{
									Namespaces:  []string{testContext.Namespace},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			}).
			withLocalChange(func(i interface{}) {
				pod := i.(*v1.Pod)
				pod.Spec.Affinity = nil
			}),
		NewTestMatch("secret matches with original",
			&v1.Secret{
				ObjectMeta: standardObjectMeta(),
				Data: map[string][]byte{
					"key": []byte("secretValue"),
				},
			}),
		NewTestMatch("serviceaccount matches with original",
			&v1.ServiceAccount{
				ObjectMeta: standardObjectMeta(),
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
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
		NewTestDiff("service with named port diffs with existing",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       "http",
							Protocol:   v1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromString("http"),
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: v1.ServiceTypeLoadBalancer,
				},
			}).
			withLocalChange(func(a interface{}) {
				b := a.(*v1.Service)
				b.Spec.Ports[0].TargetPort = intstr.FromString("https")
			}),
		NewTestMatch("service with named port matches with original",
			&v1.Service{
				ObjectMeta: standardObjectMeta(),
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       "http",
							Protocol:   v1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromString("http"),
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
		NewTestDiff("deployment does not match when replicas changes",
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
			}).
			withLocalChange(func(i interface{}) {
				var replicas int32

				pod := i.(*appsv1.Deployment)
				pod.Spec.Replicas = &replicas
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
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
				},
				Webhooks: []admregv1beta1.MutatingWebhook{
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
									APIGroups:   []string{"", "apps"},
									APIVersions: []string{"*"},
									Scope:       scopeRef(admregv1beta1.AllScopes),
								},
							},
						},
					},
				},
			}),
		NewTestMatch("pdb match",
			&v1beta12.PodDisruptionBudget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta12.SchemeGroupVersion.String(),
					Kind:       "PodDisruptionBudget",
				},
				ObjectMeta: standardObjectMeta(),
				Spec: v1beta12.PodDisruptionBudgetSpec{
					MinAvailable: intstrRef(intstr.FromInt(1)),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "example",
						},
					},
				},
			}),
		NewTestDiff("pdb diff on matchlabels",
			&v1beta12.PodDisruptionBudget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta12.SchemeGroupVersion.String(),
					Kind:       "PodDisruptionBudget",
				},
				ObjectMeta: standardObjectMeta(),
				Spec: v1beta12.PodDisruptionBudgetSpec{
					MinAvailable: intstrRef(intstr.FromInt(1)),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "example",
						},
					},
				},
			}).
			withRemoteChange(func(i interface{}) {
				pdb := i.(*v1beta12.PodDisruptionBudget)
				pdb.Spec.Selector.MatchLabels = map[string]string{
					"app": "example2",
				}
			}),
		NewTestMatch("pdb match even though status changes",
			&v1beta12.PodDisruptionBudget{
				ObjectMeta: standardObjectMeta(),
				Spec: v1beta12.PodDisruptionBudgetSpec{
					MinAvailable: intstrRef(intstr.FromInt(1)),
				},
			}).
			withRemoteChange(func(i interface{}) {
				pdb := i.(*v1beta12.PodDisruptionBudget)
				pdb.Status.CurrentHealthy = 1
				pdb.Status.DesiredHealthy = 1
				pdb.Status.ExpectedPods = 1
				pdb.Status.ObservedGeneration = 1
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
		NewTestMatch("unstructured match", &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "value",
				},
			},
		}).withGroupVersionResource(&schema.GroupVersionResource{
			Version:  "v1",
			Resource: "serviceaccounts",
		}),
		NewTestMatch("node match",
			&v1.Node{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"},
				Spec: v1.NodeSpec{
					PodCIDR: "10.0.0.0/24",
				},
				// ignore due to already removed field
			}).withIgnoreVersions([]string{"v1.10"}),
		NewTestDiff("node diff for podcidr",
			&v1.Node{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"},
				Spec: v1.NodeSpec{
					PodCIDR: "10.0.0.0/24",
				},
			}).
			withLocalChange(func(i interface{}) {
				n := i.(*v1.Node)
				n.Spec.PodCIDR = "10.0.0.1/24"
				// ignore due to already removed field
			}).withIgnoreVersions([]string{"v1.10"}),
		NewTestMatch("statefulset match for volumeclaimtemplates",
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "test-", Namespace: "default"},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32ref(0),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"a": "b"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"a": "b"},
						},
						Spec: v1.PodSpec{},
					},
					VolumeClaimTemplates: []v1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "vault-raft",
							},
							Spec: v1.PersistentVolumeClaimSpec{
								AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
								Resources: v1.ResourceRequirements{
									Requests: map[v1.ResourceName]resource.Quantity{
										v1.ResourceStorage: resource.MustParse("2G"),
									},
								},
								VolumeMode: volumeModeRef(v1.PersistentVolumeFilesystem),
							},
							Status: v1.PersistentVolumeClaimStatus{
								Phase: "Pending",
							},
						},
					},
				},
			}),
		NewTestDiff("statefulset diff for template",
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "test-", Namespace: "default"},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32ref(0),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"a": "b"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"a": "b"},
						},
						Spec: v1.PodSpec{},
					},
				},
			},
		).withLocalChange(func(i interface{}) {
			n := i.(*appsv1.StatefulSet)
			n.Spec.Template.ObjectMeta.Labels = map[string]string{"c": "d"}
		}),
	}
	runAll(t, tests)
}

func runAll(t *testing.T, tests []*TestItem) {
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
		err := testMatchOnObject(test)
		if err != nil {
			if *failonerror {
				t.Fatalf("Test '%s' failed: %s %s", test.name, err, errors.GetDetails(err))
			} else {
				t.Errorf("Test '%s' failed: %s %s", test.name, err, errors.GetDetails(err))
			}
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

func scopeRef(scopeType admregv1beta1.ScopeType) *admregv1beta1.ScopeType {
	return &scopeType
}

func volumeModeRef(mode v1.PersistentVolumeMode) *v1.PersistentVolumeMode {
	return &mode
}
