package types

import "github.com/bytedance/sonic"

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ProblemDetail struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance,omitempty"`
	InvalidParams []InvalidParam `json:"invalid-params,omitempty"`
}

func (p *ProblemDetail) Error() string {
	errBytes, _ := sonic.Marshal(p)
	return string(errBytes)
}
