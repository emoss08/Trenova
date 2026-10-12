package triageruleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewRow(id string, mutate func(*triageports.MessageRow)) *triageports.MessageRow {
	row := &triageports.MessageRow{
		ID:                pulid.ID(id),
		OrganizationID:    testOrg,
		BusinessUnitID:    testBU,
		FromAddress:       "luis@drivers.trenova.example",
		SenderDomain:      "drivers.trenova.example",
		Subject:           "POD for S-104251",
		Classification:    inboundmessage.ClassificationProofOfDelivery,
		Status:            inboundmessage.StatusActioned,
		ReceivedAt:        1_760_000_000,
		MatchedShipmentID: "shp_1",
		ReviewedBy:        "usr_dana",
		PersonFiled:       true,
		Reference:         "S-104251",
	}
	if mutate != nil {
		mutate(row)
	}

	return row
}

func TestMismatchOf(t *testing.T) {
	t.Parallel()

	file := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderDomain,
		"drivers.trenova.example")
	ignore := ruleFor("trule_ignore", inboundmessage.NovaActionIgnore, triage.SenderDomain,
		"drivers.trenova.example")
	assign := ruleFor("trule_assign", inboundmessage.NovaActionAssign, triage.SenderAnyone, "")
	assign.AssigneeID = "usr_dana"
	link := ruleFor("trule_link", inboundmessage.NovaActionLinkBySender, triage.SenderAnyone, "")

	tests := map[string]struct {
		rule *triage.Rule
		row  *triageports.MessageRow
		want inboundmessageports.RuleMismatch
	}{
		"a corrected kind": {
			rule: file,
			row:  previewRow("1", func(r *triageports.MessageRow) { r.ClassificationCorrected = true }),
			want: inboundmessageports.MismatchKindCorrected,
		},
		"filing without a shipment": {
			rule: file,
			row:  previewRow("2", func(r *triageports.MessageRow) { r.MatchedShipmentID = pulid.Nil }),
			want: inboundmessageports.MismatchNoMatchedRecord,
		},
		"linking without a party": {
			rule: link,
			row:  previewRow("3", nil),
			want: inboundmessageports.MismatchNoMatchedRecord,
		},
		"a person ignored what the rule would file": {
			rule: file,
			row:  previewRow("4", func(r *triageports.MessageRow) { r.Status = inboundmessage.StatusIgnored }),
			want: inboundmessageports.MismatchPersonDisagreed,
		},
		"a person handled it without filing": {
			rule: file,
			row:  previewRow("5", func(r *triageports.MessageRow) { r.PersonFiled = false }),
			want: inboundmessageports.MismatchPersonDisagreed,
		},
		"a person worked what the rule would ignore": {
			rule: ignore,
			row:  previewRow("6", nil),
			want: inboundmessageports.MismatchPersonDisagreed,
		},
		"a person gave it to someone else": {
			rule: assign,
			row:  previewRow("7", func(r *triageports.MessageRow) { r.OwnerID = "usr_marcus" }),
			want: inboundmessageports.MismatchPersonDisagreed,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, found := MismatchOf(tt.rule, tt.row)
			require.True(t, found)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMismatchOf_AgreementIsNotAMismatch(t *testing.T) {
	t.Parallel()

	file := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderDomain,
		"drivers.trenova.example")
	_, found := MismatchOf(file, previewRow("1", nil))
	assert.False(t, found)

	ignore := ruleFor("trule_ignore", inboundmessage.NovaActionIgnore, triage.SenderAnyone, "")
	_, found = MismatchOf(ignore, previewRow("2", func(r *triageports.MessageRow) {
		r.Status = inboundmessage.StatusIgnored
	}))
	assert.False(t, found)

	_, found = MismatchOf(ignore, previewRow("3", func(r *triageports.MessageRow) {
		r.Status = inboundmessage.StatusInReview
		r.ReviewedBy = pulid.Nil
	}))
	assert.False(t, found, "a message nobody has settled has no human outcome to disagree with")
}

func TestBuildPreview_ConfirmsEachCandidateAndPutsOffOnesFirst(t *testing.T) {
	t.Parallel()

	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderDomain,
		"drivers.trenova.example")
	candidates := &triageports.RuleCandidates{
		Total: 4,
		Rows: []*triageports.MessageRow{
			previewRow("imsg_new_ok", func(r *triageports.MessageRow) { r.ReceivedAt = 300 }),
			previewRow("imsg_off", func(r *triageports.MessageRow) {
				r.ReceivedAt = 200
				r.MatchedShipmentID = pulid.Nil
			}),
			previewRow("imsg_lookalike", func(r *triageports.MessageRow) {
				r.FromAddress = "x@notdrivers.trenova.example.com"
				r.ReceivedAt = 250
			}),
			previewRow("imsg_old_ok", func(r *triageports.MessageRow) {
				r.ReceivedAt = 100
				r.Reference = ""
				r.Subject = "Delivered PRO 77120"
			}),
		},
	}

	preview := BuildPreview(rule, candidates, 50)

	assert.Equal(t, 3, preview.Matched, "the candidate Covers refuses is not counted")
	assert.Equal(t, 1, preview.Off)
	assert.Equal(t, int64(50), preview.Since)
	require.Len(t, preview.Matches, 3)
	assert.Equal(t, pulid.ID("imsg_off"), preview.Matches[0].MessageID)
	require.NotNil(t, preview.Matches[0].Mismatch)
	assert.Equal(t, pulid.ID("imsg_new_ok"), preview.Matches[1].MessageID)
	assert.Equal(t, pulid.ID("imsg_old_ok"), preview.Matches[2].MessageID)
	assert.Equal(t, "77120", preview.Matches[2].Reference)
}

func TestBuildPreview_BoundsTheMatches(t *testing.T) {
	t.Parallel()

	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	rows := make([]*triageports.MessageRow, 0, maxPreviewMatches+10)
	for i := range maxPreviewMatches + 10 {
		rows = append(rows, previewRow(pulid.MustNew("imsg_").String(), func(r *triageports.MessageRow) {
			r.ReceivedAt = int64(i)
		}))
	}

	preview := BuildPreview(rule, &triageports.RuleCandidates{Total: 900, Rows: rows}, 0)

	assert.Equal(t, 900, preview.Matched)
	assert.Len(t, preview.Matches, maxPreviewMatches)
}

func TestSentenceHash_IsStableAndSensitive(t *testing.T) {
	t.Parallel()

	a := ruleFor("trule_a", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	b := ruleFor("trule_b", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	assert.Equal(t, SentenceHash(a), SentenceHash(b), "the id is not part of the sentence")

	b.Action = inboundmessage.NovaActionIgnore
	assert.NotEqual(t, SentenceHash(a), SentenceHash(b))
}
