package patch

import (
	"reflect"
	"strings"

	"emperror.dev/errors"
	json "github.com/json-iterator/go"
)

func IgnorePDBSelector() CalculateOption {
	return func(current, modified []byte) ([]byte, []byte, error) {
		currentResource := map[string]interface{}{}
		if err := json.Unmarshal(current, &currentResource); err != nil {
			return []byte{}, []byte{}, errors.Wrap(err, "could not unmarshal byte sequence for current")
		}

		modifiedResource := map[string]interface{}{}
		if err := json.Unmarshal(modified, &modifiedResource); err != nil {
			return []byte{}, []byte{}, errors.Wrap(err, "could not unmarshal byte sequence for modified")
		}

		if isPDB(currentResource) || isPDB(modifiedResource) {
			if reflect.DeepEqual(getPDBSelector(currentResource), getPDBSelector(modifiedResource)) {
				var err error
				current, err = deletePDBSelector(currentResource)
				if err != nil {
					return nil, nil, errors.Wrap(err, "delete pdb selector from current")
				}
				modified, err = deletePDBSelector(modifiedResource)
				if err != nil {
					return nil, nil, errors.Wrap(err, "delete pdb selector from modified")
				}
			}
		}

		return current, modified, nil
	}
}

func isPDB(resource map[string]interface{}) bool {
	if av, ok := resource["apiVersion"].(string); ok {
		return strings.HasPrefix(av, "policy/") && resource["kind"] == "PodDisruptionBudget"
	}
	return false
}

func getPDBSelector(resource map[string]interface{}) interface{} {
	if spec, ok := resource["spec"]; ok {
		if spec, ok := spec.(map[string]interface{}); ok {
			if selector, ok := spec["selector"]; ok {
				return selector
			}
		}
	}
	return nil
}


func deletePDBSelector(resource map[string]interface{}) ([]byte, error) {
	if spec, ok := resource["spec"]; ok {
		if spec, ok := spec.(map[string]interface{}); ok {
			delete(spec, "selector")
		}
	}

	obj, err := json.ConfigCompatibleWithStandardLibrary.Marshal(resource)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not marshal byte sequence")
	}

	return obj, nil
}