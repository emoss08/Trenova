package customerupdateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type stubComments struct {
	services.ShipmentCommentService
	items []*shipment.ShipmentComment
}

func (s *stubComments) ListByShipmentID(
	context.Context,
	*repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: s.items}, nil
}

func agentEmailComment(at int64) *shipment.ShipmentComment {
	return &shipment.ShipmentComment{
		CreatedAt: at,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "email_customer"},
	}
}

func TestAlreadyTold(t *testing.T) {
	t.Parallel()

	const now = int64(1_767_225_600)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("shp_")

	cases := []struct {
		name  string
		items []*shipment.ShipmentComment
		want  bool
	}{
		{
			name:  "nothing said yet",
			items: nil,
			want:  false,
		},
		{
			name:  "an email ten minutes ago",
			items: []*shipment.ShipmentComment{agentEmailComment(now - 600)},
			want:  true,
		},
		{
			name: "a delay notice sent from the board",
			items: []*shipment.ShipmentComment{{
				CreatedAt: now - 120,
				Type:      shipment.CommentTypeCustomerUpdate,
				Metadata:  map[string]any{"source": "board", "tool": SourceDelayNotice},
			}},
			want: true,
		},
		{
			name:  "an email just outside the window",
			items: []*shipment.ShipmentComment{agentEmailComment(now - 3601)},
			want:  false,
		},
		{
			name: "a dispatcher typing an update is not an email",
			items: []*shipment.ShipmentComment{{
				CreatedAt: now - 60,
				Type:      shipment.CommentTypeCustomerUpdate,
				Metadata:  map[string]any{"source": "user"},
			}},
			want: false,
		},
		{
			name: "another agent tool is not this one",
			items: []*shipment.ShipmentComment{{
				CreatedAt: now - 60,
				Type:      shipment.CommentTypeCustomerUpdate,
				Metadata:  map[string]any{"source": "agent", "tool": "add_shipment_comment"},
			}},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			told, err := AlreadyTold(
				t.Context(), &stubComments{items: tc.items}, tenant, shipmentID, now,
			)
			require.NoError(t, err)
			require.Equal(t, tc.want, told)
		})
	}
}

// With no comment service wired the guard cannot read anything back, and a
// guard that cannot read must not silently refuse every send.
func TestAlreadyTold_SaysNothingWhenItCannotRead(t *testing.T) {
	t.Parallel()

	told, err := AlreadyTold(
		t.Context(), nil,
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		pulid.MustNew("shp_"), 1_767_225_600,
	)
	require.NoError(t, err)
	require.False(t, told)
}
