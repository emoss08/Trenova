package edilading

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxLookupRounds bounds how often a run is repeated to answer the lookups it needs: one
	// round answers every code the map reached, a second the codes a lookup's answer led to.
	maxLookupRounds  = 3
	programCacheSize = 256
	programCacheTTL  = 5 * time.Minute
	customFieldScope = "shipment"
)

var (
	ErrNoProject       = errors.New("edi: the partner has no Lading project")
	ErrNoActiveVersion = errors.New("edi: the partner's Lading project has no active version")
)

// Program is one version of a partner's project, ready to run: its files encoded once with the
// organisation's context, and the manifest its check produced.
type Program struct {
	Project  *edi.LadingProject
	Version  *edi.LadingVersion
	Encoded  *lading.Encoded
	Manifest *lading.Check
}

// Acknowledgment is the acknowledgment Trenova returns for a set the version receives.
func (p *Program) Acknowledgment(set edi.TransactionSet) edi.AcknowledgmentType {
	_, clause := p.Manifest.Receiving(string(set))
	if clause == nil {
		return edi.AcknowledgmentTypeNone
	}
	switch clause.Acknowledge {
	case "999":
		return edi.AcknowledgmentType999
	case lading.AcknowledgeNone:
		return edi.AcknowledgmentTypeNone
	default:
		return edi.AcknowledgmentType997
	}
}

type Programs struct {
	Project *edi.LadingProject
	Active  *Program
	Shadow  *Program
}

type RuntimeParams struct {
	fx.In

	Logger       *zap.Logger
	Projects     ediports.LadingProjectRepository
	Mappings     ediports.EDIMappingProfileRepository
	Orgs         tenantports.OrganizationRepository
	Locations    locationports.LocationRepository
	CustomFields customfieldports.CustomFieldDefinitionRepository
}

type Runtime struct {
	l            *zap.Logger
	client       *lading.Client
	projects     ediports.LadingProjectRepository
	mappings     ediports.EDIMappingProfileRepository
	orgs         tenantports.OrganizationRepository
	locations    locationports.LocationRepository
	customFields customfieldports.CustomFieldDefinitionRepository
	cache        *programCache
}

func NewRuntime(p RuntimeParams) *Runtime { //nolint:gocritic // fx params are passed by value
	return &Runtime{
		l:            p.Logger.Named("edi.lading.runtime"),
		client:       lading.Default(),
		projects:     p.Projects,
		mappings:     p.Mappings,
		orgs:         p.Orgs,
		locations:    p.Locations,
		customFields: p.CustomFields,
		cache:        newProgramCache(programCacheSize, programCacheTTL),
	}
}

func (r *Runtime) Client() *lading.Client {
	return r.client
}

// Load is the partner's active and shadow versions, ready to run.
func (r *Runtime) Load(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
) (*Programs, error) {
	runnable, err := r.projects.GetRunnableVersions(ctx, ediports.GetRunnableLadingVersionRequest{
		TenantInfo: tenantInfo,
		PartnerID:  partnerID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, ErrNoProject
		}
		return nil, err
	}
	programs := &Programs{Project: runnable.Project}
	if runnable.Active != nil {
		if programs.Active, err = r.program(
			ctx,
			tenantInfo,
			runnable.Project,
			runnable.Active,
		); err != nil {
			return nil, err
		}
	}
	if runnable.Shadow != nil && shadowing(runnable.Shadow) {
		if programs.Shadow, err = r.program(
			ctx,
			tenantInfo,
			runnable.Project,
			runnable.Shadow,
		); err != nil {
			return nil, err
		}
	}
	return programs, nil
}

// Active is the partner's active version, or ErrNoActiveVersion.
func (r *Runtime) Active(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
) (*Program, error) {
	programs, err := r.Load(ctx, tenantInfo, partnerID)
	if err != nil {
		return nil, err
	}
	if programs.Active == nil {
		return nil, ErrNoActiveVersion
	}
	return programs.Active, nil
}

// Version is a version of a project encoded for a run, whatever its status: tests, replay and
// the studio's runs use it.
func (r *Runtime) Version(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	project *edi.LadingProject,
	version *edi.LadingVersion,
) (*Program, error) {
	return r.program(ctx, tenantInfo, project, version)
}

