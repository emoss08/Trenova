package boardhint_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestEveryFormSaysWhatThePersonSees(t *testing.T) {
	t.Parallel()

	for _, form := range boardcatalog.AllForms() {
		assert.NotEmpty(
			t,
			boardhint.Shows(form),
			"form %s says nothing a person needs to see",
			form,
		)
	}
	for _, metric := range boardregistry.Catalog().Metrics() {
		assert.NotEmpty(t, boardhint.Line(&boardhint.Show{Metric: metric.Name}), metric.Name)
	}
}

func TestLineDrawsTheMetricAndItsFiguresFromTheCatalog(t *testing.T) {
	t.Parallel()

	line := boardhint.Line(&boardhint.Show{Metric: operationsboard.MetricOnTimeByWeek})

	assert.Contains(t, line, "When the person needs to see how it moved over time")
	assert.Contains(t, line, `show_answer {"title": "…", "answer": "… {average:onTimeByWeek} …", `+
		`"groups": [{"title": "…", "metric": "onTimeByWeek", "period": "…"}]}`)
	assert.Contains(t, line, "Its period is the one the question asks about: thisQuarter "+
		"(the default), lastQuarter.")
	assert.Contains(t, line, "{lowest:onTimeByWeek} (on time, the lowest week)")
	assert.Contains(t, line, "at most 6 in all")
	assert.Contains(t, line, "{count} does not apply")
}

func TestLineAddsTheRecordsGroupAndTheSubject(t *testing.T) {
	t.Parallel()

	period := pulid.ID("fp_01JHINTPERIOD0000000000000")
	line := boardhint.Line(&boardhint.Show{
		Metric: operationsboard.MetricOnTimeByWeek,
		For:    period,
		Period: string(boardcatalog.PeriodLastQuarter),
		Records: &boardhint.Records{
			Words: "the late loads", Measure: "lateness", From: "shipment_list",
		},
	})

	assert.Contains(t, line, `"for": "fp_01JHINTPERIOD0000000000000", "period": "lastQuarter"}`)
	assert.Contains(t, line, `"measure": "lateness", "groups": [`)
	assert.Contains(t, line, `{"title": "…", "records": ["…"]}]}`)
	assert.Contains(t, line, "The second group holds the late loads, read with shipment_list")
	assert.NotContains(t, line, "{count} does not apply")
	assert.Empty(t, boardhint.Line(&boardhint.Show{Metric: "noSuchMetric"}))
}

func TestBriefNamesTheGroupAndItsSubject(t *testing.T) {
	t.Parallel()

	for _, metric := range boardregistry.Catalog().Metrics() {
		brief := boardhint.Brief(&boardhint.Show{Metric: metric.Name})
		assert.Contains(t, brief, `"metric": "`+metric.Name+`"`)
		assert.LessOrEqual(t, len(brief), 200, metric.Name)
		if metric.For != "" {
			assert.Contains(t, brief, `"for": "the `+metric.For.Noun()+`'s id"`)
		}
	}
}

func TestLineForRecordsAloneNamesTheirMeasure(t *testing.T) {
	t.Parallel()

	line := boardhint.Line(&boardhint.Show{Records: &boardhint.Records{
		Words: "these facilities", Measure: "dwell",
	}})

	assert.Contains(t, line, `show_answer {"title": "…", "answer": "… …", "measure": "dwell", `+
		`"groups": [{"title": "…", "records": ["…"]}]}`)
	assert.Contains(t, line, "{total:dwell}")
	assert.Contains(t, line, "never typed")
}

func TestLineAddsTheRecordsTheCatalogSaysAMetricIsReadWith(t *testing.T) {
	t.Parallel()

	line := boardhint.Line(&boardhint.Show{Metric: billingboard.MetricAgingCompare})

	assert.Contains(t, line, `"measure": "amountDue", "groups": [`)
	assert.Contains(t, line, `{"title": "…", "records": ["…"]}]}`)
	assert.Contains(t, line, "read with receivable_list_open_items")
	assert.NotContains(t, line, "{count} does not apply")

	own := boardhint.Line(&boardhint.Show{
		Metric:  billingboard.MetricAgingCompare,
		Records: &boardhint.Records{Words: "these invoices", Measure: "total", From: "invoice_list"},
	})
	assert.Contains(t, own, "The second group holds these invoices, read with invoice_list")
	assert.Contains(t, own, `"measure": "total"`)
}

func TestLineLeadsWithTheFiguresTheMetricDeclaresAndSaysTheRestExist(t *testing.T) {
	t.Parallel()

	line := boardhint.Line(&boardhint.Show{Metric: settlementboard.MetricSettlementBatch})

	assert.Contains(t, line, `"answer": "… {batchNet:settlementBatch} …"`)
	assert.Contains(t, line, "The figures an answer leads with, written into it instead of "+
		"typed numbers: {batchNet:settlementBatch} (net pay), "+
		"{blockedSettlements:settlementBatch} (settlements that cannot be released as they "+
		"stand), {criticalExceptionSettlements:settlementBatch}")
	assert.Contains(t, line, "It also keeps batchGross, batchReimbursements")
	assert.NotContains(t, line, "{batchGross:settlementBatch}")
}

func TestLineForAListSaysAListToGoThroughIsItsTable(t *testing.T) {
	t.Parallel()

	for _, line := range []string{
		boardhint.Line(&boardhint.Show{Metric: billingboard.MetricUnappliedReceipts}),
		boardhint.Line(&boardhint.Show{Records: &boardhint.Records{
			Words: "these facilities", Measure: "dwell",
		}}),
	} {
		assert.Contains(t, line, boardhint.ListAsked)
		assert.NotContains(t, line[len(line)-len(boardhint.ListAsked):], "go through it",
			"the exception is never the hint's last word")
	}
	assert.NotContains(t,
		boardhint.Line(&boardhint.Show{Metric: operationsboard.MetricOnTimeByWeek}),
		boardhint.ListAsked)
}

func TestSubjectLineIsTheHintOfTheMetricThatReadsTheRecord(t *testing.T) {
	t.Parallel()

	line, metric := boardhint.SubjectLine(pulid.ID("ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"))
	assert.Equal(t, "iftaReturnChecks", metric)
	assert.Contains(t, line, `"metric": "iftaReturnChecks", "for": "ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"`)
	assert.Contains(t, line, "fuel_get_ifta_return", "the lines the board lists beside it")

	line, metric = boardhint.SubjectLine(pulid.ID("edip_01M7V48HPFRT5N51S07BK4QZ5C"))
	assert.Equal(t, "partnerDay", metric)
	assert.Contains(t, line, `"for": "edip_01M7V48HPFRT5N51S07BK4QZ5C"`)

	line, metric = boardhint.SubjectLine(pulid.ID("shp_01M7V48EQGE9TBM60WF4JKM3VQ"))
	assert.Empty(t, line, "no metric reads one shipment")
	assert.Empty(t, metric)
}
