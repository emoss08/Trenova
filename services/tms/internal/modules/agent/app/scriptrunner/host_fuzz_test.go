package scriptrunner

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/pkg/scriptvm"
)

func FuzzHostAnswersWhateverAScriptSends(f *testing.F) {
	for _, seed := range []struct {
		function string
		payload  string
	}{
		{"read", `{"tool":"shipment_list","args":{"status":"Delayed"}}`},
		{"read", `{"tool":"customer_get","args":{"id":"c1","nested":{"a":[1,2,{"b":null}]}}}`},
		{"read", `{"tool":"","args":null}`},
		{"read", `[1,2,3]`},
		{"read", `"just a string"`},
		{"gather", `[{"tool":"customer_get","args":{"id":"1"}},{"tool":"x"}]`},
		{"gather", `[null, 7, "x", {"tool":{"no":"pe"}}]`},
		{"gather", `{"not":"a list"}`},
		{"result", `{"total": 3}`},
		{"result", ``},
		{"result", `\xff\xfe`},
		{"unknown", `{}`},
	} {
		f.Add(seed.function, []byte(seed.payload))
	}

	f.Fuzz(func(t *testing.T, function string, payload []byte) {
		limits := codemode.DefaultLimits()
		h := newHost(newBroker(), &limits)

		reply, err := h.Call(t.Context(), scriptvm.Function(function), payload)
		switch scriptvm.Function(function) {
		case scriptvm.FunctionRead, scriptvm.FunctionResult:
			if err != nil {
				t.Fatalf("%s answered an error: %v", function, err)
			}
			var answer codemode.Reply
			if decodeErr := sonic.Unmarshal(reply, &answer); decodeErr != nil || answer.Status == "" {
				t.Fatalf("%s answered %q, not a reply", function, reply)
			}
		case scriptvm.FunctionGather:
			if err != nil {
				t.Fatalf("gather answered an error: %v", err)
			}
			var list []codemode.Reply
			if sonic.Unmarshal(reply, &list) != nil {
				var single codemode.Reply
				if decodeErr := sonic.Unmarshal(reply, &single); decodeErr != nil ||
					single.Status != codemode.CallInvalid {
					t.Fatalf("gather answered %q, neither a list nor a refusal", reply)
				}
				return
			}
			var calls []sonic.NoCopyRawMessage
			if sonic.Unmarshal(payload, &calls) == nil && len(list) != len(calls) {
				t.Fatalf("gather answered %d replies for %d calls", len(list), len(calls))
			}
		default:
			if err == nil {
				t.Fatalf("an unknown function %q was answered", function)
			}
		}
	})
}
