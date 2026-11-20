package lading

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonutils"
)

var (
	flowStateInstants = []string{"awaitingResponse", "responseDue", "answerDue"}
	flowStateDueLists = []string{"pending", "batches"}
)

func ShiftFlowState(state jsonutils.RawJSON, by int64) (jsonutils.RawJSON, error) {
	if len(state) == 0 || by == 0 {
		return state, nil
	}

	var fields map[string]jsonutils.RawJSON
	if err := sonic.Unmarshal(state, &fields); err != nil {
		return nil, fmt.Errorf("read the flow state: %w", err)
	}
	if fields == nil {
		return state, nil
	}

	for _, name := range flowStateInstants {
		shifted, err := shiftInstant(fields[name], by)
		if err != nil {
			return nil, fmt.Errorf("shift %s: %w", name, err)
		}
		if shifted != nil {
			fields[name] = shifted
		}
	}
	for _, name := range flowStateDueLists {
		shifted, err := shiftDueList(fields[name], by)
		if err != nil {
			return nil, fmt.Errorf("shift %s: %w", name, err)
		}
		if shifted != nil {
			fields[name] = shifted
		}
	}

	out, err := sonic.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("write the flow state: %w", err)
	}
	return out, nil
}

func shiftInstant(raw jsonutils.RawJSON, by int64) (jsonutils.RawJSON, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var at int64
	if err := sonic.Unmarshal(raw, &at); err != nil {
		return nil, err
	}
	return sonic.Marshal(at - by)
}

func shiftDueList(raw jsonutils.RawJSON, by int64) (jsonutils.RawJSON, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var items []map[string]jsonutils.RawJSON
	if err := sonic.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	for index := range items {
		shifted, err := shiftInstant(items[index]["due"], by)
		if err != nil {
			return nil, err
		}
		if shifted != nil {
			items[index]["due"] = shifted
		}
	}
	return sonic.Marshal(items)
}
