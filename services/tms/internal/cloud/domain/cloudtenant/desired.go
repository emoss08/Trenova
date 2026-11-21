package cloudtenant

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

type Lifecycle string

const (
	LifecycleActive    = Lifecycle("active")
	LifecycleSuspended = Lifecycle("suspended")
	LifecycleRemoved   = Lifecycle("removed")
)

type AIProvider string

const (
	AIProviderTrenova  = AIProvider("trenova")
	AIProviderCustomer = AIProvider("customer")
)

const (
	CommandArchive       = "archive"
	CommandRestore       = "restore"
	CommandImport        = "import"
	ArchivePurposeFinal  = "final"
	maxFeatures          = 200
	maxAddons            = 50
	maxAddonEndings      = 50
	maxAllowances        = 20
	maxCommands          = 20
	maxNameLength        = 100
	maxEmailLength       = 254
	maxPlanFieldLength   = 100
	maxReasonLength      = 500
	maxMeterLength       = 40
	maxSupportLevel      = 40
	maxFinalBackupDays   = 3650
	maxRetentionDays     = 3650
	maxAddonCodeLength   = 40
	maxFeatureKeyLength  = 100
	maxCommandKindLength = 40
	maxLocationLength    = 1024
)

var (
	ErrInstanceIDMismatch = errors.New(
		"the entry's instanceId must be the lowercase UUID the path names",
	)
	ErrRevisionInvalid = errors.New("the entry's revision must be 1 or more")

	tokenSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Plan struct {
	Key                 string `json:"key"`
	Name                string `json:"name"`
	BackupRetentionDays int64  `json:"backupRetentionDays"`
	Support             string `json:"support"`
}

type AddonEnding struct {
	Addon  string    `json:"addon"`
	EndsAt time.Time `json:"endsAt"`
}

type Allowance struct {
	Meter        string `json:"meter"`
	AppMeter     string `json:"appMeter,omitempty"`
	Unit         string `json:"unit"`
	Quantity     int64  `json:"quantity"`
	HardCap      bool   `json:"hardCap"`
	Ceiling      *int64 `json:"ceiling,omitempty"`
	AlwaysBilled *bool  `json:"alwaysBilled,omitempty"`
}

type Invite struct {
	TokenSHA256 string    `json:"tokenSha256"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type FirstAdministrator struct {
	Name             string  `json:"name"`
	Email            string  `json:"email"`
	OrganizationName string  `json:"organizationName"`
	Invite           *Invite `json:"invite,omitempty"`
}

type Account struct {
	PlanURL              string     `json:"planUrl"`
	SupportURL           string     `json:"supportUrl,omitempty"`
	NewSupportRequestURL string     `json:"newSupportRequestUrl,omitempty"`
	FinalBackupDays      int        `json:"finalBackupDays"`
	EndsAt               *time.Time `json:"endsAt,omitempty"`
}

type Command struct {
	Kind         string    `json:"kind"`
	ID           string    `json:"id"`
	Purpose      string    `json:"purpose,omitempty"`
	Location     string    `json:"location,omitempty"`
	ExpectSHA256 string    `json:"expectSha256,omitempty"`
	RequestedAt  time.Time `json:"requestedAt"`
}

type Desired struct {
	InstanceID         string             `json:"instanceId"`
	Revision           int64              `json:"revision"`
	Lifecycle          Lifecycle          `json:"lifecycle"`
	OrganizationName   string             `json:"organizationName"`
	Plan               Plan               `json:"plan"`
	Addons             []string           `json:"addons"`
	Features           []string           `json:"features"`
	AddonEndings       []AddonEnding      `json:"addonEndings,omitempty"`
	AIProvider         AIProvider         `json:"aiProvider"`
	Allowances         []Allowance        `json:"allowances"`
	FirstAdministrator FirstAdministrator `json:"firstAdministrator"`
	Account            Account            `json:"account"`
	Commands           []Command          `json:"commands"`
}

func ValidInstanceID(raw string) bool {
	parsed, err := uuid.Parse(raw)
	return err == nil && parsed.String() == raw
}

func (d *Desired) CheckAddress(pathInstanceID string) error {
	if !ValidInstanceID(pathInstanceID) || d.InstanceID != pathInstanceID {
		return ErrInstanceIDMismatch
	}
	if d.Revision < 1 {
		return ErrRevisionInvalid
	}
	return nil
}

func (d *Desired) Digest() (string, error) {
	body, err := sonic.ConfigStd.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("encode the tenant entry: %w", err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func (d *Desired) Refusal() *Refusal {
	problems := make([]string, 0)
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	switch d.Lifecycle {
	case LifecycleActive, LifecycleSuspended, LifecycleRemoved:
	default:
		add("lifecycle must be active, suspended or removed")
	}
	if !boundedText(d.OrganizationName, maxNameLength) {
		add("organizationName must be 1 to %d characters", maxNameLength)
	}
	if !boundedText(d.Plan.Key, maxPlanFieldLength) ||
		!boundedText(d.Plan.Name, maxPlanFieldLength) {
		add("plan must name its key and name")
	}
	if d.Plan.BackupRetentionDays < 1 || d.Plan.BackupRetentionDays > maxRetentionDays {
		add("plan.backupRetentionDays must be 1 to %d", maxRetentionDays)
	}
	if !boundedText(d.Plan.Support, maxSupportLevel) {
		add("plan.support is required")
	}
	if len(d.Addons) > maxAddons || !allBounded(d.Addons, maxAddonCodeLength) {
		add("addons must be at most %d codes", maxAddons)
	}
	if len(d.Features) > maxFeatures || !allBounded(d.Features, maxFeatureKeyLength) {
		add("features must be at most %d keys", maxFeatures)
	}
	d.endingProblems(add)
	d.allowanceProblems(add)
	d.administratorProblems(add)
	if d.Account.FinalBackupDays < 1 || d.Account.FinalBackupDays > maxFinalBackupDays {
		add("account.finalBackupDays must be 1 to %d", maxFinalBackupDays)
	}
	if len(d.Commands) > maxCommands {
		add("commands must be at most %d", maxCommands)
	}
	imports := 0
	for _, c := range d.Commands {
		if !boundedText(c.Kind, maxCommandKindLength) || !ValidInstanceID(c.ID) {
			add("each command names its kind and a lowercase UUID id")
			break
		}
		if c.Kind == CommandImport {
			imports++
			if !tokenSHA256Pattern.MatchString(c.ExpectSHA256) ||
				!boundedText(c.Location, maxLocationLength) {
				add("an import command names its location and the archive's lowercase hex SHA-256")
			}
		}
	}
	if imports > 1 {
		add("an entry carries at most one import command")
	}

	if len(problems) == 0 {
		return nil
	}
	return Invalid(d.Revision, strings.Join(problems, "; "))
}

func (d *Desired) allowanceProblems(add func(string, ...any)) {
	switch d.AIProvider {
	case AIProviderTrenova, AIProviderCustomer:
	default:
		add("aiProvider must be trenova or customer")
	}
	if len(d.Allowances) > maxAllowances {
		add("allowances must be at most %d", maxAllowances)
	}
	for _, a := range d.Allowances {
		if !boundedText(a.Meter, maxMeterLength) || (a.Unit != "count" && a.Unit != "bytes") ||
			a.Quantity < 0 || (a.Ceiling != nil && *a.Ceiling < a.Quantity) {
			add(
				"each allowance names a meter, a count or bytes unit, a quantity and a ceiling at or above it",
			)
			return
		}
	}
}

func (d *Desired) endingProblems(add func(string, ...any)) {
	if len(d.AddonEndings) > maxAddonEndings {
		add("addonEndings must be at most %d", maxAddonEndings)
	}
	for _, e := range d.AddonEndings {
		if !boundedText(e.Addon, maxAddonCodeLength) || e.EndsAt.IsZero() {
			add("each addonEnding names an add-on and when it ends")
			return
		}
	}
}

func (d *Desired) administratorProblems(add func(string, ...any)) {
	admin := d.FirstAdministrator
	if !boundedText(admin.Name, maxNameLength) {
		add("firstAdministrator.name must be 1 to %d characters", maxNameLength)
	}
	if !boundedText(admin.Email, maxEmailLength) {
		add("firstAdministrator.email must be 1 to %d characters", maxEmailLength)
	}
	if admin.OrganizationName != "" && admin.OrganizationName != d.OrganizationName {
		add("firstAdministrator.organizationName must be the tenant's organizationName")
	}
	if admin.Invite != nil &&
		(!tokenSHA256Pattern.MatchString(admin.Invite.TokenSHA256) || admin.Invite.ExpiresAt.IsZero()) {
		add("firstAdministrator.invite must carry a lowercase hex SHA-256 and when it expires")
	}
}

func (d *Desired) ImportCommand() *Command {
	for i := range d.Commands {
		if d.Commands[i].Kind == CommandImport {
			return &d.Commands[i]
		}
	}
	return nil
}

func (d *Desired) SameIdentity(held *Desired) bool {
	return held == nil ||
		(held.OrganizationName == d.OrganizationName &&
			strings.EqualFold(
				strings.TrimSpace(held.FirstAdministrator.Email),
				strings.TrimSpace(d.FirstAdministrator.Email),
			) &&
			held.FirstAdministrator.Name == d.FirstAdministrator.Name)
}

func boundedText(s string, maxLength int) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed != "" && utf8.RuneCountInString(s) <= maxLength &&
		!strings.ContainsFunc(s, isControl)
}

func allBounded(values []string, maxLength int) bool {
	return !slices.ContainsFunc(values, func(v string) bool { return !boundedText(v, maxLength) })
}

func isControl(r rune) bool {
	return r < 0x20 || r == 0x7f
}
