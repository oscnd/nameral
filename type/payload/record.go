package payload

type RecordAddBody struct {
	Name  *string `json:"name"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

type RecordSetBody struct {
	Hash  *string `json:"hash"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

type RecordDeleteBody struct {
	Hash *string `json:"hash"`
}

type Record struct {
	Hash  *string `json:"hash"`
	Name  *string `json:"name"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
}
