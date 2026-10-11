package agentdefinition

import (
	"slices"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// AccessMode says who may use an agent, among the people who may use the
// assistant at all. Open is a stored state rather than the absence of
// grants, so taking the last role off an agent restricted to roles leaves
// it usable by nobody instead of by everybody.
type AccessMode string

const (
	AccessEveryone = AccessMode("Everyone")
	AccessRoles    = AccessMode("Roles")
)

func (m AccessMode) IsValid() bool {
	switch m {
	case AccessEveryone, AccessRoles:
		return true
	default:
		return false
	}
}

func AllAccessModes() []AccessMode {
	return []AccessMode{AccessEveryone, AccessRoles}
}

// AudienceCoverage is how much of an agent a role could use: all of its
// tools, some of them, or none because the role cannot use the assistant.
type AudienceCoverage string

const (
	CoverageFull    = AudienceCoverage("Full")
	CoveragePartial = AudienceCoverage("Partial")
	CoverageNone    = AudienceCoverage("None")
)

func (c AudienceCoverage) IsValid() bool {
	switch c {
	case CoverageFull, CoveragePartial, CoverageNone:
		return true
	default:
		return false
	}
}

// EffectiveAccessMode is who may use the agent as the checks read it. A
// system agent is open to everyone whatever its row says: the platform fires
// it for people, and restricting it would break the feature it serves.
func (d *Definition) EffectiveAccessMode() AccessMode {
	if d.IsSystem() || d.AccessMode == "" {
		return AccessEveryone
	}

	return d.AccessMode
}

// OpenToEveryone reports whether everyone who may use the assistant may use
// the agent.
func (d *Definition) OpenToEveryone() bool {
	return d.EffectiveAccessMode() == AccessEveryone
}

// UsableWith reports whether a person whose roles grant the agents named in
// granted may use this one, given that they may use the assistant.
func (d *Definition) UsableWith(granted []pulid.ID) bool {
	return d.OpenToEveryone() || slices.Contains(granted, d.ID)
}

// AccessRefusal says why the agent cannot take the access mode, or "" when
// it can.
func (d *Definition) AccessRefusal(mode AccessMode) string {
	switch {
	case !mode.IsValid():
		return "Access must be Everyone or Roles"
	case mode == AccessRoles && d.IsSystem():
		return d.Name + " is a system agent and is always open to everyone who can use " +
			"the assistant"
	default:
		return ""
	}
}

func (d *Definition) validateAccess(multiErr *errortypes.MultiError) {
	if d.AccessMode == "" {
		return
	}
	if refusal := d.AccessRefusal(d.AccessMode); refusal != "" {
		multiErr.Add("accessMode", errortypes.ErrInvalid, refusal)
	}
}
