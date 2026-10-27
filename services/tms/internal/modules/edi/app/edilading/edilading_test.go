package edilading

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func failedFields(t *testing.T, err error) []string {
	t.Helper()
	var multi *errortypes.MultiError
	require.ErrorAs(t, err, &multi)
	fields := make([]string, 0, len(multi.Errors))
	for _, item := range multi.Errors {
		fields = append(fields, item.Field)
	}
	return fields
}

func TestValidateProject_RefusesPathsThatLeaveTheProject(t *testing.T) {
	t.Parallel()

	partner := lading.File{Path: "partner.lading", Source: "partner acme \"Acme\"\n"}
	tests := []struct {
		name    string
		files   []lading.File
		samples []lading.Sample
		field   string
	}{
		{name: "no files", field: "files"},
		{
			name:  "an absolute path",
			files: []lading.File{partner, {Path: "/etc/maps.lading", Source: "map x"}},
			field: "files[1].path",
		},
		{
			name:  "a parent directory",
			files: []lading.File{partner, {Path: "maps/../../x.lading", Source: "map x"}},
			field: "files[1].path",
		},
		{
			name:  "a source file that is not Lading",
			files: []lading.File{partner, {Path: "maps/tender.txt", Source: "map x"}},
			field: "files[1].path",
		},
		{
			name:  "the same path twice",
			files: []lading.File{partner, partner},
			field: "files[1].path",
		},
		{
			name:    "a sample at a file's path",
			files:   []lading.File{partner},
			samples: []lading.Sample{{Path: "partner.lading", Content: "ISA"}},
			field:   "samples[0].path",
		},
		{
			name:  "a file over its size",
			files: []lading.File{{Path: "partner.lading", Source: strings.Repeat("x", MaxFileBytes+1)}},
			field: "files[0].source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateProject(tt.files, tt.samples)
			assert.Contains(t, failedFields(t, err), tt.field)
		})
	}

	require.NoError(t, validateProject(
		[]lading.File{partner, {Path: "maps/tender.lading", Source: "map tenderIn"}},
		[]lading.Sample{{Path: "samples/tender.x12", Content: "ISA"}},
	))
}

func TestMergeSamples_ReplacesThoseAtTheSamePathAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	merged := MergeSamples(
		[]lading.Sample{
			{Path: "samples/a.x12", Content: "old a"},
			{Path: "samples/b.x12", Content: "b"},
		},
		[]lading.Sample{
			{Path: "samples/a.x12", Content: "new a"},
			{Path: "samples/c.x12", Content: "c"},
		},
	)

	assert.Equal(t, []lading.Sample{
		{Path: "samples/b.x12", Content: "b"},
		{Path: "samples/a.x12", Content: "new a"},
		{Path: "samples/c.x12", Content: "c"},
	}, merged)
}

func TestValidateRunInput_TakesExactlyOneBoundedInput(t *testing.T) {
	t.Parallel()

	x12 := "ISA*00~"
	csv := "a,b\n1,2\n"
	huge := strings.Repeat("x", MaxSampleBytes+1)

	require.NoError(t, validateRunInput(&lading.RunInput{X12: &x12}))
	require.NoError(t, validateRunInput(&lading.RunInput{CSV: &csv}))
	require.NoError(t, validateRunInput(&lading.RunInput{JSON: jsonutils.RawJSON(`{"a":1}`)}))
	require.Error(t, validateRunInput(&lading.RunInput{}), "no input")
	require.Error(t, validateRunInput(&lading.RunInput{X12: &x12, CSV: &csv}), "two inputs")
	require.Error(t, validateRunInput(&lading.RunInput{X12: &huge}), "an input over its size")
}

func TestNotReady_NamesEveryReasonAVersionCannotBeCertified(t *testing.T) {
	t.Parallel()

	err := notReady(&edi.LadingReadiness{Errors: 2, Failing: 1, UncoveredFlowClauses: 3})

	var multi *errortypes.MultiError
	require.ErrorAs(t, err, &multi)
	require.Len(t, multi.Errors, 4, "errors, failing tests, no tests and uncovered flow clauses")
	assert.Equal(t, "files", multi.Errors[0].Field)
	assert.Equal(t, errortypes.ErrRequired, multi.Errors[2].Code, "a project with no tests")
}

