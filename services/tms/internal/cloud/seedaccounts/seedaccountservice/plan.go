package seedaccountservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type UserAction string

const (
	UserActionRetire         = UserAction("retire")
	UserActionAlreadyRetired = UserAction("already_retired")
	UserActionUnproven       = UserAction("unproven")
)

type Provenance string

const (
	ProvenanceSeedTracking   = Provenance("seed_tracking")
	ProvenanceSeededPassword = Provenance("seeded_password")
)

type OrganizationAction string

const (
	OrganizationActionRemove = OrganizationAction("remove")
	OrganizationActionReview = OrganizationAction("review")
)

type UserReport struct {
	Footprint       *seedaccountport.UserFootprint    `json:"footprint"`
	Provenance      []Provenance                      `json:"provenance"`
	Action          UserAction                        `json:"action"`
	Delete          bool                              `json:"delete"`
	Stripped        *seedaccountport.StripUserResult  `json:"stripped,omitempty"`
	Removed         *seedaccountport.RemoveUserResult `json:"removed,omitempty"`
	SessionsRevoked bool                              `json:"sessionsRevoked"`
}

type OrganizationReport struct {
	Footprint *seedaccountport.OrganizationFootprint `json:"footprint"`
	Action    OrganizationAction                     `json:"action"`
	Reasons   []string                               `json:"reasons"`
	Removed   bool                                   `json:"removed"`
}

type Report struct {
	Applied       bool                        `json:"applied"`
	SystemUser    *seedaccountport.SystemUser `json:"systemUser,omitempty"`
	Users         []*UserReport               `json:"users"`
	Organizations []*OrganizationReport       `json:"organizations"`
	AuditEntries  int                         `json:"auditEntries"`
	Errors        []string                    `json:"errors"`
}

func (r *Report) retiring() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.Users))
	for _, user := range r.Users {
		if user.Action == UserActionRetire {
			ids = append(ids, user.Footprint.ID)
		}
	}

	return ids
}

func (r *Report) leaving() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.Users))
	for _, user := range r.Users {
		if user.Action == UserActionRetire && user.Delete {
			ids = append(ids, user.Footprint.ID)
		}
	}

	return ids
}

func (r *Report) removedOrganization(id pulid.ID) bool {
	for _, org := range r.Organizations {
		if org.Removed && org.Footprint.ID == id {
			return true
		}
	}

	return false
}

func knownAccount(fp *seedaccountport.UserFootprint) bool {
	for _, account := range seedaccountport.KnownAccounts() {
		if fp.Username == account.Username &&
			strings.EqualFold(fp.EmailAddress, account.EmailAddress) {
			return true
		}
	}

	return false
}

func knownOrganization(fp *seedaccountport.OrganizationFootprint) bool {
	for _, org := range seedaccountport.KnownOrganizations() {
		if fp.Name == org.Name && fp.ScacCode == org.ScacCode {
			return true
		}
	}

	return false
}

func retired(fp *seedaccountport.UserFootprint) bool {
	return fp.Status == domaintypes.StatusInactive &&
		fp.IsLocked &&
		len(fp.Memberships) == 0 &&
		len(fp.RoleAssignments) == 0 &&
		len(fp.ActiveAPIKeys) == 0 &&
		len(fp.MFAAuthenticators) == 0 &&
		fp.OpenResetTokens == 0
}

func classifyUser(
	fp *seedaccountport.UserFootprint,
	seededPassword func(hash string) bool,
) *UserReport {
	report := &UserReport{Footprint: fp, Provenance: make([]Provenance, 0, 2)}
	if fp.SeedTracked {
		report.Provenance = append(report.Provenance, ProvenanceSeedTracking)
	}
	if fp.PasswordHash != "" && seededPassword(fp.PasswordHash) {
		report.Provenance = append(report.Provenance, ProvenanceSeededPassword)
	}

	switch {
	case retired(fp):
		report.Action = UserActionAlreadyRetired
	case len(report.Provenance) == 0:
		report.Action = UserActionUnproven
	default:
		report.Action = UserActionRetire
	}

	return report
}

func classifyOrganization(
	fp *seedaccountport.OrganizationFootprint,
	system *seedaccountport.SystemUser,
) *OrganizationReport {
	reasons := make([]string, 0, 4)
	if !fp.SeedTracked {
		reasons = append(reasons, "not recorded in seed_created_entities as created by the "+
			seedaccountport.SeedName+" seed")
	}
	if system != nil && system.OrganizationID == fp.ID {
		reasons = append(reasons, "hosts the instance system user")
	}
	if len(fp.Users) > 0 {
		reasons = append(reasons, "still has users: "+strings.Join(fp.Users, ", "))
	}
	if len(fp.History) > 0 {
		reasons = append(
			reasons,
			"holds append-only audit history: "+DescribeReferences(fp.History),
		)
	}
	if len(fp.Data) > 0 {
		reasons = append(reasons, "holds data beyond seed defaults: "+DescribeReferences(fp.Data))
	}

	action := OrganizationActionRemove
	if len(reasons) > 0 {
		action = OrganizationActionReview
	}

	return &OrganizationReport{Footprint: fp, Action: action, Reasons: reasons}
}

func DescribeReferences(refs []seedaccountport.Reference) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts = append(
			parts,
			fmt.Sprintf("%s.%s (%s)", ref.Table, ref.Column, DescribeRows(ref.Rows)),
		)
	}

	return strings.Join(parts, ", ")
}

func DescribeRows(rows int64) string {
	if rows >= seedaccountport.ReferenceCap {
		return strconv.FormatInt(seedaccountport.ReferenceCap, 10) + "+"
	}

	return strconv.FormatInt(rows, 10)
}
