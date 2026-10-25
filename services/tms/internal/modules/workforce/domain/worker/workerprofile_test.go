package worker

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validWorkerProfile() *WorkerProfile {
	return &WorkerProfile{
		ID:               pulid.MustNew("wrkp_"),
		WorkerID:         pulid.MustNew("wrk_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		OrganizationID:   pulid.MustNew("org_"),
		DOB:              946684800,
		LicenseNumber:    "DL123456",
		CDLClass:         CDLClassA,
		Endorsement:      EndorsementTypeNone,
		LicenseExpiry:    1893456000,
		HireDate:         1609459200,
		ComplianceStatus: ComplianceStatusCompliant,
	}
}

func validWorkerPTO() *WorkerPTO {
	return &WorkerPTO{
		ID:             pulid.MustNew("wrkpto_"),
		WorkerID:       pulid.MustNew("wrk_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         PTOStatusRequested,
		Type:           PTOTypeVacation,
		StartDate:      1700000000,
		EndDate:        1700100000,
		Reason:         "Family vacation",
	}
}

func TestWorkerProfile_GetTableName(t *testing.T) {
	t.Parallel()

	wp := &WorkerProfile{}
	assert.Equal(t, "worker_profiles", wp.GetTableName())
}

func TestWorkerProfile_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(wp *WorkerProfile)
		wantErr bool
	}{
		{
			name:    "valid profile passes",
			modify:  func(_ *WorkerProfile) {},
			wantErr: false,
		},
		{
			name: "missing DOB fails",
			modify: func(wp *WorkerProfile) {
				wp.DOB = 0
			},
			wantErr: true,
		},
		{
			name: "missing license number fails",
			modify: func(wp *WorkerProfile) {
				wp.LicenseNumber = ""
			},
			wantErr: true,
		},
		{
			name: "license number too long fails",
			modify: func(wp *WorkerProfile) {
				wp.LicenseNumber = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCDEFGHIJKLMNOP1"
			},
			wantErr: true,
		},
		{
			name: "invalid endorsement fails",
			modify: func(wp *WorkerProfile) {
				wp.Endorsement = EndorsementType("Z")
			},
			wantErr: true,
		},
		{
			name: "missing license expiry fails",
			modify: func(wp *WorkerProfile) {
				wp.LicenseExpiry = 0
			},
			wantErr: true,
		},
		{
			name: "missing hire date fails",
			modify: func(wp *WorkerProfile) {
				wp.HireDate = 0
			},
			wantErr: true,
		},
		{
			name: "invalid compliance status fails",
			modify: func(wp *WorkerProfile) {
				wp.ComplianceStatus = ComplianceStatus("Unknown")
			},
			wantErr: true,
		},
		{
			name: "hazmat endorsement without expiry fails",
			modify: func(wp *WorkerProfile) {
				wp.Endorsement = EndorsementTypeHazmat
				wp.HazmatExpiry = nil
			},
			wantErr: true,
		},
		{
			name: "hazmat endorsement with zero expiry fails",
			modify: func(wp *WorkerProfile) {
				wp.Endorsement = EndorsementTypeHazmat
				zeroVal := int64(0)
				wp.HazmatExpiry = &zeroVal
			},
			wantErr: true,
		},
		{
			name: "tanker hazmat endorsement without expiry fails",
			modify: func(wp *WorkerProfile) {
				wp.Endorsement = EndorsementTypeTankerHazmat
				wp.HazmatExpiry = nil
			},
			wantErr: true,
		},
		{
			name: "hazmat endorsement with valid expiry passes",
			modify: func(wp *WorkerProfile) {
				wp.Endorsement = EndorsementTypeHazmat
				expiry := int64(1893456000)
				wp.HazmatExpiry = &expiry
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wp := validWorkerProfile()
			tt.modify(wp)

			multiErr := errortypes.NewMultiError()
			wp.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestWorkerPTO_GetTableName(t *testing.T) {
	t.Parallel()

	wpto := &WorkerPTO{}
	assert.Equal(t, "worker_pto", wpto.GetTableName())
}

func TestWorkerPTO_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(wpto *WorkerPTO)
		wantErr bool
	}{
		{
			name:    "valid PTO passes",
			modify:  func(_ *WorkerPTO) {},
			wantErr: false,
		},
		{
			name: "missing worker ID fails",
			modify: func(wpto *WorkerPTO) {
				wpto.WorkerID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "invalid status fails",
			modify: func(wpto *WorkerPTO) {
				wpto.Status = PTOStatus("Pending")
			},
			wantErr: true,
		},
		{
			name: "invalid type fails",
			modify: func(wpto *WorkerPTO) {
				wpto.Type = PTOType("Jury")
			},
			wantErr: true,
		},
		{
			name: "missing start date fails",
			modify: func(wpto *WorkerPTO) {
				wpto.StartDate = 0
			},
			wantErr: true,
		},
		{
			name: "missing end date fails",
			modify: func(wpto *WorkerPTO) {
				wpto.EndDate = 0
			},
			wantErr: true,
		},
		{
			name: "end date before start date fails",
			modify: func(wpto *WorkerPTO) {
				wpto.StartDate = 1700100000
				wpto.EndDate = 1700000000
			},
			wantErr: true,
		},
		{
			name: "a single day off starts and ends on the same date",
			modify: func(wpto *WorkerPTO) {
				wpto.StartDate = 1700000000
				wpto.EndDate = 1700000000
			},
			wantErr: false,
		},
		{
			name: "missing reason fails",
			modify: func(wpto *WorkerPTO) {
				wpto.Reason = ""
			},
			wantErr: true,
		},
		{
			name: "reason too long fails",
			modify: func(wpto *WorkerPTO) {
				long := make([]byte, 256)
				for i := range long {
					long[i] = 'a'
				}
				wpto.Reason = string(long)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wpto := validWorkerPTO()
			tt.modify(wpto)

			multiErr := errortypes.NewMultiError()
			wpto.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}
