package dispatchcontrol

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestDispatchControl_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		entity          DispatchControl
		wantGracePeriod *int
		wantErr         bool
	}{
		{
			name: "valid entity passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeNever,
				ServiceFailureGracePeriod:  nil,
			},
			wantGracePeriod: nil,
			wantErr:         false,
		},
		{
			name: "pickup service failures default missing grace period",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickup,
				ServiceFailureGracePeriod:  nil,
			},
			wantGracePeriod: new(DefaultServiceFailureGracePeriod),
			wantErr:         false,
		},
		{
			name: "grace period must be greater than zero when recording service failures",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeDelivery,
				ServiceFailureGracePeriod:  new(0),
			},
			wantErr: true,
		},
		{
			name: "negative grace period rejected after normalization",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickupDelivery,
				ServiceFailureGracePeriod:  new(-1),
			},
			wantErr: true,
		},
		{
			name: "grace period valid when recording service failures",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyAvailability,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickupDelivery,
				ServiceFailureGracePeriod:  new(15),
			},
			wantGracePeriod: new(15),
			wantErr:         false,
		},
		{
			name: "invalid auto assignment strategy fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategy("Invalid"),
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeNever,
			},
			wantErr: true,
		},
		{
			name: "invalid compliance enforcement level fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevel("Invalid"),
				RecordServiceFailures:      ServiceIncidentTypeNever,
			},
			wantErr: true,
		},
		{
			name: "invalid record service failures fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentType("Invalid"),
			},
			wantErr: true,
		},
		{
			name: "availability strategy passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyAvailability,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelBlock,
				RecordServiceFailures:      ServiceIncidentTypePickup,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "load balancing strategy passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyLoadBalancing,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelAudit,
				RecordServiceFailures:      ServiceIncidentTypeDelivery,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "all valid service incident types pass",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeAllExceptShipper,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "pickup delivery service incident type passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickupDelivery,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "empty auto assignment strategy fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategy(""),
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeNever,
			},
			wantErr: true,
		},
		{
			name: "empty compliance enforcement level fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevel(""),
				RecordServiceFailures:      ServiceIncidentTypeNever,
			},
			wantErr: true,
		},
		{
			name: "empty record service failures fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentType(""),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.entity.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
			if tt.wantGracePeriod != nil {
				require.NotNil(t, tt.entity.ServiceFailureGracePeriod)
				assert.Equal(t, *tt.wantGracePeriod, *tt.entity.ServiceFailureGracePeriod)
			}
		})
	}
}

func TestDispatchControl_NormalizeServiceFailureSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		incidentType    ServiceIncidentType
		gracePeriod     *int
		wantGracePeriod *int
	}{
		{
			name:            "never preserves nil grace period",
			incidentType:    ServiceIncidentTypeNever,
			gracePeriod:     nil,
			wantGracePeriod: nil,
		},
		{
			name:            "pickup defaults nil grace period",
			incidentType:    ServiceIncidentTypePickup,
			gracePeriod:     nil,
			wantGracePeriod: new(DefaultServiceFailureGracePeriod),
		},
		{
			name:            "delivery defaults nil grace period",
			incidentType:    ServiceIncidentTypeDelivery,
			gracePeriod:     nil,
			wantGracePeriod: new(DefaultServiceFailureGracePeriod),
		},
		{
			name:            "pickup delivery defaults nil grace period",
			incidentType:    ServiceIncidentTypePickupDelivery,
			gracePeriod:     nil,
			wantGracePeriod: new(DefaultServiceFailureGracePeriod),
		},
		{
			name:            "all except shipper defaults nil grace period",
			incidentType:    ServiceIncidentTypeAllExceptShipper,
			gracePeriod:     nil,
			wantGracePeriod: new(DefaultServiceFailureGracePeriod),
		},
		{
			name:            "explicit valid grace period is preserved",
			incidentType:    ServiceIncidentTypeDelivery,
			gracePeriod:     new(45),
			wantGracePeriod: new(45),
		},
		{
			name:            "explicit zero grace period is preserved for validation",
			incidentType:    ServiceIncidentTypePickup,
			gracePeriod:     new(0),
			wantGracePeriod: new(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := &DispatchControl{
				RecordServiceFailures:     tt.incidentType,
				ServiceFailureGracePeriod: tt.gracePeriod,
			}

			entity.NormalizeServiceFailureSettings()

			if tt.wantGracePeriod == nil {
				assert.Nil(t, entity.ServiceFailureGracePeriod)
				return
			}
			require.NotNil(t, entity.ServiceFailureGracePeriod)
			assert.Equal(t, *tt.wantGracePeriod, *entity.ServiceFailureGracePeriod)
		})
	}
}

func TestDispatchControl_GetTableName(t *testing.T) {
	t.Parallel()

	dc := &DispatchControl{}
	assert.Equal(t, "dispatch_controls", dc.GetTableName())
}