func (r *Runtime) program(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	project *edi.LadingProject,
	version *edi.LadingVersion,
) (*Program, error) {
	key := programKey{versionID: version.ID, revision: version.Version}
	if cached := r.cache.get(key); cached != nil {
		return &Program{
			Project:  project,
			Version:  version,
			Encoded:  cached,
			Manifest: &version.Manifest,
		}, nil
	}
	ladingContext, err := r.Context(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	encoded, err := lading.Encode(&lading.Project{Files: version.Files}, ladingContext)
	if err != nil {
		return nil, err
	}
	r.cache.put(key, encoded)
	return &Program{
		Project:  project,
		Version:  version,
		Encoded:  encoded,
		Manifest: &version.Manifest,
	}, nil
}

// Context is what a check or a run is told about the organisation.
func (r *Runtime) Context(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*lading.Context, error) {
	org, err := r.orgs.GetByID(ctx, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, fmt.Errorf("load the organization for Lading: %w", err)
	}
	result := &lading.Context{
		Org: lading.Org{Name: org.Name, SCAC: org.ScacCode, Zone: org.Timezone},
	}
	if r.customFields == nil {
		return result, nil
	}
	definitions, err := r.customFields.GetActiveByResourceType(
		ctx,
		customfieldports.GetActiveByResourceTypeRequest{
			TenantInfo:   tenantInfo,
			ResourceType: customFieldScope,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("load custom fields for Lading: %w", err)
	}
	if len(definitions) == 0 {
		return result, nil
	}
	fields := make([]lading.CustomField, 0, len(definitions))
	for _, definition := range definitions {
		fields = append(fields, lading.CustomField{
			Name:  definition.Name,
			Type:  customFieldType(definition.FieldType),
			Label: definition.Label,
		})
	}
	result.CustomFields = map[string][]lading.CustomField{customFieldScope: fields}
	return result, nil
}

func customFieldType(fieldType customfield.FieldType) string {
	switch fieldType {
	case customfield.FieldTypeNumber:
		return "Decimal"
	case customfield.FieldTypeDate:
		return "Date"
	case customfield.FieldTypeBoolean:
		return "Bool"
	case customfield.FieldTypeMultiSelect:
		return "Texts"
	case customfield.FieldTypeText, customfield.FieldTypeSelect:
	}
	return "Text"
}

type RunRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerID  pulid.ID
	Program    *Program
	Map        string
	Input      lading.RunInput
	Control    lading.Control
	Now        time.Time
	Rules      bool
}

// Run runs a map, answering the lookups it needs from the partner's mappings until it needs no
// more.
func (r *Runtime) Run(ctx context.Context, req *RunRequest) (*lading.RunResult, error) {
	resolved, err := r.Resolve(ctx, req)
	if err != nil {
		return nil, err
	}
	return resolved.Result, nil
}

type Resolved struct {
	Result  *lading.RunResult
	Lookups []lading.LookupAnswer
}

// Resolve is Run with the lookup answers it gave the map, for a second run that must see the
// same answers.
func (r *Runtime) Resolve(ctx context.Context, req *RunRequest) (*Resolved, error) {
	settings, err := Settings(req.Program.Project)
	if err != nil {
		return nil, err
	}
	run := &lading.RunRequest{
		Map:      req.Map,
		Input:    req.Input,
		Settings: settings,
		Now:      instant(req.Now),
		Control:  req.Control,
		Rules:    req.Rules,
	}
	answered := map[lading.Need]bool{}
	for round := 0; ; round++ {
		result, runErr := r.client.Run(ctx, req.Program.Encoded, run)
		if runErr != nil {
			return nil, runErr
		}
		needs := unanswered(result.Needs, answered)
		if len(needs) == 0 || round == maxLookupRounds {
			return &Resolved{Result: result, Lookups: run.Lookups}, nil
		}
		answers, lookupErr := r.Lookup(ctx, req.TenantInfo, req.PartnerID, needs)
		if lookupErr != nil {
			return nil, lookupErr
		}
		for _, need := range needs {
			answered[need] = true
		}
		run.Lookups = append(run.Lookups, answers...)
	}
}

func unanswered(needs []lading.Need, answered map[lading.Need]bool) []lading.Need {
	result := make([]lading.Need, 0, len(needs))
	for _, need := range needs {
		if !answered[need] {
			result = append(result, need)
		}
	}
	return result
}

// Lookup answers lookups from the partner's mappings: a code with no mapping is answered with
// none, which the run reports as unmapped.
func (r *Runtime) Lookup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	needs []lading.Need,
) ([]lading.LookupAnswer, error) {
	if len(needs) == 0 {
		return nil, nil
	}
	types := make([]edi.MappingEntityType, 0, len(needs))
	sources := make([]pulid.ID, 0, len(needs))
	seenType := map[edi.MappingEntityType]bool{}
	for _, need := range needs {
		entityType, ok := LookupEntity(need.Kind)
		if !ok {
			continue
		}
		if !seenType[entityType] {
			seenType[entityType] = true
			types = append(types, entityType)
		}
		sources = append(sources, edi.MappingSourceID(need.Code))
	}
	items, err := r.mappings.GetMappingItems(ctx, ediports.GetMappingItemsRequest{
		PartnerID:   partnerID,
		TenantInfo:  tenantInfo,
		EntityTypes: types,
		SourceIDs:   sources,
	})
	if err != nil {
		return nil, err
	}
	index := make(map[edi.MappingEntityType]map[pulid.ID]*edi.EDIMappingProfileItem, len(types))
	locationIDs := make([]pulid.ID, 0)
	for _, item := range items {
		if index[item.EntityType] == nil {
			index[item.EntityType] = map[pulid.ID]*edi.EDIMappingProfileItem{}
		}
		index[item.EntityType][item.SourceID] = item
		if item.EntityType == edi.MappingEntityTypeLocation {
			locationIDs = append(locationIDs, item.TargetID)
		}
	}
	zones, err := r.locationZones(ctx, tenantInfo, locationIDs)
	if err != nil {
		return nil, err
	}
	answers := make([]lading.LookupAnswer, 0, len(needs))
	for _, need := range needs {
		answer := lading.LookupAnswer{Kind: need.Kind, Code: need.Code}
		entityType, ok := LookupEntity(need.Kind)
		if ok {
			if item := index[entityType][edi.MappingSourceID(need.Code)]; item != nil {
				answer.Found = &lading.Found{
					ID:    item.TargetID.String(),
					Label: item.TargetLabel,
					Zone:  zones[item.TargetID],
				}
			}
		}
		answers = append(answers, answer)
	}
	return answers, nil
}

