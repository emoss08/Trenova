package board

import (
	"github.com/emoss08/trenova/shared/jsonutils"
)

type Payload struct {
	Board    Board    `json:"board"`
	Snapshot Snapshot `json:"snapshot"`
	Author   string   `json:"author,omitempty"`
}

func (p *Payload) Map() (map[string]any, error) {
	out := map[string]any{}
	if err := jsonutils.Convert(p, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func PayloadOf(raw map[string]any) (*Payload, error) {
	payload := &Payload{}
	if err := jsonutils.Convert(raw, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
