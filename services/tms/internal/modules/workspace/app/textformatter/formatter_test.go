package textformatter

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/dataentrycontrolportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type record struct {
	orgID   pulid.ID
	buID    pulid.ID
	applied []dataentrycontrol.TextFormat
}

func (r *record) GetOrganizationID() pulid.ID { return r.orgID }

func (r *record) GetBusinessUnitID() pulid.ID { return r.buID }

func (r *record) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	r.applied = append(r.applied, format)
}

func TestFormatAppliesTheTenantsSettings(t *testing.T) {
	t.Parallel()

	repo := dataentrycontrolportsmock.NewMockDataEntryControlRepository(t)
	entity := &record{orgID: pulid.MustNew("org_"), buID: pulid.MustNew("bu_")}
	repo.EXPECT().
		GetOrCreate(mock.Anything, entity.orgID, entity.buID).
		Return(&dataentrycontrol.DataEntryControl{
			CodeCase:  dataentrycontrol.CaseFormatLower,
			NameCase:  dataentrycontrol.CaseFormatUpper,
			EmailCase: dataentrycontrol.CaseFormatAsEntered,
			CityCase:  dataentrycontrol.CaseFormatTitleCase,
		}, nil).
		Once()

	err := New(Params{Repo: repo}).Format(t.Context(), entity)

	require.NoError(t, err)
	assert.Equal(t, []dataentrycontrol.TextFormat{{
		CodeCase:  dataentrycontrol.CaseFormatLower,
		NameCase:  dataentrycontrol.CaseFormatUpper,
		EmailCase: dataentrycontrol.CaseFormatAsEntered,
		CityCase:  dataentrycontrol.CaseFormatTitleCase,
	}}, entity.applied)
}

func TestFormatReturnsTheReadError(t *testing.T) {
	t.Parallel()

	repo := dataentrycontrolportsmock.NewMockDataEntryControlRepository(t)
	entity := &record{orgID: pulid.MustNew("org_"), buID: pulid.MustNew("bu_")}
	readErr := errors.New("db down")
	repo.EXPECT().
		GetOrCreate(mock.Anything, entity.orgID, entity.buID).
		Return(nil, readErr).
		Once()

	err := New(Params{Repo: repo}).Format(t.Context(), entity)

	require.ErrorIs(t, err, readErr)
	assert.Empty(t, entity.applied)
}