func TestInboundMaps_AreTheX12MapsTrenovaReads(t *testing.T) {
	t.Parallel()

	manifest := &lading.Check{Maps: []lading.Map{
		{Name: "tenderIn", Direction: lading.DirectionInbound, Source: "x12"},
		{Name: "statusOut", Direction: lading.DirectionOutbound, Source: "document"},
		{Name: "ordersCsv", Direction: lading.DirectionInbound, Source: "csv"},
		{Name: "answerIn", Direction: lading.DirectionInbound, Source: "x12"},
	}}

	assert.Equal(t, []string{"answerIn", "tenderIn"}, inboundMaps(manifest))
}

func TestRequireReplays_NeedsAFreshReplayOfEveryInboundMapThatRejectsNothingNew(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	version := &edi.LadingVersion{
		ID:        pulid.MustNew("ldv_"),
		UpdatedAt: 1_000,
		Manifest: lading.Check{Maps: []lading.Map{
			{Name: "tenderIn", Direction: lading.DirectionInbound, Source: "x12"},
			{Name: "statusIn", Direction: lading.DirectionInbound, Source: "x12"},
			{Name: "answerIn", Direction: lading.DirectionInbound, Source: "x12"},
		}},
	}
	replays := ediportsmock.NewMockLadingReplayRepository(t)
	replays.EXPECT().
		ListReplays(mock.Anything, ediports.ListLadingReplaysRequest{
			TenantInfo: tenantInfo,
			VersionID:  version.ID,
		}).
		Return([]*edi.LadingReplay{
			{
				Map: "tenderIn", Status: edi.LadingReplayStatusCompleted, CreatedAt: 1_500,
				Summary: &lading.ReplaySummary{Total: 40, Same: 40},
			},
			{
				Map: "statusIn", Status: edi.LadingReplayStatusCompleted, CreatedAt: 1_200,
				Summary: &lading.ReplaySummary{Total: 10, NewlyRejected: 2},
			},
			{
				Map: "answerIn", Status: edi.LadingReplayStatusCompleted, CreatedAt: 900,
				Summary: &lading.ReplaySummary{Total: 5, Same: 5},
			},
		}, nil).
		Once()
	projects := &Projects{l: zap.NewNop(), replays: replays}

	err := projects.requireReplays(t.Context(), tenantInfo, version)

	var multi *errortypes.MultiError
	require.ErrorAs(t, err, &multi)
	require.Len(t, multi.Errors, 2)
	assert.Equal(t, errortypes.ErrRequired, multi.Errors[0].Code,
		"answerIn was replayed before the draft last changed")
	assert.Equal(t, []any{"answerIn"}, multi.Errors[0].Args)
	assert.Equal(t, []any{"statusIn", 2}, multi.Errors[1].Args)
}

func TestStartReplay_RecordsNothingWithoutTheWorkflowService(t *testing.T) {
	t.Parallel()

	projects := &Projects{
		l:        zap.NewNop(),
		projects: ediportsmock.NewMockLadingProjectRepository(t),
		replays:  ediportsmock.NewMockLadingReplayRepository(t),
	}

	replay, err := projects.StartReplay(t.Context(), &StartReplayRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		VersionID:  pulid.MustNew("ldv_"),
		Map:        "tenderIn",
		From:       1_000,
		To:         2_000,
	}, nil)

	assert.Nil(t, replay)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, "Replays need the workflow service, which is not configured", business.Message)
}

func TestStartReplay_RefusesAPeriodThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()

	projects := &Projects{l: zap.NewNop()}

	_, err := projects.StartReplay(t.Context(), &StartReplayRequest{From: 2_000, To: 1_000}, nil)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "to", validation.Field)
	assert.Equal(t, errortypes.ErrInvalid, validation.Code)
}

func TestValidateSettings_KeepsOnlyDeclaredSettingsOfTheirType(t *testing.T) {
	t.Parallel()

	manifest := &lading.Check{Partner: &lading.Partner{Settings: []lading.Setting{
		{Name: "autoAccept", Type: "Bool"},
		{Name: "maxStops", Type: "Int"},
		{Name: "maxWeight", Type: "Weight"},
		{Name: "minimum", Type: "Money"},
	}}}

	settings, err := ValidateSettings(manifest, map[string]any{
		"autoAccept": true,
		"maxStops":   float64(6),
		"maxWeight":  map[string]any{"value": "44000", "unit": "lb"},
		"minimum":    map[string]any{"amount": "250.00", "currency": "USD"},
	})
	require.NoError(t, err)
	assert.Len(t, settings, 4)

	_, err = ValidateSettings(manifest, map[string]any{
		"autoAccept": "yes",
		"maxStops":   float64(2.5),
		"maxWeight":  float64(44000),
		"unknown":    true,
	})
	assert.ElementsMatch(t, []string{
		"settings.autoAccept",
		"settings.maxStops",
		"settings.maxWeight",
		"settings.unknown",
	}, failedFields(t, err))

	cleared, err := ValidateSettings(manifest, map[string]any{"autoAccept": nil})
	require.NoError(t, err)
	assert.Empty(t, cleared, "a null value returns the setting to its default")
}

