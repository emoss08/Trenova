package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestCarriesExternalContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		taint *agent.RunTaint
		want  bool
	}{
		{name: "no taint kept", taint: nil, want: false},
		{name: "clean thread", taint: &agent.RunTaint{}, want: false},
		{name: "a web read stays capped", taint: &agent.RunTaint{
			Seen: []agent.TaintSource{agent.TaintSourceWeb},
		}, want: true},
		{name: "pasted partner text stays capped", taint: &agent.RunTaint{
			Marks: []agent.TaintMark{{
				Source:   agent.TaintSourceAttachment,
				ToolName: "edi_inspect_x12",
			}},
		}, want: true},
		{name: "a driver's comment holds, it does not cap", taint: &agent.RunTaint{
			Marks: []agent.TaintMark{{
				Source:   agent.TaintSourceRecordNote,
				ToolName: "shipment_get",
			}},
		}, want: false},
		{name: "overflow may have dropped the mark", taint: &agent.RunTaint{
			Overflowed: true,
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CarriesExternalContent(tt.taint))
		})
	}
}
