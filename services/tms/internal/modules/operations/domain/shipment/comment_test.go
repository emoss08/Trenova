package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestShipmentComment_ValidateRequiresUserForUserAuthoredSources(t *testing.T) {
	t.Parallel()

	entity := validShipmentComment()
	entity.UserID = pulid.Nil
	entity.Source = shipmentvocab.CommentSourceUser

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestShipmentComment_ValidateAllowsSystemSourceWithoutUser(t *testing.T) {
	t.Parallel()

	entity := validShipmentComment()
	entity.UserID = pulid.Nil
	entity.Source = shipmentvocab.CommentSourceSystem

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func validShipmentComment() *ShipmentComment {
	return &ShipmentComment{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ShipmentID:     pulid.MustNew("shp_"),
		UserID:         pulid.MustNew("usr_"),
		Comment:        "hello",
		Source:         shipmentvocab.CommentSourceUser,
	}
}

func TestShipmentComment_WrittenOutside(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		comment ShipmentComment
		outside bool
	}{
		"a dispatcher's note": {
			comment: ShipmentComment{
				Source: shipmentvocab.CommentSourceUser,
				Type:   shipmentvocab.CommentTypeDispatch,
			},
		},
		"a dispatcher's note to the driver": {
			comment: ShipmentComment{
				Source:     shipmentvocab.CommentSourceUser,
				Type:       shipmentvocab.CommentTypeDispatch,
				Visibility: shipmentvocab.CommentVisibilityDriver,
			},
		},
		"a driver's Dash post": {
			comment: ShipmentComment{
				Source:     shipmentvocab.CommentSourceUser,
				Type:       shipmentvocab.CommentTypeDriverUpdate,
				Visibility: shipmentvocab.CommentVisibilityDriver,
				Metadata:   map[string]any{CommentMetadataOrigin: CommentOriginDash},
			},
			outside: true,
		},
		"a Dash post written before the origin was stamped": {
			comment: ShipmentComment{
				Source: shipmentvocab.CommentSourceUser,
				Type:   shipmentvocab.CommentTypeDriverUpdate,
			},
			outside: true,
		},
		"an EDI status message": {
			comment: ShipmentComment{
				Source:   shipmentvocab.CommentSourceSystem,
				Type:     shipmentvocab.CommentTypeStatusUpdate,
				Metadata: map[string]any{CommentMetadataOrigin: CommentOriginEDI},
			},
			outside: true,
		},
		"an integration's note": {
			comment: ShipmentComment{
				Source: shipmentvocab.CommentSourceIntegration,
				Type:   shipmentvocab.CommentTypeInternal,
			},
			outside: true,
		},
		"a system note from inside Trenova": {
			comment: ShipmentComment{
				Source: shipmentvocab.CommentSourceSystem,
				Type:   shipmentvocab.CommentTypeBilling,
			},
		},
		"an agent's note from a clean run": {
			comment: ShipmentComment{
				Source: shipmentvocab.CommentSourceAI,
				Type:   shipmentvocab.CommentTypeDriverUpdate,
			},
		},
		"an agent's note written after outside content": {
			comment: ShipmentComment{
				Source:   shipmentvocab.CommentSourceAI,
				Type:     shipmentvocab.CommentTypeInternal,
				Metadata: map[string]any{CommentMetadataTainted: true},
			},
			outside: true,
		},
		"the record of an email an agent sent": {
			comment: ShipmentComment{
				Source:   shipmentvocab.CommentSourceSystem,
				Type:     shipmentvocab.CommentTypeCustomerUpdate,
				Metadata: map[string]any{CommentMetadataOrigin: CommentOriginAgent},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.outside, tc.comment.WrittenOutside())
		})
	}
	assert.False(t, (*ShipmentComment)(nil).WrittenOutside())
}
