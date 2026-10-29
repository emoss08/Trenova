// Package pageaccessservice answers what a screen needs and what a person or
// a role would see on it, from the page access manifest (pkg/pageaccess) and
// the reference views (refview), and words what granting or withholding
// each resource means for the role editor. It decides nothing itself: every
// field goes through refview with the caller's or the role's grants, and it
// never reads a record.
package pageaccessservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

// maxGuideResources bounds one role editor request: a category's resources.
const maxGuideResources = 64

type Params struct {
	fx.In

	Engine   permissionports.PermissionEngine
	Roles    permissionports.RoleRepository
	Registry *permissioncatalog.Registry
}

type Service struct {
	engine   permissionports.PermissionEngine
	roles    permissionports.RoleRepository
	registry *permissioncatalog.Registry
	views    *refview.Views
	manifest *pageaccess.Manifest
	// readBy is how many screens read each kind as root records.
	readBy map[string]int
}

func New(p Params) *Service {
	return NewWithManifest(p, pageaccess.Default)
}

// NewWithManifest is New over a given manifest, for tests.
func NewWithManifest(p Params, manifest *pageaccess.Manifest) *Service {
	readBy := make(map[string]int)
	for i := range manifest.Screens {
		for _, kind := range manifest.Screens[i].Reads {
			readBy[kind]++
		}
	}
	return &Service{
		engine:   p.Engine,
		roles:    p.Roles,
		registry: p.Registry,
		views:    refview.New(p.Registry),
		manifest: manifest,
		readBy:   readBy,
	}
}

type Decision string

const (
	DecisionVisible             Decision = "visible"
	DecisionWithheldKind        Decision = "withheld_kind"
	DecisionWithheldSensitivity Decision = "withheld_sensitivity"
)

type Requirement struct {
	Resource      string
	ResourceName  string
	Operation     string
	OperationName string
	Allowed       bool
}

type Endpoint struct {
	Method   string
	Path     string
	Requires []Requirement
}

// Field is one field a screen selects on a linked record of a kind. A field
// written Type.field is of a record that is part of one of the kind.
type Field struct {
	Name     string
	InView   bool
	Decision Decision
}

type Kind struct {
	Resource     string
	ResourceName string
	Use          pageaccess.Use
	CanRead      bool
	Fields       []Field
	Withheld     int
}

// Access is what a screen needs and what a person or a role would see on
// it. Blocked is set when the route, or a kind the screen reads as root
// records, is refused: the screen would not open, or would open empty.
type Access struct {
	Path       string
	Name       string
	Operations []string
	Requires   []Requirement
	Endpoints  []Endpoint
	Kinds      []Kind
	Withheld   int
	Blocked    bool
}

// Screen is one screen of the manifest, for choosing one to preview.
type Screen struct {
	Path string
	Name string
}

// Screens is every screen of the manifest, by path.
func (s *Service) Screens() []Screen {
	out := make([]Screen, 0, len(s.manifest.Screens))
	for i := range s.manifest.Screens {
		out = append(
			out,
			Screen{Path: s.manifest.Screens[i].Path, Name: s.manifest.Screens[i].Name},
		)
	}
	return out
}

// access is a caller's or a role's grants, as a screen is described.
type access interface {
	refview.Access
	allows(ctx context.Context, resource string, op permission.Operation) bool
}

// ForCaller is the caller's own access to the screen at a location, or nil
// when no screen is there.
func (s *Service) ForCaller(
	ctx context.Context,
	actor *permissionports.RequestActor,
	location string,
) (*Access, error) {
	screen, ok := s.manifest.Screen(location)
	if !ok {
		return nil, nil //nolint:nilnil // no screen is at that location
	}
	person, err := s.person(ctx, actor, s.checksFor(screen))
	if err != nil {
		return nil, err
	}

	return s.describe(ctx, screen, person), nil
}

// RoleDraft is a role as the editor holds it before it is saved.
type RoleDraft struct {
	MaxSensitivity permission.FieldSensitivity
	ParentRoleIDs  []pulid.ID
	Permissions    []DraftPermission
}

