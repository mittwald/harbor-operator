package helper

type InterfaceHash []byte

type jsonPatchOp struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

type JSONPatch struct {
	ops []jsonPatchOp
}
