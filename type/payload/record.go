package payload

type RecordAddBody struct {
	Name  *string `json:"name"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

type RecordSetBody struct {
	No    *uint64 `json:"no"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
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