type DraftPermission struct {
	Resource   string
	Operations []permission.Operation
}

type PreviewRequest struct {
	Tenant   pagination.TenantInfo
	Location string
	// RoleID previews a saved role; Draft one being edited. Exactly one is
	// set.
	RoleID pulid.ID
	Draft  *RoleDraft
}

// Preview is what a person holding only a role (and the roles it inherits)
// would need and see on the screen at a location, or nil when no screen is
// there. It is worked out from the role's grants and ceiling through
// refview, before any organization access policy, and reads no record.
func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*Access, error) {
	if strings.TrimSpace(req.Location) == "" {
		return nil, errortypes.NewValidationError("location", errortypes.ErrRequired,
			"Choose a screen to preview")
	}
	grants, err := s.grants(ctx, req.Tenant, req.RoleID, req.Draft)
	if err != nil {
		return nil, err
	}
	screen, ok := s.manifest.Screen(req.Location)
	if !ok {
		return nil, nil //nolint:nilnil // no screen is at that location
	}

	return s.describe(
			ctx,
			screen,
			roleAccess{RoleAccess: refview.NewRoleAccess(s.registry, grants)},
		),
		nil
}

func (s *Service) grants(
	ctx context.Context,
	tenant pagination.TenantInfo,
	roleID pulid.ID,
	draft *RoleDraft,
) (permissioncatalog.RoleGrants, error) {
	if (roleID.IsNil()) == (draft == nil) {
		return permissioncatalog.RoleGrants{}, errortypes.NewValidationError(
			"roleId", errortypes.ErrInvalid, "Choose a saved role or send the role being edited",
		)
	}

	var role *rbac.Role
	if draft != nil {
		built, err := s.draftRole(draft)
		if err != nil {
			return permissioncatalog.RoleGrants{}, err
		}
		role = built
	} else {
		saved, err := s.roles.GetByID(ctx, permissionports.GetRoleByIDRequest{
			ID:         roleID,
			TenantInfo: tenant,
		})
		if err != nil {
			return permissioncatalog.RoleGrants{}, err
		}
		role = saved
	}

	roles := []*rbac.Role{role}
	if len(role.ParentRoleIDs) > 0 {
		inherited, err := s.roles.GetRolesWithInheritance(ctx, role.ParentRoleIDs)
		if err != nil {
			return permissioncatalog.RoleGrants{}, err
		}
		found := make(map[pulid.ID]struct{}, len(inherited))
		for _, parent := range inherited {
			if parent.OrganizationID.IsNotNil() && parent.OrganizationID != tenant.OrgID {
				return permissioncatalog.RoleGrants{}, foreignParentError()
			}
			found[parent.ID] = struct{}{}
		}
		for _, id := range role.ParentRoleIDs {
			if _, ok := found[id]; !ok {
				return permissioncatalog.RoleGrants{}, foreignParentError()
			}
		}
		roles = append(roles, inherited...)
	}

	return permissioncatalog.GrantsOf(roles...), nil
}

func foreignParentError() error {
	return errortypes.NewValidationError(
		"parentRoleIds",
		errortypes.ErrInvalid,
		"Inherited roles must belong to your organization",
	)
}

