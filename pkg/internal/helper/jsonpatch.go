package helper

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type jsonPatchOp struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

type JSONPatch struct {
	ops []jsonPatchOp
}

func (p *JSONPatch) Type() types.PatchType {
	return types.JSONPatchType
}

func (p *JSONPatch) Data(obj runtime.Object) ([]byte, error) {
	return json.Marshal(p.ops)
}

func (p *JSONPatch) AddOp(op, path string, value interface{}) {
	p.ops = append(p.ops, jsonPatchOp{op, path, value})
}
