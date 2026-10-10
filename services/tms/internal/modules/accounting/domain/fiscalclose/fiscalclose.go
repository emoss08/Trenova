package fiscalclose

import "github.com/emoss08/trenova/pkg/errortypes"

type Blocker struct {
	Field    string               `json:"field"`
	Code     errortypes.ErrorCode `json:"code"`
	Message  string               `json:"message"`
	Category string               `json:"category"`
}

type Result struct {
	CanClose bool       `json:"canClose"`
	Blockers []*Blocker `json:"blockers"`
}
