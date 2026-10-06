package seedaccountservice

import (
	"context"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	auditSource       = "cloud_retire_seed_accounts"
	keySource         = "source"
	keyUsername       = "username"
	keyUserID         = "userId"
	keyOrganizationID = "organizationId"
	keyRecordedIn     = "recordedInOrganizationId"
	keyName           = "name"
	keyStatus         = "status"
)

type auditTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type auditEntry struct {
	change  *services.SecurityChange
	subject pulid.ID
}

func (s *Service) audit(ctx context.Context, report *Report) {
	entries := make([]auditEntry, 0)
	for _, user := range report.Users {
		if user.Action != UserActionRetire || user.Stripped == nil || user.Removed == nil {
			continue
		}
		entries = append(entries, userEntries(user)...)
	}
	for _, org := range report.Organizations {
		if !org.Removed {
			continue
		}
		entries = append(entries, organizationEntry(org))
	}

	for i := range entries {
		entry := entries[i]
		target, ok := s.auditTarget(report, entry.subject, entry.change.BusinessUnitID)
		if !ok {
			s.l.Error("no surviving organization to record a seed account retirement in",
				zap.String("resource", entry.change.Resource.String()),
				zap.String("resourceID", entry.change.ResourceID),
				zap.String("operation", string(entry.change.Operation)),
				zap.String("organizationID", entry.subject.String()),
				zap.String("comment", entry.change.Comment),
				zap.Any("metadata", entry.change.Metadata),
			)
			report.Errors = append(report.Errors,
				"no surviving organization to record \""+entry.change.Comment+"\" for "+
					entry.change.ResourceID+" in; it was logged instead")
			continue
		}

		if target.OrganizationID != entry.subject {
			entry.change.Metadata[keyRecordedIn] = target.OrganizationID.String()
		}
		entry.change.OrganizationID = target.OrganizationID
		entry.change.BusinessUnitID = target.BusinessUnitID
		s.auditor.RecordChange(ctx, entry.change)
		report.AuditEntries++
	}
}

func (s *Service) auditTarget(
	report *Report,
	orgID, buID pulid.ID,
) (auditTenant, bool) {
	if orgID.IsNotNil() && buID.IsNotNil() && !report.removedOrganization(orgID) {
		return auditTenant{OrganizationID: orgID, BusinessUnitID: buID}, true
	}
	system := report.SystemUser
	if system == nil || report.removedOrganization(system.OrganizationID) {
		return auditTenant{}, false
	}

	return auditTenant{
		OrganizationID: system.OrganizationID,
		BusinessUnitID: system.BusinessUnitID,
	}, true
}

func baseMetadata(user *seedaccountport.UserFootprint, orgID pulid.ID) map[string]any {
	return map[string]any{
		keySource:         auditSource,
		keyUsername:       user.Username,
		keyUserID:         user.ID.String(),
		keyOrganizationID: orgID.String(),
	}
}

func userEntries(user *UserReport) []auditEntry {
	fp := user.Footprint
	stripped := user.Stripped
	actor := services.SystemAuditActor()
	units := businessUnits(fp, stripped)

	entries := make([]auditEntry, 0,
		len(stripped.Memberships)+len(stripped.RoleAssignments)+len(stripped.APIKeys)+1)

	for _, membership := range stripped.Memberships {
		metadata := baseMetadata(fp, membership.OrganizationID)
		metadata["membershipId"] = membership.ID.String()
		entries = append(entries, auditEntry{
			subject: membership.OrganizationID,
			change: &services.SecurityChange{
				Resource:       permission.ResourceUser,
				ResourceID:     fp.ID.String(),
				Operation:      permission.OpUpdate,
				Actor:          actor,
				BusinessUnitID: membership.BusinessUnitID,
				Before: map[string]any{
					"membershipId":    membership.ID.String(),
					keyOrganizationID: membership.OrganizationID.String(),
				},
				After:    map[string]any{"membershipId": nil},
				Comment:  "Removed the organization membership of a legacy seeded account",
				Metadata: metadata,
			},
		})
	}

	for _, assignment := range stripped.RoleAssignments {
		metadata := baseMetadata(fp, assignment.OrganizationID)
		metadata["assignmentId"] = assignment.ID.String()
		entries = append(entries, auditEntry{
			subject: assignment.OrganizationID,
			change: &services.SecurityChange{
				Resource:       permission.ResourceRole,
				ResourceID:     assignment.RoleID.String(),
				Operation:      permission.OpUnassign,
				Actor:          actor,
				BusinessUnitID: units.of(assignment.OrganizationID),
				Before: map[string]any{
					keyUserID:      fp.ID.String(),
					"roleId":       assignment.RoleID.String(),
					"assignmentId": assignment.ID.String(),
				},
				Comment:  "Removed a role from a legacy seeded account",
				Metadata: metadata,
			},
		})
	}

	for _, key := range stripped.APIKeys {
		metadata := baseMetadata(fp, key.OrganizationID)
		metadata["keyPrefix"] = key.KeyPrefix
		entries = append(entries, auditEntry{
			subject: key.OrganizationID,
			change: &services.SecurityChange{
				Resource:       permission.ResourceAPIKey,
				ResourceID:     key.ID.String(),
				Operation:      permission.OpUpdate,
				Actor:          actor,
				BusinessUnitID: key.BusinessUnitID,
				Before:         map[string]any{keyStatus: "active", keyName: key.Name},
				After:          map[string]any{keyStatus: "revoked", keyName: key.Name},
				Comment:        "Revoked an API key created by a legacy seeded account",
				Metadata:       metadata,
			},
		})
	}

	entries = append(entries, dispositionEntry(user, actor))

	return entries
}

