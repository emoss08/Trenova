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
