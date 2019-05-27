package objectmatch

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/klogr"
)

func TestIntegration_Pod(t *testing.T) {

	if !*integration {
		t.Skip()
	}

	tests := []struct{
		pod *v1.Pod
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Namespace: testContext.Namespace,
				},
				Spec:       v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "test-container", Image:"test-image",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:             "test-volume",
									MountPath:        "/tmp/test",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "test-volume",
							VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/tmp/test"}},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		existingPod, err := testContext.Client.CoreV1().Pods(test.pod.Namespace).Create(test.pod)
		defer func() {
			testContext.Client.CoreV1().Pods("default").Delete(test.pod.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: new(int64),
			})
		}()

		if err != nil {
			t.Fatalf("Failed to create pod: %s", err.Error())
		}

		matched, err := NewPodMatcher(New(klogr.New())).Match(existingPod, test.pod)
		if err != nil {
			t.Fatalf("Failed to match objects: %s", err)
		}

		if !matched {
			t.Fatalf("Objects did not match")
		}
	}
}
