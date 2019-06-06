package objectmatcher

import (
	"encoding/json"
	"fmt"

	"github.com/goph/emperror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

var DefaultPatchMaker = &PatchMaker{
	annotator: DefaultAnnotator,
}

type PatchMaker struct {
	annotator *Annotator
}

func (p *PatchMaker) Calculate(currentObject, modifiedObject runtime.Object) (*PatchResult, error) {
	current, err := json.Marshal(currentObject)
	if err != nil {
		return nil, emperror.Wrap(err, "Failed to convert current object to byte sequence")
	}

	modified, err := json.Marshal(modifiedObject)
	if err != nil {
		return nil, emperror.Wrap(err, "Failed to convert current object to byte sequence")
	}

	modified, _, err = DeleteNullInJson(modified)
	if err != nil {
		return nil, emperror.Wrap(err, "Failed to delete null from modified object")
	}

	original, err := DefaultAnnotator.GetOriginalConfiguration(currentObject)
	if err != nil {
		return nil, emperror.Wrap(err, "Failed to get original configuration")
	}

	var patch []byte

	switch currentObject.(type) {
	default:
		lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(modifiedObject)
		if err != nil {
			return nil, emperror.WrapWith(err, "Failed to lookup patch meta", "current object", currentObject)
		}
		patch, err = strategicpatch.CreateThreeWayMergePatch(original, modified, current, lookupPatchMeta, true)
		if err != nil {
			return nil, emperror.Wrap(err, "Failed to generate strategic merge patch")
		}
	case *unstructured.Unstructured:
		patch, err = jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current)
		if err != nil {
			return nil, emperror.Wrap(err, "Failed to generate merge patch")
		}
	}

	return &PatchResult{
		Patch:    patch,
		Current:  current,
		Modified: modified,
		Original: original,
	}, nil
}

type PatchResult struct {
	Patch    []byte
	Current  []byte
	Modified []byte
	Original []byte
}

func (p *PatchResult) IsUnmodified() bool {
	return string(p.Patch) == "{}"
}

func (p *PatchResult) String() string {
	return fmt.Sprintf("\nPatch: %s \nCurrent: %s\nModified: %s\nOriginal: %s\n", p.Patch, p.Current, p.Modified, p.Original)
}
