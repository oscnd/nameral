package payload

type RecordSetBody struct {
	Name  *string `json:"name"`
	Type  *string `json:"type"`
	Value *string `json:"records"`
}

type RecordDeleteBody struct {
	No *uint64 `json:"no"`
}

type Record struct {
	No     *uint64   `json:"no"`
	Name   *string   `json:"name"`
	Type   *string   `json:"type"`
	Values []*string `json:"values"`
}
