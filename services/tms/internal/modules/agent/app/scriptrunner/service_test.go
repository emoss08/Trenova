package scriptrunner

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var testRunner = sync.OnceValue(func() *Service { return NewRunner(zap.NewNop()) })

type fakeBroker struct {
	mu      sync.Mutex
	limits  codemode.Limits
	calls   []codemode.Call
	replies map[string]codemode.Reply
	noted   map[int]bool
}

func newBroker() *fakeBroker {
	return &fakeBroker{
		noted: map[int]bool{},
		replies: map[string]codemode.Reply{
			"shipment_list": {
				Status: codemode.CallOK,
				Data:   sonic.NoCopyRawMessage(`[{"id":"s1","customerId":"c1"},{"id":"s2","customerId":"c2"}]`),
			},
			"receivable_get_bank_receipt": {
				Status:  codemode.CallRefused,
				Message: "you may not read bank receipts",
			},
		},
	}
}

func (b *fakeBroker) Index() []codemode.Read {
	return []codemode.Read{
		{Name: "shipment_list", Description: "List shipments."},
		{Name: "customer_get", Description: "Get one customer."},
		{Name: "receivable_get_bank_receipt", Description: "One bank receipt."},
	}
}

func (b *fakeBroker) Bound(limits *codemode.Limits) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.limits = *limits
}

func (b *fakeBroker) Read(_ context.Context, call codemode.Call) codemode.Reply {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, call)
	if reply, ok := b.replies[call.Tool]; ok {
		return reply
	}
	id, _ := call.Arguments["id"].(string)
	data, _ := sonic.Marshal(map[string]any{"id": id, "name": "Customer " + id})
	return codemode.Reply{Status: codemode.CallOK, Data: data}
}

func (b *fakeBroker) NoteReply(seq int, _ int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.noted[seq] = true
}

func (b *fakeBroker) Summary() codemode.Summary {
	return codemode.Summary{}
}

func run(t *testing.T, code string, broker *fakeBroker) *scriptports.Outcome {
	t.Helper()
	outcome, err := testRunner().Run(t.Context(), &scriptports.Request{
		Code:     code,
		Timezone: "America/Chicago",
		Broker:   broker,
	})
	require.NoError(t, err)
	return outcome
}

func TestAScriptJoinsReadsAndPrints(t *testing.T) {
	t.Parallel()

	broker := newBroker()
	outcome := run(t, `
loads = trenova.shipment.list(status="Delayed")
customers = trenova.gather(*[trenova.customer.get.defer(id=l["customerId"]) for l in loads])
for load, customer in zip(loads, customers):
    print(load["id"], customer["name"])
`, broker)

	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)
	assert.Equal(t, "s1 Customer c1\ns2 Customer c2\n", outcome.Output)
	assert.Equal(t, codemode.DefaultLimits(), broker.limits)
	assert.Equal(t, codemode.DefaultLimits(), outcome.Limits)
}

func TestGatheredReadsAreNumberedInTheOrderTheScriptAsked(t *testing.T) {
	t.Parallel()

	broker := newBroker()
	outcome := run(t, `
trenova.gather(*[trenova.customer.get.defer(id=str(i)) for i in range(12)])
`, broker)
	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)

	require.Len(t, broker.calls, 12)
	bySeq := map[int]string{}
	for _, call := range broker.calls {
		bySeq[call.Seq] = call.Arguments["id"].(string)
		assert.True(t, broker.noted[call.Seq], "seq %d's reply time is noted", call.Seq)
	}
	for seq := 1; seq <= 12; seq++ {
		assert.Equal(t, strconv.Itoa(seq-1), bySeq[seq])
	}
}

func TestARefusedReadRaisesRefusedWithItsMessage(t *testing.T) {
	t.Parallel()

	outcome := run(t, `
try:
    trenova.receivable.get_bank_receipt(id="r1")
except trenova.Refused as refused:
    print(refused)
`, newBroker())

	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)
	assert.Equal(t,
		"receivable_get_bank_receipt: refused: you may not read bank receipts\n", outcome.Output)
}

func TestArgumentsPastTheFrameLimitAreRefusedWithoutAReadAndNamed(t *testing.T) {
	t.Parallel()

	broker := newBroker()
	outcome := run(t, `
try:
    trenova.customer.get(id="x" * 70000)
except trenova.Refused as refused:
    print(refused)
`, broker)

	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)
	assert.Equal(t, "customer_get: invalid: the call's arguments are too large\n", outcome.Output)
	assert.Empty(t, broker.calls)
	assert.Equal(t, codemode.LimitFrame, outcome.Limit)
}

func TestTheResultComesBackOnce(t *testing.T) {
	t.Parallel()

	outcome := run(t, "trenova.result({'total': 2, 'names': ['a', 'b']})\n", newBroker())

	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)
	assert.JSONEq(t, `{"total": 2, "names": ["a", "b"]}`, string(outcome.Result))
}

func TestAResultPastItsLimitIsDroppedAndNamed(t *testing.T) {
	t.Parallel()

	outcome := run(t, "trenova.result('x' * 70000)\n", newBroker())

	assert.Equal(t, scriptports.Errored, outcome.Status)
	assert.Empty(t, outcome.Result)
	assert.Equal(t, codemode.LimitResult, outcome.Limit)
}

func TestAnErrorComesBackWithTheScriptsTraceback(t *testing.T) {
	t.Parallel()

	outcome := run(t, "rows = []\nrows[3]\n", newBroker())

	assert.Equal(t, scriptports.Errored, outcome.Status)
	require.NotNil(t, outcome.Error)
	assert.Equal(t, "IndexError", outcome.Error.Name)
	assert.Contains(t, outcome.Error.Traceback, `File "script.py", line 2`)
}

func TestTodayIsTheOrganizationsDay(t *testing.T) {
	t.Parallel()

	outcome := run(t, "import datetime\nprint(datetime.datetime.now().astimezone().tzname())\n", newBroker())

	require.Equal(t, scriptports.Succeeded, outcome.Status, "%+v", outcome.Error)
	assert.Contains(t, []string{"CDT\n", "CST\n"}, outcome.Output)
}

func TestScriptsTheRunnerRefusesBeforeRunning(t *testing.T) {
	t.Parallel()

	for name, req := range map[string]*scriptports.Request{
		"empty":     {Code: "  ", Broker: newBroker()},
		"too long":  {Code: strings.Repeat("x", codemode.MaxCodeChars+1), Broker: newBroker()},
		"no broker": {Code: "print(1)"},
		"too many inputs": {Code: "print(1)", Broker: newBroker(), Inputs: func() map[string]any {
			inputs := map[string]any{}
			for i := range codemode.MaxInputKeys + 1 {
				inputs[strconv.Itoa(i)] = "v"
			}
			return inputs
		}()},
	} {
		_, err := testRunner().Run(t.Context(), req)
		assert.True(t, errortypes.IsBusinessError(err), "%s: %v", name, err)
	}
}
