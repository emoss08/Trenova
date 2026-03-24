package worker

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validWorker() *Worker {
	return &Worker{
		ID:             pulid.MustNew("wrk_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("st_"),
		Status:         domaintypes.StatusActive,
		Type:           WorkerTypeEmployee,
		DriverType:     DriverTypeOTR,
		FirstName:      "John",
		LastName:       "Doe",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		Gender:         GenderMale,
		Profile: &WorkerProfile{
			ID:               pulid.MustNew("wrkp_"),
			BusinessUnitID:   pulid.MustNew("bu_"),
			OrganizationID:   pulid.MustNew("org_"),
			DOB:              946684800,
			LicenseNumber:    "DL123456",
			CDLClass:         CDLClassA,
			Endorsement:      EndorsementTypeNone,
			LicenseExpiry:    1893456000,
			HireDate:         1609459200,
			ComplianceStatus: ComplianceStatusCompliant,
		},
	}
}

func TestWorker_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(w *Worker)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *Worker) {},
			wantErr: false,
		},
		{
			name: "missing first name fails",
			modify: func(w *Worker) {
				w.FirstName = ""
			},
			wantErr: true,
		},
		{
			name: "missing last name fails",
			modify: func(w *Worker) {
				w.LastName = ""
			},
			wantErr: true,
		},
		{
			name: "missing address line 1 fails",
			modify: func(w *Worker) {
				w.AddressLine1 = ""
			},
			wantErr: true,
		},
		{
			name: "missing city fails",
			modify: func(w *Worker) {
				w.City = ""
			},
			wantErr: true,
		},
		{
			name: "missing postal code fails",
			modify: func(w *Worker) {
				w.PostalCode = ""
			},
			wantErr: true,
		},
		{
			name: "invalid postal code fails",
			modify: func(w *Worker) {
				w.PostalCode = "ABCDE"
			},
			wantErr: true,
		},
		{
			name: "valid postal code with extension passes",
			modify: func(w *Worker) {
				w.PostalCode = "12345-6789"
			},
			wantErr: false,
		},
		{
			name: "missing state ID fails",
			modify: func(w *Worker) {
				w.StateID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "invalid gender fails",
			modify: func(w *Worker) {
				w.Gender = Gender("Unknown")
			},
			wantErr: true,
		},
		{
			name: "invalid worker type fails",
			modify: func(w *Worker) {
				w.Type = WorkerType("Invalid")
			},
			wantErr: true,
		},
		{
			name: "valid E.164 phone passes",
			modify: func(w *Worker) {
				w.PhoneNumber = "+12025551234"
			},
			wantErr: false,
		},
		{
			name: "empty phone passes",
			modify: func(w *Worker) {
				w.PhoneNumber = ""
			},
			wantErr: false,
		},
		{
			name: "non-E.164 phone fails",
			modify: func(w *Worker) {
				w.PhoneNumber = "555-0101"
			},
			wantErr: true,
		},
		{
			name: "valid E.164 emergency contact phone passes",
			modify: func(w *Worker) {
				w.EmergencyContactPhone = "+12025551234"
			},
			wantErr: false,
		},
		{
			name: "empty emergency contact phone passes",
			modify: func(w *Worker) {
				w.EmergencyContactPhone = ""
			},
			wantErr: false,
		},
		{
			name: "non-E.164 emergency contact phone fails",
			modify: func(w *Worker) {
				w.EmergencyContactPhone = "555-0101"
			},
			wantErr: true,
		},
		{
			name: "missing profile fails",
			modify: func(w *Worker) {
				w.Profile = nil
			},
			wantErr: true,
		},
		{
			name: "invalid status fails",
			modify: func(w *Worker) {
				w.Status = domaintypes.Status("Bad")
			},
			wantErr: true,
		},
		{
			name: "profile with missing license number fails",
			modify: func(w *Worker) {
				w.Profile.LicenseNumber = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := validWorker()
			tt.modify(w)

			multiErr := errortypes.NewMultiError()
			w.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestWorker_GetTableName(t *testing.T) {
	t.Parallel()

	w := &Worker{}
	assert.Equal(t, "workers", w.GetTableName())
}

func TestWorker_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		w := &Worker{}
		require.True(t, w.ID.IsNil())

		err := w.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, w.ID.IsNil())
		assert.NotZero(t, w.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("wrk_")
		w := &Worker{ID: existingID}

		err := w.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, w.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		w := &Worker{}

		err := w.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, w.UpdatedAt)
	})
}

func TestWorker_FullName(t *testing.T) {
	t.Parallel()

	w := &Worker{FirstName: "Jane", LastName: "Smith"}
	assert.Equal(t, "Jane Smith", w.FullName())
}
