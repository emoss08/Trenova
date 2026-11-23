package board

import (
	"github.com/emoss08/trenova/shared/jsonutils"
)

type Payload struct {
	Board    Board    `json:"board"`
	Place    Place    `json:"place"`
	Snapshot Snapshot `json:"snapshot"`
	Author   string   `json:"author,omitempty"`
}

func NewPayload(layout *Board, snapshot Snapshot, author string) *Payload {
	payload := &Payload{Board: *layout, Snapshot: snapshot, Author: author}
	payload.Board.Place = layout.Placed()
	payload.Place = payload.Board.Place

	return payload
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
	payload.Board.Place = payload.Board.Placed()
	payload.Place = payload.Board.Place

	return payload, nil
}
