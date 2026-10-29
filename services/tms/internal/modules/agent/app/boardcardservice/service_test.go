package boardcardservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordsummaryports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var now = time.Unix(1_793_000_000, 0)

type fakePermissions struct{ allowed map[permission.Resource]bool }

func (f *fakePermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: f.allowed[permission.Resource(req.Resource)],
	}, nil
}

type fakeSummaries struct {
	asked recordsummaryports.RecordSummaries
	refs  []map[permission.Resource][]pulid.ID
}

func (f *fakeSummaries) Summaries(
	_ context.Context,
	req *recordsummaryports.RecordSummaryRequest,
) (recordsummaryports.RecordSummaries, error) {
	f.refs = append(f.refs, req.Refs)
	out := recordsummaryports.RecordSummaries{}
	for resource, ids := range req.Refs {
		for _, id := range ids {
			if summary, ok := f.asked[resource][id]; ok {
				if out[resource] == nil {
					out[resource] = map[pulid.ID]boardassemble.Summary{}
				}
				out[resource][id] = summary
			}
		}
	}

	return out, nil
}

type fakeArtifacts struct {
	byID      map[pulid.ID]*assistantartifact.Artifact
	questions map[pulid.ID]string
}

func (f *fakeArtifacts) GetByID(
	_ context.Context,
	req assistantartifactports.GetArtifactRequest,
) (*assistantartifact.Artifact, error) {
	artifact, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	return artifact, nil
}

func (f *fakeArtifacts) TurnQuestions(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
	[]pulid.ID,
) (map[pulid.ID]string, error) {
	return f.questions, nil
}

type fakeThreads struct{ refuse bool }

func (f *fakeThreads) GetThread(
	context.Context,
	*conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	if f.refuse {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}

	return &conversation.Thread{}, nil
}

type fakeAnalytics struct {
	opts []*kernelports.AnalyticsRequestOptions
}

func (f *fakeAnalytics) GetAnalytics(
	_ context.Context,
	opts *kernelports.AnalyticsRequestOptions,
) (kernelports.AnalyticsData, error) {
	f.opts = append(f.opts, opts)

	return kernelports.AnalyticsData{
		"activeShipments": map[string]any{"count": 41},
		"onTimePercent":   map[string]any{"percent": 97.4},
		"profitability":   map[string]any{"avgCpm": 2.31, "avgMarginPct": nil},
	}, nil
}

type fakeReceivables struct{ calls int }

func (f *fakeReceivables) GetDashboardKPIs(
	context.Context,
	customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDashboardKPIs, error) {
	f.calls++

	return &customerledgerports.ARDashboardKPIs{
		Overview:       &customerledgerports.ARBalanceOverview{OverdueMinor: 824000},
		CurrentDSODays: 41.26,
	}, nil
}

type fakeActivity struct{ entries []*audit.Entry }

func (f *fakeActivity) ListByResourceID(
	context.Context,
	*auditports.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{Items: f.entries}, nil
}

type fakeOrganizations struct{}