func dispositionEntry(user *UserReport, actor services.AuditActor) auditEntry {
	fp := user.Footprint
	metadata := baseMetadata(fp, fp.CurrentOrganizationID)
	metadata["mfaAuthenticatorsRemoved"] = len(user.Stripped.MFAAuthenticators)
	metadata["passwordResetTokensInvalidated"] = user.Stripped.ResetTokens
	metadata["provenance"] = user.Provenance

	before := map[string]any{
		keyStatus:      fp.Status,
		"isLocked":     fp.IsLocked,
		"username":     fp.Username,
		"emailAddress": fp.EmailAddress,
	}

	change := &services.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     fp.ID.String(),
		Actor:          actor,
		BusinessUnitID: fp.BusinessUnitID,
		Before:         before,
		Metadata:       metadata,
	}
	if user.Removed.Deleted {
		change.Operation = permission.OpDelete
		change.Comment = "Deleted a legacy seeded account that nothing referenced"
	} else {
		change.Operation = permission.OpLock
		change.Comment = "Disabled and locked a legacy seeded account and replaced its password " +
			"with an unusable one; it is kept because other rows reference it"
		change.After = map[string]any{
			keyStatus:            domaintypes.StatusInactive,
			"isLocked":           true,
			"mustChangePassword": true,
			"username":           fp.Username,
			"emailAddress":       fp.EmailAddress,
		}
		metadata["references"] = DescribeReferences(user.Removed.References)
	}

	return auditEntry{subject: fp.CurrentOrganizationID, change: change}
}

func organizationEntry(org *OrganizationReport) auditEntry {
	fp := org.Footprint

	return auditEntry{
		subject: fp.ID,
		change: &services.SecurityChange{
			Resource:       permission.ResourceOrganization,
			ResourceID:     fp.ID.String(),
			Operation:      permission.OpDelete,
			Actor:          services.SystemAuditActor(),
			BusinessUnitID: fp.BusinessUnitID,
			Before: map[string]any{
				keyName:    fp.Name,
				"scacCode": fp.ScacCode,
			},
			Comment: "Deleted a demo organization created by the legacy admin account seed",
			Metadata: map[string]any{
				keySource:         auditSource,
				keyOrganizationID: fp.ID.String(),
			},
		},
	}
}

type unitLookup map[pulid.ID]pulid.ID

func businessUnits(
	fp *seedaccountport.UserFootprint,
	stripped *seedaccountport.StripUserResult,
) unitLookup {
	units := make(unitLookup, len(stripped.Memberships)+1)
	units[fp.CurrentOrganizationID] = fp.BusinessUnitID
	for _, membership := range stripped.Memberships {
		units[membership.OrganizationID] = membership.BusinessUnitID
	}
	for _, key := range stripped.APIKeys {
		units[key.OrganizationID] = key.BusinessUnitID
	}

	return units
}

func (u unitLookup) of(orgID pulid.ID) pulid.ID {
	return u[orgID]
}
