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
	"strings"

	v1 "k8s.io/kubernetes/pkg/apis/core/v1"

	"github.com/goph/emperror"
	corev1 "k8s.io/api/core/v1"
)

type podMatcher struct {
	objectMatcher ObjectMatcher
}

func NewPodMatcher(objectMatcher ObjectMatcher) *podMatcher {
	return &podMatcher{
		objectMatcher: objectMatcher,
	}
}

// Match compares two corev1.Pod objects
func (m podMatcher) Match(oldOrig, newOrig *corev1.Pod) (bool, error) {
	old := oldOrig.DeepCopy()

	new := newOrig.DeepCopy()
	v1.SetObjectDefaults_Pod(new)

	type Pod struct {
		ObjectMeta
		Spec corev1.PodSpec
	}

	generatedTokenName := ""
	tmpVolume := []corev1.Volume{}
	for _, volume := range old.Spec.Volumes {
		if !strings.HasPrefix(volume.Name, old.Spec.ServiceAccountName+"-token-") {
			tmpVolume = append(tmpVolume, volume)
		} else {
			generatedTokenName = volume.Name
		}
	}
	old.Spec.Volumes = tmpVolume

	tmpInitContainers := []corev1.Container{}
	for _, initContainer := range old.Spec.InitContainers {
		tmpVolumeMounts := []corev1.VolumeMount{}
		for _, volumeMount := range initContainer.VolumeMounts {
			if volumeMount.Name != generatedTokenName {
				tmpVolumeMounts = append(tmpVolumeMounts, volumeMount)
			}
		}
		initContainer.VolumeMounts = tmpVolumeMounts
		tmpInitContainers = append(tmpInitContainers, initContainer)
	}
	old.Spec.InitContainers = tmpInitContainers

	tmpContainers := []corev1.Container{}
	for _, container := range old.Spec.Containers {
		tmpVolumeMounts := []corev1.VolumeMount{}
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name != generatedTokenName {
				tmpVolumeMounts = append(tmpVolumeMounts, volumeMount)
			}
		}
		container.VolumeMounts = tmpVolumeMounts
		tmpContainers = append(tmpContainers, container)
	}
	old.Spec.Containers = tmpContainers

	oldData, err := json.Marshal(Pod{
		ObjectMeta: m.objectMatcher.GetObjectMeta(old.ObjectMeta),
		Spec:       old.Spec,
	})

	if err != nil {
		return false, emperror.WrapWith(err, "could not marshal old object", "name", old.Name)
	}
	newObject := Pod{
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