func (fakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type fixture struct {
	svc         *Service
	permissions *fakePermissions
	summaries   *fakeSummaries
	artifacts   *fakeArtifacts
	threads     *fakeThreads
	analytics   *fakeAnalytics
	receivables *fakeReceivables
	activity    *fakeActivity
	actor       *permissionports.RequestActor
	thread      pulid.ID
}

func newFixture() *fixture {
	f := &fixture{
		permissions: &fakePermissions{allowed: map[permission.Resource]bool{
			permission.ResourceShipment: true,
		}},
		summaries:   &fakeSummaries{asked: recordsummaryports.RecordSummaries{}},
		artifacts:   &fakeArtifacts{byID: map[pulid.ID]*assistantartifact.Artifact{}},
		threads:     &fakeThreads{},
		analytics:   &fakeAnalytics{},
		receivables: &fakeReceivables{},
		activity:    &fakeActivity{},
		actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		thread: pulid.MustNew("thr_"),
	}
	f.svc = &Service{
		l:             zap.NewNop(),
		permissions:   f.permissions,
		summaries:     f.summaries,
		artifacts:     f.artifacts,
		threads:       f.threads,
		analytics:     f.analytics,
		receivables:   f.receivables,
		activity:      f.activity,
		organizations: fakeOrganizations{},
		now:           func() time.Time { return now },
	}

	return f
}

func (f *fixture) request(shown *board.Board) *assistantports.BoardFillRequest {
	return &assistantports.BoardFillRequest{Actor: f.actor, ThreadID: f.thread, Board: shown}
}

func parsed(t *testing.T, args map[string]any) *board.Board {
	t.Helper()
	shown, problems := boardparse.Parse(args)
	require.Empty(t, problems)

	return shown
}

func (f *fixture) analysis(threadID pulid.ID, display *analysis.Display) pulid.ID {
	payload := map[string]any{}
	_ = jsonutils.Convert(display, &payload)
	id := pulid.MustNew("art_")
	f.artifacts.byID[id] = &assistantartifact.Artifact{
		ID: id, ThreadID: threadID, Kind: assistantartifact.KindAnalysis,
		Title: "Carrier scorecard", Payload: payload,
	}

	return id
}

func TestFillReadsOnlyTheRecordsTheReaderMayRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment, invoice := pulid.MustNew("shp_"), pulid.MustNew("inv_")
	f.summaries.asked[permission.ResourceShipment] = map[pulid.ID]boardassemble.Summary{
		shipment: {Label: "S2610-0412", Status: "InTransit"},
	}
	shown := parsed(t, map[string]any{
		"title": "Late and unpaid", "verdict": "Both [late][1].",
		"blocks": []any{
			map[string]any{"type": "records", "records": []any{shipment.String()}},
			map[string]any{"type": "records", "records": []any{invoice.String()}},
		},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	require.Len(t, f.summaries.refs, 1)
	assert.Equal(t, map[permission.Resource][]pulid.ID{
		permission.ResourceShipment: {shipment},
	}, f.summaries.refs[0])
	assert.Equal(t, board.StateReady, filled.Blocks[0].State)
	assert.Equal(t, "S2610-0412", filled.Blocks[0].Rows[0].Label)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[1].State)
	assert.Equal(t, now.Unix(), filled.ReadAt)
}

func TestFillReadsMetricsFromTheFiguresHomeShows(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.allowed[permission.ResourceAccountsReceivable] = true
	shown := parsed(t, map[string]any{
		"title": "The day", "verdict": "Steady [today][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{"label": "Active", "metric": "activeShipments"},
			map[string]any{"label": "On time", "metric": "onTimePercent"},
			map[string]any{"label": "Margin", "metric": "marginPercent"},
			map[string]any{"label": "Overdue", "metric": "arOverdue"},
		}}},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	require.Len(t, f.analytics.opts, 1)
	assert.Equal(t, kernelports.AnalyticsIncludeMetrics, f.analytics.opts[0].Include)
	assert.Equal(t, "America/Chicago", f.analytics.opts[0].Timezone)
	assert.Equal(t, 1, f.receivables.calls)
	figures := filled.Blocks[0].Figures
	assert.Equal(t, "41", board.FormatValue(figures[0].Value))
	assert.Equal(t, "97.4%", board.FormatValue(figures[1].Value))
	assert.Nil(t, figures[2].Value, "a margin the reader may not see is not shown")
	assert.Equal(t, "$8,240.00", board.FormatValue(figures[3].Value))
}

func TestFillLeavesOutAMetricTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shown := parsed(t, map[string]any{
		"title": "Receivables", "verdict": "Open [money][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{"label": "Overdue", "metric": "arOverdue"},
		}}},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	assert.Equal(t, 0, f.receivables.calls)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[0].State)
}

