package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The fields of a medical, drug and alcohol or injury record that say what
// happened to a person stay Restricted, while what scheduling and the posted
// log need is Internal. An agent at the default data access, and a person
// whose role reaches only Internal, is shown the second and never the first.
func TestMedicalRecordsKeepWhatHappenedToThePersonRestricted(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	cases := map[Resource]struct {
		internal   []string
		restricted []string
	}{
		ResourceWorkerDOTTest: {
			internal: []string{
				"testType", "isDot", "reason", "status", "scheduledAt", "collectedAt",
				"resultAt", "drawEntryId",
			},
			restricted: []string{
				"result", "substance", "alcoholConcentration", "collectorName",
				"collectionSite", "specimenId", "labName", "mroName", "mroVerifiedAt",
				"notes", "documentId",
			},
		},
		ResourceWorkerLeave: {
			internal: []string{
				"leaveType", "status", "frequency", "fmlaDesignated", "startsAt", "endsAt",
				"certificationStatus", "certificationDueAt", "usedOn", "hours",
				"countsAgainstEntitlement",
			},
			restricted: []string{
				"reason", "notes", "militaryCaregiver", "eligibilityHoursWorked", "documentId",
			},
		},
		ResourceQualification: {
			internal: []string{
				"employerName", "employerDotNumber", "employedFrom", "employedTo", "status",
				"method", "requestedAt", "followUpCount", "hadAccidents", "accidentCount",
			},
			restricted: []string{
				"contactName", "contactPhone", "contactEmail", "hadDrugAlcoholViolations",
				"findings", "notes", "documentId",
			},
		},
		ResourceWorkerCredential: {
			internal: []string{
				"workerId", "credentialTypeId", "status", "issuingAuthority", "issuedAt",
				"expiresAt", "health", "daysUntilExpiry", "documentId", "verifiedAt",
			},
			restricted: []string{"notes", "archiveReason"},
		},
		ResourceWorkerInjury: {
			internal: []string{
				"caseNumber", "caseYear", "classification", "illnessType", "status",
				"recordable", "occurredAt", "daysAway", "daysRestricted", "privacyCase",
				"logName", "claimStatus",
			},
			restricted: []string{
				"treatment", "description", "bodyPart", "harmfulAgent", "location",
				"claimNumber", "claimCarrier", "notes", "documentId",
			},
		},
	}
	for resource, tc := range cases {
		for _, field := range tc.internal {
			assert.Equal(t, SensitivityInternal,
				registry.GetFieldSensitivity(resource.String(), field), "%s.%s", resource, field)
		}
		for _, field := range tc.restricted {
			assert.Equal(t, SensitivityRestricted,
				registry.GetFieldSensitivity(resource.String(), field), "%s.%s", resource, field)
		}
	}
}