func TestDispatchControl_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dc_")
	dc := &DispatchControl{ID: id}
	assert.Equal(t, id, dc.GetID())
}

func TestDispatchControl_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	dc := &DispatchControl{OrganizationID: orgID}
	assert.Equal(t, orgID, dc.GetOrganizationID())
}

func TestDispatchControl_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	dc := &DispatchControl{BusinessUnitID: buID}
	assert.Equal(t, buID, dc.GetBusinessUnitID())
}

func TestDispatchControl_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		dc := &DispatchControl{}
		require.True(t, dc.ID.IsNil())

		err := dc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, dc.ID.IsNil())
		assert.NotZero(t, dc.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("dc_")
		dc := &DispatchControl{ID: existingID}

		err := dc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, dc.ID)
		assert.NotZero(t, dc.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		dc := &DispatchControl{}

		err := dc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, dc.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		dc := &DispatchControl{}

		err := dc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, dc.CreatedAt)
		assert.NotZero(t, dc.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		dc := &DispatchControl{}

		err := dc.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, dc.ID.IsNil())
		assert.Zero(t, dc.CreatedAt)
		assert.Zero(t, dc.UpdatedAt)
	})
}

//go:fix inline
func intPtr(v int) *int {
	return new(v)
}

func TestNewDefaultDispatchControl(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	dc := NewDefaultDispatchControl(orgID, buID)

	assert.Equal(t, orgID, dc.OrganizationID)
	assert.Equal(t, buID, dc.BusinessUnitID)
	assert.True(t, dc.EnableAutoAssignment)
	assert.Equal(t, AutoAssignmentStrategyProximity, dc.AutoAssignmentStrategy)
	assert.True(t, dc.EnforceWorkerAssign)
	assert.True(t, dc.EnforceTrailerContinuity)
	assert.True(t, dc.EnforceHOSCompliance)
	assert.True(t, dc.EnforceWorkerPTARestrictions)
	assert.True(t, dc.EnforceWorkerTractorFleetContinuity)
	assert.True(t, dc.EnforceDriverQualificationCompliance)
	assert.True(t, dc.EnforceMedicalCertCompliance)
	assert.True(t, dc.EnforceHazmatCompliance)
	assert.True(t, dc.EnforceDrugAndAlcoholCompliance)
	assert.Equal(t, ComplianceEnforcementLevelWarning, dc.ComplianceEnforcementLevel)
	assert.Equal(t, ServiceIncidentTypeNever, dc.RecordServiceFailures)
}

func TestServiceIncidentType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    ServiceIncidentType
		want string
	}{
		{name: "never", s: ServiceIncidentTypeNever, want: "Never"},
		{name: "pickup", s: ServiceIncidentTypePickup, want: "Pickup"},
		{name: "delivery", s: ServiceIncidentTypeDelivery, want: "Delivery"},
		{name: "pickup delivery", s: ServiceIncidentTypePickupDelivery, want: "PickupDelivery"},
		{
			name: "all except shipper",
			s:    ServiceIncidentTypeAllExceptShipper,
			want: "AllExceptShipper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}

func TestServiceIncidentType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    ServiceIncidentType
		want bool
	}{
		{name: "never", s: ServiceIncidentTypeNever, want: true},
		{name: "pickup", s: ServiceIncidentTypePickup, want: true},
		{name: "delivery", s: ServiceIncidentTypeDelivery, want: true},
		{name: "pickup delivery", s: ServiceIncidentTypePickupDelivery, want: true},
		{name: "all except shipper", s: ServiceIncidentTypeAllExceptShipper, want: true},
		{name: "invalid", s: ServiceIncidentType("Invalid"), want: false},
		{name: "empty", s: ServiceIncidentType(""), want: false},
		{name: "lowercase", s: ServiceIncidentType("never"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.s.IsValid())
		})
	}
}

func TestServiceIncidentTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ServiceIncidentType
		wantErr bool
	}{
		{name: "never", input: "Never", want: ServiceIncidentTypeNever, wantErr: false},
		{name: "pickup", input: "Pickup", want: ServiceIncidentTypePickup, wantErr: false},
		{name: "delivery", input: "Delivery", want: ServiceIncidentTypeDelivery, wantErr: false},
		{
			name:    "pickup delivery",
			input:   "PickupDelivery",
			want:    ServiceIncidentTypePickupDelivery,
			wantErr: false,
		},
		{
			name:    "all except shipper",
			input:   "AllExceptShipper",
			want:    ServiceIncidentTypeAllExceptShipper,
			wantErr: false,
		},
		{name: "invalid", input: "Invalid", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ServiceIncidentTypeFromString(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidServiceIncidentType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAutoAssignmentStrategy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    AutoAssignmentStrategy
		want string
	}{
		{name: "proximity", a: AutoAssignmentStrategyProximity, want: "Proximity"},
		{name: "availability", a: AutoAssignmentStrategyAvailability, want: "Availability"},
		{name: "load balancing", a: AutoAssignmentStrategyLoadBalancing, want: "LoadBalancing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.a.String())
		})
	}
}