func TestFillReadsARecordsHistoryOldestFirst(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.allowed[permission.ResourceAuditLog] = true
	shipment := pulid.MustNew("shp_")
	f.summaries.asked[permission.ResourceShipment] = map[pulid.ID]boardassemble.Summary{
		shipment: {Label: "S2610-0412"},
	}
	f.activity.entries = []*audit.Entry{
		{
			Timestamp:     now.Unix() - 60,
			Operation:     permission.OpUpdate,
			Comment:       "Out of hours",
			PrincipalType: "api_key",
		},
		{
			Timestamp: now.Unix() - 3600,
			Operation: permission.OpCreate,
			User:      &tenant.User{Name: "Ana Ruiz"},
		},
	}
	shown := parsed(t, map[string]any{
		"title": "One load", "verdict": "Stuck [here][1].",
		"blocks": []any{map[string]any{"type": "timeline", "record": shipment.String()}},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	events := filled.Blocks[0].Events
	require.Len(t, events, 2)
	assert.Equal(t, "create", events[0].Operation)
	assert.Equal(t, "Ana Ruiz", events[0].Actor)
	assert.Equal(t, "Out of hours", events[1].Comment)
	assert.Empty(t, events[1].Actor)
	assert.Equal(t, "api_key", events[1].Principal)
}

func TestFillDrawsTheTimelineUnreadableWithoutTheAuditLog(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment := pulid.MustNew("shp_")
	f.summaries.asked[permission.ResourceShipment] = map[pulid.ID]boardassemble.Summary{
		shipment: {Label: "S2610-0412"},
	}
	shown := parsed(t, map[string]any{
		"title": "One load", "verdict": "Stuck [here][1].",
		"blocks": []any{map[string]any{"type": "timeline", "record": shipment.String()}},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[0].State)
}

func TestCheckRefusesWhatTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shown := parsed(t, map[string]any{
		"title": "Unpaid", "verdict": "Open [money][1].",
		"blocks": []any{map[string]any{
			"type": "records", "records": []any{pulid.MustNew("inv_").String()},
		}},
	})

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Contains(t, mistake.Reason, "may not read invoices")
}

func TestCheckRefusesAnAnalysisFromAnotherConversation(t *testing.T) {
	t.Parallel()

	f := newFixture()
	elsewhere := f.analysis(pulid.MustNew("thr_"), &analysis.Display{Charts: []analysis.Chart{{}}})
	shown := parsed(t, map[string]any{
		"title": "Lanes", "verdict": "Thin [revenue][1].",
		"blocks": []any{map[string]any{
			"type": "chart", "analysis": elsewhere.String(), "chart": 1.0,
		}},
	})

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Contains(t, mistake.Reason, "not one an analysis_run result kept in this conversation")
}

func TestCheckRefusesAChartTheAnalysisNeverDrew(t *testing.T) {
	t.Parallel()

	f := newFixture()
	id := f.analysis(f.thread, &analysis.Display{Charts: []analysis.Chart{{}}})
	shown := parsed(t, map[string]any{
		"title": "Lanes", "verdict": "Thin [revenue][1].",
		"blocks": []any{map[string]any{"type": "chart", "analysis": id.String(), "chart": 2.0}},
	})

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Equal(t, "block 1 asks for chart 2 of an analysis that drew 1", mistake.Reason)
}

func TestCheckRefusesAComparisonByAColumnTheTableLacks(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.allowed[permission.ResourceCarrier] = true
	id := f.analysis(f.thread, &analysis.Display{Tables: []analysis.Table{{
		Columns: []analysis.TableColumn{{Name: "carrier_id"}, {Name: "on_time"}},
	}}})
	shown := parsed(t, map[string]any{
		"title": "Carriers", "verdict": "Blue Line [wins][1].",
		"blocks": []any{map[string]any{
			"type":     "compare",
			"records":  []any{pulid.MustNew("car_").String(), pulid.MustNew("car_").String()},
			"analysis": id.String(), "table": 1.0, "key": "carrier_id",
			"rows": []any{map[string]any{"column": "claims"}},
		}},
	})

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Equal(t,
		`block 1: the table has no "claims" column; its columns are carrier_id, on_time`,
		mistake.Reason)
}

func TestCheckRefusesATimelineWithoutTheAuditLog(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shown := parsed(t, map[string]any{
		"title": "One load", "verdict": "Stuck [here][1].",
		"blocks": []any{
			map[string]any{"type": "timeline", "record": pulid.MustNew("shp_").String()},
		},
	})

	err := f.svc.Check(t.Context(), f.request(shown))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Contains(t, mistake.Reason, "leave the timeline out")
}

func (f *fixture) board(t *testing.T, shipment pulid.ID, snapshot board.Snapshot) pulid.ID {
	t.Helper()
	shown := parsed(t, map[string]any{
		"title": "Late loads", "verdict": "One is [late][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{
				"label": "Late now", "records": []any{shipment.String()}, "measure": "lateness",
			},
		}}},
	})
	payload, err := (&board.Payload{Board: *shown, Snapshot: snapshot, Author: "Dispatch Desk"}).Map()
	require.NoError(t, err)
	id := pulid.MustNew("art_")
	f.artifacts.byID[id] = &assistantartifact.Artifact{
		ID: id, ThreadID: f.thread, Kind: assistantartifact.KindBoard, Title: shown.Title,
		Payload: payload, CreatedAt: now.Unix() - 600, MessageID: pulid.MustNew("msg_"),
	}

	return id
}