func (r *Runtime) locationZones(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	zones := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 || r.locations == nil {
		return zones, nil
	}
	locations, err := r.locations.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
		TenantInfo:  tenantInfo,
		LocationIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, location := range locations {
		zones[location.ID] = location.Timezone
	}
	return zones, nil
}

var lookupEntities = map[string]edi.MappingEntityType{
	"Customer":             edi.MappingEntityTypeCustomer,
	"Location":             edi.MappingEntityTypeLocation,
	"Commodity":            edi.MappingEntityTypeCommodity,
	"Accessorial":          edi.MappingEntityTypeAccessorialCharge,
	"ServiceType":          edi.MappingEntityTypeServiceType,
	"ShipmentType":         edi.MappingEntityTypeShipmentType,
	"RatingTemplate":       edi.MappingEntityTypeFormulaTemplate,
	"ServiceFailureReason": edi.MappingEntityTypeServiceFailureReasonCode,
}

// LookupEntity is the mapping entity a lookup kind resolves through.
func LookupEntity(kind string) (edi.MappingEntityType, bool) {
	entity, ok := lookupEntities[kind]
	return entity, ok
}

// Settings are a project's setting values that differ from its partner file's defaults.
func Settings(project *edi.LadingProject) (map[string]jsonutils.RawJSON, error) {
	var values map[string]any
	if project != nil {
		values = project.Settings
	}
	settings := make(map[string]jsonutils.RawJSON, len(values))
	for name, value := range values {
		encoded, err := sonic.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode setting %s: %w", name, err)
		}
		settings[name] = encoded
	}
	return settings, nil
}

func instant(at time.Time) string {
	if at.IsZero() {
		at = time.Now()
	}
	return at.UTC().Format(time.RFC3339)
}

func shadowing(version *edi.LadingVersion) bool {
	return version.Status == edi.LadingVersionStatusShadow &&
		version.ShadowUntil != nil &&
		*version.ShadowUntil > time.Now().Unix()
}

type programKey struct {
	versionID pulid.ID
	revision  int64
}

type programEntry struct {
	encoded *lading.Encoded
	at      time.Time
}

// programCache keeps encoded versions so a run does not re-read the organisation and re-encode
// the files; the module keeps the checked program by the same bytes.
type programCache struct {
	mu      sync.Mutex
	size    int
	ttl     time.Duration
	entries map[programKey]*programEntry
	order   []programKey
}

func newProgramCache(size int, ttl time.Duration) *programCache {
	return &programCache{
		size:    size,
		ttl:     ttl,
		entries: make(map[programKey]*programEntry, size),
		order:   make([]programKey, 0, size),
	}
}

func (c *programCache) get(key programKey) *lading.Encoded {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil
	}
	if time.Since(entry.at) > c.ttl {
		delete(c.entries, key)
		return nil
	}
	return entry.encoded
}

func (c *programCache) put(key programKey, encoded *lading.Encoded) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[key]; !ok {
		c.order = append(c.order, key)
	}
	c.entries[key] = &programEntry{encoded: encoded, at: time.Now()}
	for len(c.order) > c.size {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}
