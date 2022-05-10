package helper

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Patch = &JSONPatch{}

type jsonPatchOp struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

type JSONPatch struct {
	Ops []jsonPatchOp
}

func (p *JSONPatch) Type() types.PatchType {
	return types.JSONPatchType
}

func (p *JSONPatch) Data(obj client.Object) ([]byte, error) {
	return json.Marshal(p.Ops)
}

func (p *JSONPatch) AddOp(op, path string, value interface{}) {
	p.Ops = append(p.Ops, jsonPatchOp{op, path, value})
}