func TestReplayInput_ReadsWhatEachDirectionRan(t *testing.T) {
	t.Parallel()

	inbound := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), RawX12: "ISA*00~", GeneratedAt: 1_767_787_200}
	input, ok := replayInput(inbound, edi.DocumentDirectionInbound)
	require.True(t, ok)
	require.NotNil(t, input.Input.X12)
	assert.Equal(t, "ISA*00~", *input.Input.X12)
	assert.Equal(t, "2026-01-07T12:00:00Z", input.Now)

	outbound := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), Document: jsonutils.RawJSON(`{"key":"L-1"}`)}
	input, ok = replayInput(outbound, edi.DocumentDirectionOutbound)
	require.True(t, ok)
	assert.JSONEq(t, `{"key":"L-1"}`, string(input.Input.Document))

	_, ok = replayInput(&edi.EDIMessage{}, edi.DocumentDirectionOutbound)
	assert.False(t, ok, "an outbound message from before Lading has no document to replay")
}

func TestProgramAcknowledgment_IsWhatTheReceiveClauseNames(t *testing.T) {
	t.Parallel()

	program := &Program{Manifest: &lading.Check{Flows: []lading.Flow{{
		Name: "tender",
		Clauses: []lading.Clause{
			{Kind: lading.ClauseReceive, Set: "204", Map: "tenderIn"},
			{Kind: lading.ClauseReceive, Set: "214", Map: "statusIn", Acknowledge: "999"},
			{Kind: lading.ClauseReceive, Set: "210", Map: "invoiceIn", Acknowledge: lading.AcknowledgeNone},
		},
	}}}}

	assert.Equal(t, edi.AcknowledgmentType997, program.Acknowledgment(edi.TransactionSet204))
	assert.Equal(t, edi.AcknowledgmentType999, program.Acknowledgment(edi.TransactionSet214))
	assert.Equal(t, edi.AcknowledgmentTypeNone, program.Acknowledgment(edi.TransactionSet210))
	assert.Equal(t, edi.AcknowledgmentTypeNone, program.Acknowledgment(edi.TransactionSet990),
		"a set the version does not receive is not acknowledged")
}

func TestShadowing_IsAShadowVersionWithinItsWindow(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()

	assert.True(t, shadowing(&edi.LadingVersion{Status: edi.LadingVersionStatusShadow, ShadowUntil: &future}))
	assert.False(t, shadowing(&edi.LadingVersion{Status: edi.LadingVersionStatusShadow, ShadowUntil: &past}))
	assert.False(t, shadowing(&edi.LadingVersion{Status: edi.LadingVersionStatusShadow}))
	assert.False(t, shadowing(&edi.LadingVersion{Status: edi.LadingVersionStatusCertified, ShadowUntil: &future}))
}

func TestSettings_EncodesEachValueForTheRun(t *testing.T) {
	t.Parallel()

	settings, err := Settings(&edi.LadingProject{Settings: map[string]any{
		"autoAccept": true,
		"maxWeight":  map[string]any{"value": "44000", "unit": "lb"},
	}})
	require.NoError(t, err)
	assert.JSONEq(t, `true`, string(settings["autoAccept"]))
	assert.JSONEq(t, `{"value":"44000","unit":"lb"}`, string(settings["maxWeight"]))

	empty, err := Settings(&edi.LadingProject{})
	require.NoError(t, err)
	assert.Empty(t, empty, "a project with no settings sends none")
}

func TestCustomFieldTypesAreTheOnesLadingReads(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Decimal", customFieldType(customfield.FieldTypeNumber))
	assert.Equal(t, "Date", customFieldType(customfield.FieldTypeDate))
	assert.Equal(t, "Bool", customFieldType(customfield.FieldTypeBoolean))
	assert.Equal(t, "Text", customFieldType(customfield.FieldTypeText))
	assert.Equal(t, "Text", customFieldType(customfield.FieldTypeSelect))
	assert.Equal(t, "Texts", customFieldType(customfield.FieldTypeMultiSelect))
}
