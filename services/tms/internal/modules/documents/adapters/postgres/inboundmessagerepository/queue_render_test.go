package inboundmessagerepository

import (
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var setTarget = regexp.MustCompile(`(?:SET |, )"?(\w+)"? = `)

func assignedColumns(t *testing.T, sqlText string) map[string]int {
	t.Helper()

	start := strings.Index(sqlText, " SET ")
	end := strings.Index(sqlText, " WHERE ")
	require.True(t, start >= 0 && end > start, sqlText)

	counts := make(map[string]int)
	for _, match := range setTarget.FindAllStringSubmatch(sqlText[start:end], -1) {
		counts[match[1]]++
	}

	return counts
}

func TestEveryQueuePatchAssignsEachColumnOnce(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	cases := map[string]inboundmessageports.QueuePatch{
		"snooze until a time": {Park: &inboundmessageports.ParkPatch{
			Parking:      inboundmessage.ParkingSnoozed,
			SnoozedUntil: 1_800_000_000,
			By:           owner,
		}},
		"snooze until an event": {Park: &inboundmessageports.ParkPatch{
			Parking:       inboundmessage.ParkingSnoozed,
			SnoozeEvent:   inboundmessage.SnoozeEventShipmentDelivered,
			WatchShipment: true,
			By:            owner,
		}},
		"wait for a reply": {Park: &inboundmessageports.ParkPatch{
			Parking:       inboundmessage.ParkingAwaitingReply,
			AwaitingUntil: 1_800_000_000,
		}},
		"park while also asked to unpark": {
			Unpark: true,
			Park: &inboundmessageports.ParkPatch{
				Parking:      inboundmessage.ParkingSnoozed,
				SnoozedUntil: 1_800_000_000,
			},
		},
		"unpark and settle": {
			Unpark: true,
			Status: inboundmessage.StatusActioned,
			Review: &inboundmessageports.ReviewPatch{ReviewerID: owner},
			Owner:  &owner,
		},
	}

	for name, patch := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sqlText := updateQueueStateQuery(renderDB(), inboundmessageports.UpdateQueueStateRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
				IDs:   []pulid.ID{pulid.MustNew("imsg_")},
				Patch: patch,
				Now:   1_790_000_000,
			}).String()

			for column, count := range assignedColumns(t, sqlText) {
				assert.Equal(t, 1, count, "%s is assigned %d times in %s", column, count, sqlText)
			}
		})
	}
}
