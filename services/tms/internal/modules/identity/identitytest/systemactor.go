package identitytest

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
)

func ExpectSystemActor(repo *tenantportsmock.MockUserRepository) pulid.ID {
	systemActor := pulid.MustNew("usr_")
	repo.EXPECT().
		SystemActorID(mock.Anything, mock.Anything).
		Return(systemActor, nil).
		Maybe()

	return systemActor
}