func (s *Service) draftRole(draft *RoleDraft) (*rbac.Role, error) {
	multiErr := errortypes.NewMultiError()
	if !draft.MaxSensitivity.IsValid() {
		multiErr.Add("maxSensitivity", errortypes.ErrInvalid, "Choose a valid sensitivity")
	}
	role := &rbac.Role{
		MaxSensitivity: draft.MaxSensitivity,
		ParentRoleIDs:  draft.ParentRoleIDs,
		Permissions:    make([]*rbac.ResourcePermission, 0, len(draft.Permissions)),
	}
	for _, grant := range draft.Permissions {
		if _, ok := s.registry.Get(grant.Resource); !ok {
			multiErr.Add(
				"permissions",
				errortypes.ErrInvalid,
				"A permission names an unknown resource",
			)
			continue
		}
		for _, op := range grant.Operations {
			if !s.registry.Declares(grant.Resource, op) {
				multiErr.Add("permissions", errortypes.ErrInvalid,
					"A permission names an operation its resource does not have")
			}
		}
		role.Permissions = append(role.Permissions, &rbac.ResourcePermission{
			Resource:   grant.Resource,
			Operations: grant.Operations,
		})
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return role, nil
}

type roleAccess struct {
	refview.RoleAccess
}

func (a roleAccess) allows(_ context.Context, resource string, op permission.Operation) bool {
	return a.Allows(resource, op)
}

// personAccess is a person's grants for the checks a screen needs, asked of
// the engine in one batch.
type personAccess struct {
	*refview.PersonAccess
	allowed map[check]bool
}

type check struct {
	resource string
	op       permission.Operation
}

func (a personAccess) allows(_ context.Context, resource string, op permission.Operation) bool {
	return a.allowed[check{resource: resource, op: op}]
}

func (s *Service) person(
	ctx context.Context,
	actor *permissionports.RequestActor,
	checks []check,
) (personAccess, error) {
	allowed := make(map[check]bool, len(checks))
	if len(checks) > 0 && actor != nil {
		batch := make([]permissionports.ResourceOperationCheck, 0, len(checks))
		for _, c := range checks {
			batch = append(batch, permissionports.ResourceOperationCheck{
				Resource:  c.resource,
				Operation: c.op,
			})
		}
		result, err := s.engine.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			UserID:         actor.UserID,
			APIKeyID:       actor.APIKeyID,
			BusinessUnitID: actor.BusinessUnitID,
			OrganizationID: actor.OrganizationID,
			Checks:         batch,
		})
		if err != nil {
			return personAccess{}, err
		}
		for i, c := range checks {
			allowed[c] = i < len(result.Results) && result.Results[i].Allowed
		}
	}
	reads := func(_ context.Context, kind permission.Resource) (bool, error) {
		return allowed[check{resource: kind.String(), op: permission.OpRead}], nil
	}

	return personAccess{
		PersonAccess: refview.NewPersonAccess(s.engine, actor, reads),
		allowed:      allowed,
	}, nil
}

// checksFor is every check a screen's description asks: what its route and
// its REST endpoints demand, and the read of every kind it shows.
func (s *Service) checksFor(screen *pageaccess.Screen) []check {
	var checks []check
	add := func(resource string, op permission.Operation) {
		c := check{resource: resource, op: op}
		if !slices.Contains(checks, c) {
			checks = append(checks, c)
		}
	}
	for _, r := range screen.Requires {
		add(r.Resource, permission.Operation(r.Operation))
	}
	for _, e := range screen.Endpoints {
		for _, r := range e.Requires {
			add(r.Resource, permission.Operation(r.Operation))
		}
	}
	for _, kind := range s.kindsOf(screen) {
		add(kind, permission.OpRead)
	}
	return checks
}

