package helper

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (p *JSONPatch) Type() types.PatchType {
	return types.JSONPatchType
}

func (p *JSONPatch) Data(obj runtime.Object) ([]byte, error) {
	return json.Marshal(p.Ops)
}

func (p *JSONPatch) AddOp(op, path string, value interface{}) {
	p.Ops = append(p.Ops, jsonPatchOp{op, path, value})
}
