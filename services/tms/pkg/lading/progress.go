package lading

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonutils"
)

type flowProgress struct {
	Sends     uint64              `json:"sends"`
	Documents map[string]struct{} `json:"documents"`
	Load      LoadState           `json:"load"`
	Pending   []struct{}          `json:"pending"`
	Batches   []struct{}          `json:"batches"`
}

func FlowStarted(state jsonutils.RawJSON) bool {
	if len(state) == 0 {
		return false
	}
	var progress flowProgress
	if err := sonic.Unmarshal(state, &progress); err != nil {
		return true
	}
	return progress.Sends > 0 ||
		len(progress.Documents) > 0 ||
		progress.Load.Exists ||
		len(progress.Pending) > 0 ||
		len(progress.Batches) > 0
}