func TestCardComparesTheBoardWithWhatItShowedWhenAnswered(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment := pulid.MustNew("shp_")
	f.summaries.asked[permission.ResourceShipment] = map[pulid.ID]boardassemble.Summary{
		shipment: {Label: "S2610-0412", Measures: map[board.Measure]board.RowMeasure{
			board.MeasureLateness: {
				Measure: board.MeasureLateness,
				Value:   board.Value{Kind: board.ValueDuration, Number: decimal.Zero},
			},
		}},
	}
	f.artifacts.questions = map[pulid.ID]string{}
	id := f.board(t, shipment, board.Snapshot{Values: map[string]board.Value{
		board.FigureKey(0, 0): {Kind: board.ValueCount, Number: decimal.NewFromInt(1)},
	}})
	for _, artifact := range f.artifacts.byID {
		f.artifacts.questions[artifact.MessageID] = "Why are these late?"
	}

	card, err := f.svc.Card(t.Context(), &assistantports.BoardCardRequest{
		Thread:     conversationports.GetThreadRequest{ID: f.thread},
		ArtifactID: id,
		Actor:      f.actor,
	})

	require.NoError(t, err)
	assert.Equal(t, "Dispatch Desk", card.Author)
	assert.Equal(t, "Why are these late?", card.Question)
	figure := card.Filled.Blocks[0].Figures[0]
	assert.Equal(t, "0", board.FormatValue(figure.Value))
	assert.Equal(t, "1", board.FormatValue(figure.Was))
	assert.Contains(t, card.Markdown, "> Why are these late?")
	assert.Contains(t, card.Markdown, "- Late now: 0 (1 when answered)")
}

func TestCardRefusesWhatIsNotABoardOfThisConversation(t *testing.T) {
	t.Parallel()

	f := newFixture()
	id := f.board(t, pulid.MustNew("shp_"), board.Snapshot{})

	_, err := f.svc.Card(t.Context(), &assistantports.BoardCardRequest{
		Thread:     conversationports.GetThreadRequest{ID: pulid.MustNew("thr_")},
		ArtifactID: id,
		Actor:      f.actor,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	f.threads.refuse = true
	_, err = f.svc.Card(t.Context(), &assistantports.BoardCardRequest{
		Thread:     conversationports.GetThreadRequest{ID: f.thread},
		ArtifactID: id,
		Actor:      f.actor,
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, errArtifactGone))
}
