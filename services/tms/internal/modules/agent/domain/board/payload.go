package board

import (
	"github.com/emoss08/trenova/shared/jsonutils"
)

type Payload struct {
	Board    Board    `json:"board"`
	Place    Place    `json:"place"`
	Snapshot Snapshot `json:"snapshot"`
	Author   string   `json:"author,omitempty"`
	Sizes    []int    `json:"sizes,omitempty"`
}

func SizesOf(filled *Filled) []int {
	sizes := make([]int, 0, len(filled.Blocks))
	for idx := range filled.Blocks {
		block := &filled.Blocks[idx]
		size := len(block.Rows) + len(block.Checks) + len(block.Events) + len(block.Work)
		if block.Metric != nil && block.Metric.Read != nil {
			read := block.Metric.Read
			size += len(read.Log) + len(read.Checks)
			if read.Table != nil {
				size += len(read.Table.Rows)
			}
			if read.Grid != nil {
				size += len(read.Grid.Rows)
			}
			if read.Chart != nil {
				size += len(read.Chart.Rows)
			}
			if read.Split != nil {
				size += len(read.Split.Parts)
			}
		}
		if block.Table != nil {
			size += len(block.Table.Table.Rows)
		}
		sizes = append(sizes, size)
	}

	return sizes
}

func NewPayload(layout *Board, filled *Filled, author string) *Payload {
	payload := &Payload{
		Board:    *layout,
		Snapshot: filled.Snapshot(),
		Author:   author,
		Sizes:    SizesOf(filled),
	}
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
