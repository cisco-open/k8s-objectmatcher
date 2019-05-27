package objectmatch

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/klogr"
)

func TestIntegration_Service(t *testing.T) {

	if !*integration {
		t.Skip()
	}

	tests := []struct{
		service *v1.Service
	}{
		{
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-service",
					Namespace: testContext.Namespace,
				},
				Spec: v1.ServiceSpec{
					Ports:                    []v1.ServicePort{
						{
							Name:       "http",
							Protocol:   v1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
					Selector:                 map[string]string{
						"app": "test",
					},
					Type:                     v1.ServiceTypeLoadBalancer,
				},
			},
		},
	}

	for _, test := range tests {

		existingService, err := testContext.Client.CoreV1().Services(test.service.Namespace).Create(test.service)
		defer func() {
			testContext.Client.CoreV1().Pods(test.service.Namespace).Delete(test.service.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: new(int64),
			})
		}()

		if err != nil {
			t.Fatalf("Failed to create pod: %s", err.Error())
		}


		matched, err := NewServiceMatcher(New(klogr.New())).Match(existingService, test.service)
		if err != nil {
			t.Fatalf("Failed to match objects: %s", err)
		}

		if !matched {
			t.Fatalf("Objects did not match")
		}
	}
}
