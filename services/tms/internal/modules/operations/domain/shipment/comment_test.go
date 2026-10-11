package shipment

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestShipmentComment_ValidateRequiresUserForUserAuthoredSources(t *testing.T) {
	t.Parallel()

	entity := validShipmentComment()
	entity.UserID = pulid.Nil
	entity.Source = CommentSourceUser

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestShipmentComment_ValidateAllowsSystemSourceWithoutUser(t *testing.T) {
	t.Parallel()

	entity := validShipmentComment()
	entity.UserID = pulid.Nil
	entity.Source = CommentSourceSystem

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
		Source:         CommentSourceUser,
	}
}

func TestShipmentComment_WrittenOutside(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		comment ShipmentComment
		outside bool
	}{
		"a dispatcher's note": {
			comment: ShipmentComment{Source: CommentSourceUser, Type: CommentTypeDispatch},
		},
		"a dispatcher's note to the driver": {
			comment: ShipmentComment{
				Source:     CommentSourceUser,
				Type:       CommentTypeDispatch,
				Visibility: CommentVisibilityDriver,
			},
		},
		"a driver's Dash post": {
			comment: ShipmentComment{
				Source:     CommentSourceUser,
				Type:       CommentTypeDriverUpdate,
				Visibility: CommentVisibilityDriver,
				Metadata:   map[string]any{CommentMetadataOrigin: CommentOriginDash},
			},
			outside: true,
		},
		"a Dash post written before the origin was stamped": {
			comment: ShipmentComment{Source: CommentSourceUser, Type: CommentTypeDriverUpdate},
			outside: true,
		},
		"an EDI status message": {
			comment: ShipmentComment{
				Source:   CommentSourceSystem,
				Type:     CommentTypeStatusUpdate,
				Metadata: map[string]any{CommentMetadataOrigin: CommentOriginEDI},
			},
			outside: true,
		},
		"an integration's note": {
			comment: ShipmentComment{Source: CommentSourceIntegration, Type: CommentTypeInternal},
			outside: true,
		},
		"a system note from inside Trenova": {
			comment: ShipmentComment{Source: CommentSourceSystem, Type: CommentTypeBilling},
		},
		"an agent's note from a clean run": {
			comment: ShipmentComment{Source: CommentSourceAI, Type: CommentTypeDriverUpdate},
		},
		"an agent's note written after outside content": {
			comment: ShipmentComment{
				Source:   CommentSourceAI,
				Type:     CommentTypeInternal,
				Metadata: map[string]any{CommentMetadataTainted: true},
			},
			outside: true,
		},
		"the record of an email an agent sent": {
			comment: ShipmentComment{
				Source:   CommentSourceSystem,
				Type:     CommentTypeCustomerUpdate,
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