func TestAutoAssignmentStrategy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    AutoAssignmentStrategy
		want bool
	}{
		{name: "proximity", a: AutoAssignmentStrategyProximity, want: true},
		{name: "availability", a: AutoAssignmentStrategyAvailability, want: true},
		{name: "load balancing", a: AutoAssignmentStrategyLoadBalancing, want: true},
		{name: "invalid", a: AutoAssignmentStrategy("Invalid"), want: false},
		{name: "empty", a: AutoAssignmentStrategy(""), want: false},
		{name: "lowercase", a: AutoAssignmentStrategy("proximity"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.a.IsValid())
		})
	}
}

func TestAutoAssignmentStrategyFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    AutoAssignmentStrategy
		wantErr bool
	}{
		{
			name:    "proximity",
			input:   "Proximity",
			want:    AutoAssignmentStrategyProximity,
			wantErr: false,
		},
		{
			name:    "availability",
			input:   "Availability",
			want:    AutoAssignmentStrategyAvailability,
			wantErr: false,
		},
		{
			name:    "load balancing",
			input:   "LoadBalancing",
			want:    AutoAssignmentStrategyLoadBalancing,
			wantErr: false,
		},
		{name: "invalid", input: "Invalid", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AutoAssignmentStrategyFromString(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidAutoAssignmentStrategy)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestComplianceEnforcementLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    ComplianceEnforcementLevel
		want string
	}{
		{name: "warning", c: ComplianceEnforcementLevelWarning, want: "Warning"},
		{name: "block", c: ComplianceEnforcementLevelBlock, want: "Block"},
		{name: "audit", c: ComplianceEnforcementLevelAudit, want: "Audit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.c.String())
		})
	}
}

func TestComplianceEnforcementLevel_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    ComplianceEnforcementLevel
		want bool
	}{
		{name: "warning", c: ComplianceEnforcementLevelWarning, want: true},
		{name: "block", c: ComplianceEnforcementLevelBlock, want: true},
		{name: "audit", c: ComplianceEnforcementLevelAudit, want: true},
		{name: "invalid", c: ComplianceEnforcementLevel("Invalid"), want: false},
		{name: "empty", c: ComplianceEnforcementLevel(""), want: false},
		{name: "lowercase", c: ComplianceEnforcementLevel("warning"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.c.IsValid())
		})
	}
}

func TestComplianceEnforcementLevelFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ComplianceEnforcementLevel
		wantErr bool
	}{
		{
			name:    "warning",
			input:   "Warning",
			want:    ComplianceEnforcementLevelWarning,
			wantErr: false,
		},
		{name: "block", input: "Block", want: ComplianceEnforcementLevelBlock, wantErr: false},
		{name: "audit", input: "Audit", want: ComplianceEnforcementLevelAudit, wantErr: false},
		{name: "invalid", input: "Invalid", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ComplianceEnforcementLevelFromString(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidComplianceEnforcementLevel)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestComplianceEnforcementLevel_ShouldBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    ComplianceEnforcementLevel
		want bool
	}{
		{name: "block returns true", c: ComplianceEnforcementLevelBlock, want: true},
		{name: "warning returns false", c: ComplianceEnforcementLevelWarning, want: false},
		{name: "audit returns false", c: ComplianceEnforcementLevelAudit, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.c.ShouldBlock())
		})
	}
}

func TestComplianceEnforcementLevel_ShouldWarn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    ComplianceEnforcementLevel
		want bool
	}{
		{name: "warning returns true", c: ComplianceEnforcementLevelWarning, want: true},
		{name: "block returns false", c: ComplianceEnforcementLevelBlock, want: false},
		{name: "audit returns false", c: ComplianceEnforcementLevelAudit, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.c.ShouldWarn())
		})
	}
}

func TestComplianceEnforcementLevel_IsAuditOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    ComplianceEnforcementLevel
		want bool
	}{
		{name: "audit returns true", c: ComplianceEnforcementLevelAudit, want: true},
		{name: "warning returns false", c: ComplianceEnforcementLevelWarning, want: false},
		{name: "block returns false", c: ComplianceEnforcementLevelBlock, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.c.IsAuditOnly())
		})
	}
}

