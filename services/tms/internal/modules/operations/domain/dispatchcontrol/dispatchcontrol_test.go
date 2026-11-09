package dispatchcontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickupDelivery,
				ServiceFailureGracePeriod:  new(15),
			},
			wantGracePeriod: new(15),
			wantErr:         false,
		},
		{
			name: "invalid compliance enforcement level fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
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
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentType("Invalid"),
			},
			wantErr: true,
		},
		{
			name: "block enforcement recording pickup failures passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
				ComplianceEnforcementLevel: ComplianceEnforcementLevelBlock,
				RecordServiceFailures:      ServiceIncidentTypePickup,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "audit enforcement recording delivery failures passes",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
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
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypePickupDelivery,
				ServiceFailureGracePeriod:  new(10),
			},
			wantErr: false,
		},
		{
			name: "empty compliance enforcement level fails",
			entity: DispatchControl{
				ID:                         pulid.MustNew("dc_"),
				BusinessUnitID:             pulid.MustNew("bu_"),
				OrganizationID:             pulid.MustNew("org_"),
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

func TestNewDefaultDispatchControl(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	dc := NewDefaultDispatchControl(orgID, buID)

	assert.Equal(t, orgID, dc.OrganizationID)
	assert.Equal(t, buID, dc.BusinessUnitID)
	assert.True(t, dc.EnableAutoAssignment)
	assert.Equal(t, dispatcheconomics.Default(), dc.Economics)
	assert.False(t, dc.LearnDispatchStyle, "learning a dispatch style is a choice, off by default")
	assert.False(t, dc.PlanForEndMarket, "planning for end markets is a choice, off by default")
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

func TestResolvedHorizonSearchRounds(t *testing.T) {
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
			want:    int(DefaultHorizonSearchRounds),
		},
		{
			name:    "unset uses the default",
			control: &DispatchControl{},
			want:    int(DefaultHorizonSearchRounds),
		},
		{
			name:    "an explicit zero disables the search",
			control: &DispatchControl{HorizonSearchRounds: rounds(0)},
			want:    0,
		},
		{
			name:    "a negative value is malformed data and falls back to the default",
			control: &DispatchControl{HorizonSearchRounds: rounds(-10)},
			want:    int(DefaultHorizonSearchRounds),
		},
		{
			name:    "a configured value is honoured",
			control: &DispatchControl{HorizonSearchRounds: rounds(120)},
			want:    120,
		},
		{
			name:    "an oversized value is clamped",
			control: &DispatchControl{HorizonSearchRounds: rounds(5000)},
			want:    int(MaxHorizonSearchRounds),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.control.ResolvedHorizonSearchRounds())
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

func TestValidate_HorizonSearchRoundsBounds(t *testing.T) {
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
		{name: "the maximum is allowed", value: rounds(MaxHorizonSearchRounds)},
		{name: "negative is rejected", value: rounds(-1), wantErr: true},
		{name: "above the maximum is rejected", value: rounds(501), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dc := &DispatchControl{
				ComplianceEnforcementLevel: ComplianceEnforcementLevelWarning,
				RecordServiceFailures:      ServiceIncidentTypeNever,
				HorizonSearchRounds:        tt.value,
			}

			multiErr := errortypes.NewMultiError()
			dc.Validate(multiErr)

			found := false
			for _, e := range multiErr.Errors {
				if e.Field == "horizonSearchRounds" {
					found = true
				}
			}
			assert.Equal(t, tt.wantErr, found)
		})
	}
}

func TestExceptionSettings_DefaultWhenUnsetAndRejectOutOfRange(t *testing.T) {
	t.Parallel()

	dc := NewDefaultDispatchControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	dc.ExceptionDueInfoMinutes = 0
	dc.ExceptionEscalationMinutes = 0
	multiErr := errortypes.NewMultiError()
	dc.Validate(multiErr)
	require.False(t, multiErr.HasErrors())
	assert.Equal(t, DefaultExceptionDueInfoMinutes, dc.ExceptionDueInfoMinutes)
	assert.Equal(t, DefaultExceptionEscalationMinutes, dc.ExceptionEscalationMinutes)

	dc.ExceptionDueCriticalMinutes = 2
	dc.ExceptionMaxEscalations = 11
	multiErr = errortypes.NewMultiError()
	dc.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestExceptionPolicy_FallsBackForAMissingRecord(t *testing.T) {
	t.Parallel()

	var dc *DispatchControl
	policy := dc.ExceptionPolicy()
	assert.Equal(t, DefaultExceptionDueCriticalMinutes, policy.DueCriticalMinutes)
	assert.Equal(t, DefaultExceptionMaxEscalations, policy.MaxEscalations)
	assert.True(t, policy.DefaultAssigneeID.IsNil())
}

func TestEconomics_ReachesValidationWithItsOwnRules(t *testing.T) {
	t.Parallel()

	dc := NewDefaultDispatchControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	dc.ID = pulid.MustNew("dc_")
	dc.Economics.EmptyMileCents = -1

	multiErr := errortypes.NewMultiError()
	dc.Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "a negative cost is refused")
}

func TestResolvedEconomics_FillsAnUnsetOrganizationWithDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, dispatcheconomics.Default(), (&DispatchControl{}).ResolvedEconomics())
	var nothing *DispatchControl
	assert.Equal(t, dispatcheconomics.Default(), nothing.ResolvedEconomics())
}

func TestAfterLoad_ReadsAStoredEmptyEconomicsAsTheDefaults(t *testing.T) {
	t.Parallel()

	dc := &DispatchControl{}
	require.NoError(t, dc.AfterLoad(t.Context()))

	assert.Equal(t, dispatcheconomics.Default(), dc.Economics,
		"a row stored before economics existed reads as the defaults, never as zeros")
}