// kindsOf is every kind a screen depends on: what it requires, reads,
// declares or shows through a view.
func (s *Service) kindsOf(screen *pageaccess.Screen) []string {
	var kinds []string
	add := func(kind string) {
		if !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	for _, r := range screen.Requires {
		add(r.Resource)
	}
	for _, e := range screen.Endpoints {
		for _, r := range e.Requires {
			add(r.Resource)
		}
	}
	for _, kind := range screen.Reads {
		add(kind)
	}
	for _, kind := range screen.Declared {
		add(kind)
	}
	for _, kind := range screen.References {
		add(kind)
	}
	for _, linked := range s.manifest.Linked(screen) {
		add(linked.Kind)
	}
	return kinds
}

var useOrder = []pageaccess.Use{
	pageaccess.UseRequired,
	pageaccess.UseRead,
	pageaccess.UseDeclared,
	pageaccess.UseReference,
}

func (s *Service) describe(ctx context.Context, screen *pageaccess.Screen, a access) *Access {
	out := &Access{
		Path:       screen.Path,
		Name:       screen.Name,
		Operations: slices.Clone(screen.Operations),
		Requires:   s.requirements(ctx, screen.Requires, a),
		Endpoints:  make([]Endpoint, 0, len(screen.Endpoints)),
	}
	for _, r := range out.Requires {
		out.Blocked = out.Blocked || !r.Allowed
	}
	for _, e := range screen.Endpoints {
		out.Endpoints = append(out.Endpoints, Endpoint{
			Method:   e.Method,
			Path:     e.Path,
			Requires: s.requirements(ctx, e.Requires, a),
		})
	}

	linked := make(map[string][]string)
	for _, l := range s.manifest.Linked(screen) {
		linked[l.Kind] = l.Fields
	}
	for _, kind := range s.kindsOf(screen) {
		use, ok := screen.UseOf(kind)
		if !ok {
			use = pageaccess.UseReference
		}
		resource := permission.Resource(kind)
		k := Kind{
			Resource:     kind,
			ResourceName: s.resourceName(ctx, kind),
			Use:          use,
			CanRead:      a.Reads(ctx, resource),
			Fields:       make([]Field, 0, len(linked[kind])),
		}
		for _, name := range linked[kind] {
			field := s.decide(ctx, a, resource, name)
			if field.Decision != DecisionVisible {
				k.Withheld++
			}
			k.Fields = append(k.Fields, field)
		}
		if use == pageaccess.UseRead && !k.CanRead {
			out.Blocked = true
		}
		out.Withheld += k.Withheld
		out.Kinds = append(out.Kinds, k)
	}
	slices.SortStableFunc(out.Kinds, func(x, y Kind) int {
		if d := slices.Index(useOrder, x.Use) - slices.Index(useOrder, y.Use); d != 0 {
			return d
		}
		return strings.Compare(x.ResourceName, y.ResourceName)
	})

	return out
}

func (s *Service) decide(
	ctx context.Context,
	a access,
	kind permission.Resource,
	name string,
) Field {
	var decision refview.Decision
	inView := false
	if _, field, part := strings.Cut(name, "."); part {
		decision = s.views.DecidePart(ctx, a, kind, field, refview.Nested)
	} else {
		inView = s.views.InView(kind, name)
		decision = s.views.Decide(ctx, a, kind, name, refview.Nested)
	}

	return Field{Name: name, InView: inView, Decision: decisionOf(decision)}
}

func decisionOf(d refview.Decision) Decision {
	switch d {
	case refview.Visible:
		return DecisionVisible
	case refview.WithheldSensitivity:
		return DecisionWithheldSensitivity
	case refview.WithheldKind:
		return DecisionWithheldKind
	default:
		return DecisionWithheldKind
	}
}

func (s *Service) requirements(
	ctx context.Context,
	requires []pageaccess.Requirement,
	a access,
) []Requirement {
	out := make([]Requirement, 0, len(requires))
	for _, r := range requires {
		op := permission.Operation(r.Operation)
		out = append(out, Requirement{
			Resource:      r.Resource,
			ResourceName:  s.resourceName(ctx, r.Resource),
			Operation:     r.Operation,
			OperationName: s.operationName(ctx, r.Resource, op),
			Allowed:       a.allows(ctx, r.Resource, op),
		})
	}
	return out
}

func (s *Service) resourceName(ctx context.Context, resource string) string {
	if def, ok := s.registry.Get(resource); ok {
		return i18n.T(ctx, def.DisplayName)
	}
	return resource
}

func (s *Service) operationName(
	ctx context.Context,
	resource string,
	op permission.Operation,
) string {
	if def, ok := s.registry.Get(resource); ok {
		for _, candidate := range def.Operations {
			if candidate.Operation == op {
				return i18n.T(ctx, candidate.DisplayName)
			}
		}
	}
	return string(op)
}
