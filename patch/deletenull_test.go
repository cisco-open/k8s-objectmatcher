// Copyright © 2026 Banzai Cloud
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

package patch

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// statefulSetWithVCT builds a StatefulSet-shaped object whose sole
// volumeClaimTemplate entry can be customized by the given mutator, so
// callers can model the difference between a live (server-returned) object
// and a desired (user-authored) manifest.
func statefulSetWithVCT(mutate func(vct map[string]interface{})) map[string]interface{} {
	vct := map[string]interface{}{
		"spec": map[string]interface{}{
			"accessModes": []interface{}{"ReadWriteOnce"},
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"storage": "1Gi",
				},
			},
		},
	}
	mutate(vct)

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      "web",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"replicas": float64(1),
			"volumeClaimTemplates": []interface{}{
				vct,
			},
		},
	}
}

// Test_deleteVolumeClaimTemplateFields reproduces cisco-open/k8s-objectmatcher#51:
// a StatefulSet fetched from the cluster carries auto-populated ObjectMeta
// fields (namespace, creationTimestamp, etc.) on each volumeClaimTemplate
// entry that the user's desired manifest never set. Before the fix, only
// kind/apiVersion/status were normalized, so the leftover metadata caused
// deleteVolumeClaimTemplateFields to leave the two objects looking different.
func Test_deleteVolumeClaimTemplateFields(t *testing.T) {
	current := statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["kind"] = "PersistentVolumeClaim"
		vct["apiVersion"] = "v1"
		vct["metadata"] = map[string]interface{}{
			"name":              "data",
			"namespace":         "default",
			"creationTimestamp": nil,
			"resourceVersion":   "12345",
			"uid":               "abc-123",
		}
		vct["status"] = map[string]interface{}{
			"phase": "Bound",
		}
	})

	modified := statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["metadata"] = map[string]interface{}{
			"name": "data",
		}
	})

	currentBytes, err := deleteVolumeClaimTemplateFields(mustFromUnstructured(current))
	if err != nil {
		t.Fatalf("deleteVolumeClaimTemplateFields(current) error = %v", err)
	}

	modifiedBytes, err := deleteVolumeClaimTemplateFields(mustFromUnstructured(modified))
	if err != nil {
		t.Fatalf("deleteVolumeClaimTemplateFields(modified) error = %v", err)
	}

	gotCurrent := mustToUnstructured(currentBytes)
	gotModified := mustToUnstructured(modifiedBytes)

	if !reflect.DeepEqual(gotCurrent, gotModified) {
		t.Errorf("expected volumeClaimTemplates differing only in metadata to be normalized to the same value.\ncurrent:  %#v\nmodified: %#v", gotCurrent, gotModified)
	}
}

// Test_IgnoreVolumeClaimTemplateTypeMetaAndStatus_ignoresMetadata is the
// same issue #51 scenario, exercised through the exported CalculateOption
// that callers actually use.
func Test_IgnoreVolumeClaimTemplateTypeMetaAndStatus_ignoresMetadata(t *testing.T) {
	current := mustFromUnstructured(statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["kind"] = "PersistentVolumeClaim"
		vct["apiVersion"] = "v1"
		vct["metadata"] = map[string]interface{}{
			"name":              "data",
			"namespace":         "default",
			"creationTimestamp": nil,
		}
		vct["status"] = map[string]interface{}{
			"phase": "Bound",
		}
	}))

	modified := mustFromUnstructured(statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["metadata"] = map[string]interface{}{
			"name": "data",
		}
	}))

	opt := IgnoreVolumeClaimTemplateTypeMetaAndStatus()
	gotCurrent, gotModified, err := opt(current, modified)
	if err != nil {
		t.Fatalf("IgnoreVolumeClaimTemplateTypeMetaAndStatus() error = %v", err)
	}

	if !reflect.DeepEqual(mustToUnstructured(gotCurrent), mustToUnstructured(gotModified)) {
		t.Errorf("expected current and modified to be equivalent once volumeClaimTemplate metadata is ignored.\ncurrent:  %s\nmodified: %s", gotCurrent, gotModified)
	}
}

// Test_Calculate_ignoresVolumeClaimTemplateMetadata confirms the fix at the
// level callers use: PatchMaker.Calculate() should report no diff between a
// live StatefulSet and the desired one when they differ only in
// volumeClaimTemplate metadata (name/namespace), matching what
// IgnoreVolumeClaimTemplateTypeMetaAndStatus already promises for
// kind/apiVersion/status.
func Test_Calculate_ignoresVolumeClaimTemplateMetadata(t *testing.T) {
	current := &unstructured.Unstructured{Object: statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["kind"] = "PersistentVolumeClaim"
		vct["apiVersion"] = "v1"
		vct["metadata"] = map[string]interface{}{
			"name":              "data",
			"namespace":         "default",
			"creationTimestamp": nil,
		}
		vct["status"] = map[string]interface{}{
			"phase": "Bound",
		}
	})}

	modified := &unstructured.Unstructured{Object: statefulSetWithVCT(func(vct map[string]interface{}) {
		vct["metadata"] = map[string]interface{}{
			"name": "data",
		}
	})}

	result, err := DefaultPatchMaker.Calculate(current, modified,
		IgnoreStatusFields(),
		IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
	)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if !result.IsEmpty() {
		t.Errorf("expected empty patch for StatefulSets differing only in volumeClaimTemplate metadata, got patch: %s", result.Patch)
	}
}
