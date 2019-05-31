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

package objectmatch

import (
	"encoding/json"

	"github.com/goph/emperror"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

type pvcMatcher struct {
	objectMatcher ObjectMatcher
}

func NewPvcMatcher(objectMatcher ObjectMatcher) *pvcMatcher {
	return &pvcMatcher{
		objectMatcher: objectMatcher,
	}
}

// Match compares two corev1.PersistentVolumeClaim objects
func (m pvcMatcher) Match(oldOrig, newOrig *corev1.PersistentVolumeClaim) (bool, error) {
	old := oldOrig.DeepCopy()
	new := newOrig.DeepCopy()

	v1.SetObjectDefaults_PersistentVolumeClaim(new)

	type Pvc struct {
		ObjectMeta
		Spec corev1.PersistentVolumeClaimSpec
	}

	if old.Spec.VolumeMode == nil && newOrig.Spec.VolumeMode == nil {
		new.Spec.VolumeMode = nil
	}

	delete(new.ObjectMeta.Annotations, "volume.beta.kubernetes.io/storage-provisioner")
	delete(new.ObjectMeta.Annotations, "pv.kubernetes.io/bind-completed")
	delete(new.ObjectMeta.Annotations, "pv.kubernetes.io/bound-by-controller")

	old.Spec.VolumeName = new.Spec.VolumeName

	oldData, err := json.Marshal(Pvc{
		ObjectMeta: m.objectMatcher.GetObjectMeta(old.ObjectMeta),
		Spec:       old.Spec,
	})

	if err != nil {
		return false, emperror.WrapWith(err, "could not marshal old object", "name", old.Name)
	}
	newObject := Pvc{
		ObjectMeta: m.objectMatcher.GetObjectMeta(new.ObjectMeta),
		Spec:       new.Spec,
	}
	newData, err := json.Marshal(newObject)
	if err != nil {
		return false, emperror.WrapWith(err, "could not marshal new object", "name", new.Name)
	}

	matched, err := m.objectMatcher.MatchJSON(oldData, newData, newObject)
	if err != nil {
		return false, emperror.WrapWith(err, "could not match objects", "name", new.Name)
	}

	return matched, nil
}
