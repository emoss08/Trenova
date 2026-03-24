package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wt       WorkerType
		expected string
	}{
		{"employee", WorkerTypeEmployee, "Employee"},
		{"contractor", WorkerTypeContractor, "Contractor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.wt.String())
		})
	}
}

func TestWorkerType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wt       WorkerType
		expected bool
	}{
		{"employee is valid", WorkerTypeEmployee, true},
		{"contractor is valid", WorkerTypeContractor, true},
		{"empty is invalid", WorkerType(""), false},
		{"unknown is invalid", WorkerType("Freelancer"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.wt.IsValid())
		})
	}
}

func TestWorkerTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  WorkerType
		expectErr bool
	}{
		{"employee", "Employee", WorkerTypeEmployee, false},
		{"contractor", "Contractor", WorkerTypeContractor, false},
		{"invalid", "Invalid", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := WorkerTypeFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidWorkerType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEndorsementType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		et       EndorsementType
		expected string
	}{
		{"none", EndorsementTypeNone, "O"},
		{"tanker", EndorsementTypeTanker, "N"},
		{"hazmat", EndorsementTypeHazmat, "H"},
		{"tanker hazmat", EndorsementTypeTankerHazmat, "X"},
		{"passenger", EndorsementTypePassenger, "P"},
		{"double triple", EndorsementTypeDoubleTriple, "T"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.et.String())
		})
	}
}

func TestEndorsementType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		et       EndorsementType
		expected bool
	}{
		{"none is valid", EndorsementTypeNone, true},
		{"tanker is valid", EndorsementTypeTanker, true},
		{"hazmat is valid", EndorsementTypeHazmat, true},
		{"tanker hazmat is valid", EndorsementTypeTankerHazmat, true},
		{"passenger is valid", EndorsementTypePassenger, true},
		{"double triple is valid", EndorsementTypeDoubleTriple, true},
		{"empty is invalid", EndorsementType(""), false},
		{"unknown is invalid", EndorsementType("Z"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.et.IsValid())
		})
	}
}

func TestEndorsementType_RequiresHazmatExpiry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		et       EndorsementType
		expected bool
	}{
		{"hazmat requires expiry", EndorsementTypeHazmat, true},
		{"tanker hazmat requires expiry", EndorsementTypeTankerHazmat, true},
		{"none does not require expiry", EndorsementTypeNone, false},
		{"tanker does not require expiry", EndorsementTypeTanker, false},
		{"passenger does not require expiry", EndorsementTypePassenger, false},
		{"double triple does not require expiry", EndorsementTypeDoubleTriple, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.et.RequiresHazmatExpiry())
		})
	}
}

func TestEndorsementTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  EndorsementType
		expectErr bool
	}{
		{"none", "O", EndorsementTypeNone, false},
		{"tanker", "N", EndorsementTypeTanker, false},
		{"hazmat", "H", EndorsementTypeHazmat, false},
		{"tanker hazmat", "X", EndorsementTypeTankerHazmat, false},
		{"passenger", "P", EndorsementTypePassenger, false},
		{"double triple", "T", EndorsementTypeDoubleTriple, false},
		{"invalid", "Z", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := EndorsementTypeFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidEndorsementType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestComplianceStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cs       ComplianceStatus
		expected string
	}{
		{"compliant", ComplianceStatusCompliant, "Compliant"},
		{"non compliant", ComplianceStatusNonCompliant, "NonCompliant"},
		{"pending", ComplianceStatusPending, "Pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.cs.String())
		})
	}
}

func TestComplianceStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cs       ComplianceStatus
		expected bool
	}{
		{"compliant is valid", ComplianceStatusCompliant, true},
		{"non compliant is valid", ComplianceStatusNonCompliant, true},
		{"pending is valid", ComplianceStatusPending, true},
		{"empty is invalid", ComplianceStatus(""), false},
		{"unknown is invalid", ComplianceStatus("Unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.cs.IsValid())
		})
	}
}

func TestComplianceStatusFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  ComplianceStatus
		expectErr bool
	}{
		{"compliant", "Compliant", ComplianceStatusCompliant, false},
		{"non compliant", "NonCompliant", ComplianceStatusNonCompliant, false},
		{"pending", "Pending", ComplianceStatusPending, false},
		{"invalid", "Unknown", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ComplianceStatusFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidComplianceStatus)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPTOStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ps       PTOStatus
		expected string
	}{
		{"requested", PTOStatusRequested, "Requested"},
		{"approved", PTOStatusApproved, "Approved"},
		{"rejected", PTOStatusRejected, "Rejected"},
		{"cancelled", PTOStatusCancelled, "Cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ps.String())
		})
	}
}

func TestPTOStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ps       PTOStatus
		expected bool
	}{
		{"requested is valid", PTOStatusRequested, true},
		{"approved is valid", PTOStatusApproved, true},
		{"rejected is valid", PTOStatusRejected, true},
		{"cancelled is valid", PTOStatusCancelled, true},
		{"empty is invalid", PTOStatus(""), false},
		{"unknown is invalid", PTOStatus("Pending"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ps.IsValid())
		})
	}
}

func TestPTOStatusFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  PTOStatus
		expectErr bool
	}{
		{"requested", "Requested", PTOStatusRequested, false},
		{"approved", "Approved", PTOStatusApproved, false},
		{"rejected", "Rejected", PTOStatusRejected, false},
		{"cancelled", "Cancelled", PTOStatusCancelled, false},
		{"invalid", "Pending", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := PTOStatusFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidPTOStatus)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPTOType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pt       PTOType
		expected string
	}{
		{"personal", PTOTypePersonal, "Personal"},
		{"vacation", PTOTypeVacation, "Vacation"},
		{"sick", PTOTypeSick, "Sick"},
		{"holiday", PTOTypeHoliday, "Holiday"},
		{"bereavement", PTOTypeBereavement, "Bereavement"},
		{"maternity", PTOTypeMaternity, "Maternity"},
		{"paternity", PTOTypePaternity, "Paternity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.pt.String())
		})
	}
}

func TestPTOType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pt       PTOType
		expected bool
	}{
		{"personal is valid", PTOTypePersonal, true},
		{"vacation is valid", PTOTypeVacation, true},
		{"sick is valid", PTOTypeSick, true},
		{"holiday is valid", PTOTypeHoliday, true},
		{"bereavement is valid", PTOTypeBereavement, true},
		{"maternity is valid", PTOTypeMaternity, true},
		{"paternity is valid", PTOTypePaternity, true},
		{"empty is invalid", PTOType(""), false},
		{"unknown is invalid", PTOType("Jury"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.pt.IsValid())
		})
	}
}

func TestPTOTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  PTOType
		expectErr bool
	}{
		{"personal", "Personal", PTOTypePersonal, false},
		{"vacation", "Vacation", PTOTypeVacation, false},
		{"sick", "Sick", PTOTypeSick, false},
		{"holiday", "Holiday", PTOTypeHoliday, false},
		{"bereavement", "Bereavement", PTOTypeBereavement, false},
		{"maternity", "Maternity", PTOTypeMaternity, false},
		{"paternity", "Paternity", PTOTypePaternity, false},
		{"invalid", "Jury", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := PTOTypeFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidPTOType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGender_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		g        Gender
		expected string
	}{
		{"male", GenderMale, "Male"},
		{"female", GenderFemale, "Female"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.g.String())
		})
	}
}

func TestGender_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		g        Gender
		expected bool
	}{
		{"male is valid", GenderMale, true},
		{"female is valid", GenderFemale, true},
		{"empty is invalid", Gender(""), false},
		{"other is invalid", Gender("Other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.g.IsValid())
		})
	}
}

func TestGenderFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  Gender
		expectErr bool
	}{
		{"male", "Male", GenderMale, false},
		{"female", "Female", GenderFemale, false},
		{"other is invalid", "Other", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GenderFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidGender)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCDLClass_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		c        CDLClass
		expected string
	}{
		{"class A", CDLClassA, "A"},
		{"class B", CDLClassB, "B"},
		{"class C", CDLClassC, "C"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.c.String())
		})
	}
}

func TestCDLClass_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		c        CDLClass
		expected bool
	}{
		{"class A is valid", CDLClassA, true},
		{"class B is valid", CDLClassB, true},
		{"class C is valid", CDLClassC, true},
		{"empty is invalid", CDLClass(""), false},
		{"unknown is invalid", CDLClass("D"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.c.IsValid())
		})
	}
}

func TestCDLClassFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  CDLClass
		expectErr bool
	}{
		{"class A", "A", CDLClassA, false},
		{"class B", "B", CDLClassB, false},
		{"class C", "C", CDLClassC, false},
		{"invalid", "D", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := CDLClassFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidCDLClass)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDriverType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dt       DriverType
		expected string
	}{
		{"local", DriverTypeLocal, "Local"},
		{"regional", DriverTypeRegional, "Regional"},
		{"OTR", DriverTypeOTR, "OTR"},
		{"team", DriverTypeTeam, "Team"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.dt.String())
		})
	}
}

func TestDriverType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dt       DriverType
		expected bool
	}{
		{"local is valid", DriverTypeLocal, true},
		{"regional is valid", DriverTypeRegional, true},
		{"OTR is valid", DriverTypeOTR, true},
		{"team is valid", DriverTypeTeam, true},
		{"empty is invalid", DriverType(""), false},
		{"unknown is invalid", DriverType("Dedicated"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.dt.IsValid())
		})
	}
}

func TestDriverTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  DriverType
		expectErr bool
	}{
		{"local", "Local", DriverTypeLocal, false},
		{"regional", "Regional", DriverTypeRegional, false},
		{"OTR", "OTR", DriverTypeOTR, false},
		{"team", "Team", DriverTypeTeam, false},
		{"invalid", "Dedicated", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := DriverTypeFromString(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidDriverType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