func TestPresetWeights_EveryStrategyCoversAllFactors(t *testing.T) {
	strategies := []AutoAssignmentStrategy{
		AutoAssignmentStrategyProximity,
		AutoAssignmentStrategyAvailability,
		AutoAssignmentStrategyLoadBalancing,
		AutoAssignmentStrategyPerformance,
	}

	for _, strategy := range strategies {
		weights := PresetWeights(strategy)
		assert.Len(t, weights, len(AllScoringFactors()),
			"strategy %s must weight every factor", strategy)
		for _, factor := range AllScoringFactors() {
			weight, ok := weights[factor]
			assert.True(t, ok, "strategy %s is missing factor %s", strategy, factor)
			assert.Greater(t, weight, 0.0)
			assert.LessOrEqual(t, weight, 10.0)
		}
	}
}

func TestScoringFactor_NewFactorsAreValid(t *testing.T) {
	for _, factor := range []ScoringFactor{
		FactorOnTimeHistory,
		FactorSafetyHistory,
		FactorAcceptance,
		FactorPTOProximity,
	} {
		assert.True(t, factor.IsValid(), "%s must be a valid scoring factor", factor)
	}
	assert.False(t, ScoringFactor("bogus").IsValid())
}

func TestAutoAssignmentStrategy_Performance(t *testing.T) {
	assert.True(t, AutoAssignmentStrategyPerformance.IsValid())

	parsed, err := AutoAssignmentStrategyFromString("Performance")
	require.NoError(t, err)
	assert.Equal(t, AutoAssignmentStrategyPerformance, parsed)
}

func TestResolve_OverridesNewFactors(t *testing.T) {
	weights := ScoringWeights{
		"onTimeHistory": 7.5,
		"acceptance":    0,
	}

	resolved := weights.Resolve(AutoAssignmentStrategyPerformance)
	assert.InDelta(t, 7.5, resolved[FactorOnTimeHistory], 0.001)
	assert.InDelta(t, 0.0, resolved[FactorAcceptance], 0.001)
	assert.InDelta(t, 2.0, resolved[FactorSafetyHistory], 0.001,
		"untouched factors keep the preset value")
}

func TestHorizonSearchRounds(t *testing.T) {
	t.Parallel()

	rounds := func(v int16) *int16 { return &v }

	tests := []struct {
		name    string
		control *DispatchControl
		want    int
	}{
		{
			name:    "nil control uses the default",
			control: nil,
			want:    int(DefaultHorizonSearchIterations),
		},
		{
			name:    "unset uses the default",
			control: &DispatchControl{},
			want:    int(DefaultHorizonSearchIterations),
		},
		{
			name:    "an explicit zero disables the search",
			control: &DispatchControl{HorizonSearchIterations: rounds(0)},
			want:    0,
		},
		{
			name:    "a negative value is malformed data and falls back to the default",
			control: &DispatchControl{HorizonSearchIterations: rounds(-10)},
			want:    int(DefaultHorizonSearchIterations),
		},
		{
			name:    "a configured value is honoured",
			control: &DispatchControl{HorizonSearchIterations: rounds(120)},
			want:    120,
		},
		{
			name:    "an oversized value is clamped",
			control: &DispatchControl{HorizonSearchIterations: rounds(5000)},
			want:    int(MaxHorizonSearchIterations),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.control.HorizonSearchRounds())
		})
	}
}

func TestHorizonMovesPerDriver(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		int(DefaultHorizonMaxMovesPerDriver),
		(*DispatchControl)(nil).HorizonMovesPerDriver(),
	)
	assert.Equal(
		t,
		int(DefaultHorizonMaxMovesPerDriver),
		(&DispatchControl{}).HorizonMovesPerDriver(),
	)
	assert.Equal(
		t,
		5,
		(&DispatchControl{HorizonMaxMovesPerDriver: 5}).HorizonMovesPerDriver(),
	)
	assert.Equal(
		t,
		int(MaxHorizonMovesPerDriver),
		(&DispatchControl{HorizonMaxMovesPerDriver: 999}).HorizonMovesPerDriver(),
	)
}

func TestValidate_HorizonSearchIterationsBounds(t *testing.T) {
	t.Parallel()

	rounds := func(v int16) *int16 { return &v }

	tests := []struct {
		name    string
		value   *int16
		wantErr bool
	}{
		{name: "unset is allowed", value: nil},
		{name: "zero disables the search", value: rounds(0)},
		{name: "a normal value is allowed", value: rounds(50)},
		{name: "the maximum is allowed", value: rounds(MaxHorizonSearchIterations)},
		{name: "negative is rejected", value: rounds(-1), wantErr: true},
		{name: "above the maximum is rejected", value: rounds(501), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dc := &DispatchControl{
				AutoAssignmentStrategy:     AutoAssignmentStrategyProximity,
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeNever,
				HorizonSearchIterations:    tt.value,
			}

			multiErr := errortypes.NewMultiError()
			dc.Validate(multiErr)

			found := false
			for _, e := range multiErr.Errors {
				if e.Field == "horizonSearchIterations" {
					found = true
				}
			}
			assert.Equal(t, tt.wantErr, found)
		})
	}
}
