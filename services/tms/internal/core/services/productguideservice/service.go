// Package productguideservice answers where things are in Trenova and how to
// do them, from the generated product guide, for the person asking: a page
// they cannot open is named with what they lack rather than offered as if they
// could.
package productguideservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DefaultLimit and MaxLimit bound how many pages one answer names.
	DefaultLimit = 5
	MaxLimit     = 8

	nameWeight        = 6
	aliasWeight       = 5
	taskTitleWeight   = 4
	breadcrumbWeight  = 3
	taskKeywordWeight = 3
	descriptionWeight = 2
	bodyWeight        = 1

	// cutoffDivisor keeps an answer to the pages that matched about as well
	// as the best one, so "add a rate matrix" does not also list every page
	// with the word "add" in a step.
	cutoffDivisor = 2

	capabilityBrokerage       = "brokerage"
	capabilityAssetOperations = "assetOperations"
)

var Module = fx.Module("product-guide-service", fx.Provide(New))

type Params struct {
	fx.In

	Logger         *zap.Logger
	Permissions    serviceports.PermissionEngine
	Organizations  repositories.OrganizationRepository
	Vectorizer     serviceports.QueryVectorizer    `optional:"true"`
	CatalogVectors serviceports.CatalogVectorIndex `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	catalog       *productguide.Catalog
	permissions   serviceports.PermissionEngine
	organizations repositories.OrganizationRepository
	index         []indexedPage
	resources     *permission.Registry
	vectorizer    serviceports.QueryVectorizer
	vectors       serviceports.CatalogVectorIndex
	items         []serviceports.EmbeddingCatalogItem
	floor         float64
}

type indexedPage struct {
	page        *productguide.Page
	name        map[string]struct{}
	aliases     map[string]struct{}
	breadcrumb  map[string]struct{}
	description map[string]struct{}
	body        map[string]struct{}
	tasks       []indexedTask
}

type indexedTask struct {
	task     *productguide.Task
	title    map[string]struct{}
	keywords map[string]struct{}
	steps    map[string]struct{}
}

func New(p Params) serviceports.ProductGuide {
	return newService(p, productguide.Default)
}

func newService(p Params, catalog *productguide.Catalog) *Service {
	return &Service{
		l:             p.Logger.Named("service.productguide"),
		catalog:       catalog,
		permissions:   p.Permissions,
		organizations: p.Organizations,
		index:         buildIndex(catalog),
		resources:     permission.NewRegistry(),
		vectorizer:    p.Vectorizer,
		vectors:       p.CatalogVectors,
		items:         CatalogItems(catalog),
		floor:         serviceports.DefaultCatalogSimilarityFloor,
	}
}

func buildIndex(catalog *productguide.Catalog) []indexedPage {
	index := make([]indexedPage, 0, len(catalog.Pages))
	for i := range catalog.Pages {
		page := &catalog.Pages[i]
		entry := indexedPage{
			page:        page,
			name:        agentsearch.TokenSet(page.Name),
			aliases:     agentsearch.TokenSet(strings.Join(page.Aliases, " ")),
			breadcrumb:  agentsearch.TokenSet(strings.Join(page.Breadcrumb, " ")),
			description: agentsearch.TokenSet(page.Description),
			body:        agentsearch.TokenSet(page.Summary + " " + page.Notes),
			tasks:       make([]indexedTask, 0, len(page.Tasks)),
		}
		for j := range page.Tasks {
			task := &page.Tasks[j]
			entry.tasks = append(entry.tasks, indexedTask{
				task:     task,
				title:    agentsearch.TokenSet(task.Title),
				keywords: agentsearch.TokenSet(strings.Join(task.Keywords, " ")),
				steps:    agentsearch.TokenSet(strings.Join(task.Steps, " ")),
			})
		}
		index = append(index, entry)
	}

	return index
}

func (s *Service) PageForPath(path string) (*productguide.Page, bool) {
	return s.catalog.PageForPath(path)
}

type scored struct {
	entry *indexedPage
	score int
	task  *productguide.Task
}

func (s *Service) Search(
	ctx context.Context,
	req *serviceports.ProductGuideSearchRequest,
) ([]serviceports.ProductGuideMatch, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)

	onlyPage := strings.TrimSpace(req.Page)
	var similarity map[string]float64
	if onlyPage == "" {
		similarity = s.similarities(ctx, req)
	}
	candidates := s.ranked(req.Query, onlyPage, similarity, limit)
	if len(candidates) == 0 {
		return []serviceports.ProductGuideMatch{}, nil
	}

	pages := make([]*productguide.Page, 0, len(candidates))
	for _, candidate := range candidates {
		pages = append(pages, candidate.entry.page)
	}
	access, err := s.access(ctx, req.Actor, req.TenantInfo, pages)
	if err != nil {
		return nil, err
	}

	matches := make([]serviceports.ProductGuideMatch, 0, len(candidates))
	for _, candidate := range candidates {
		canOpen, missing := access.check(candidate.entry.page)
		matches = append(matches, serviceports.ProductGuideMatch{
			Page:    candidate.entry.page,
			Task:    candidate.task,
			CanOpen: canOpen,
			Missing: missing,
		})
	}

	return matches, nil
}

// rank scores every page against the question and keeps those close to the
// best. Asked about one page, it answers from that page alone, with its best
// task — or its first when the question names none.
func (s *Service) rank(terms map[string]struct{}, onlyPage string) []scored {
	candidates := make([]scored, 0, len(s.index))
	for i := range s.index {
		entry := &s.index[i]
		if onlyPage != "" {
			if entry.page.Path != onlyPage {
				continue
			}
			task, _ := bestTask(entry, terms)
			if task == nil && len(entry.page.Tasks) > 0 {
				task = &entry.page.Tasks[0]
			}
			return []scored{{entry: entry, score: 1, task: task}}
		}

		pageScore := scoreTerms(terms, []weighted{
			{entry.name, nameWeight},
			{entry.aliases, aliasWeight},
			{entry.breadcrumb, breadcrumbWeight},
			{entry.description, descriptionWeight},
			{entry.body, bodyWeight},
		})
		task, taskScore := bestTask(entry, terms)
		total := pageScore + taskScore
		if total == 0 {
			continue
		}
		candidates = append(candidates, scored{entry: entry, score: total, task: task})
	}

	sort.SliceStable(candidates, func(a, b int) bool {
		if candidates[a].score != candidates[b].score {
			return candidates[a].score > candidates[b].score
		}
		return catalogOrder(candidates[a].entry, candidates[b].entry)
	})
	if len(candidates) == 0 {
		return candidates
	}

	cutoff := candidates[0].score / cutoffDivisor
	kept := candidates[:0]
	for _, candidate := range candidates {
		if candidate.score >= cutoff {
			kept = append(kept, candidate)
		}
	}

	return kept
}

type weighted struct {
	tokens map[string]struct{}
	weight int
}

// scoreTerms counts each term once, at the heaviest field it appears in.
func scoreTerms(terms map[string]struct{}, fields []weighted) int {
	total := 0
	for term := range terms {
		for _, field := range fields {
			if _, ok := field.tokens[term]; ok {
				total += field.weight
				break
			}
		}
	}

	return total
}

func bestTask(entry *indexedPage, terms map[string]struct{}) (*productguide.Task, int) {
	var best *productguide.Task
	bestScore := 0
	for i := range entry.tasks {
		task := &entry.tasks[i]
		score := scoreTerms(terms, []weighted{
			{task.title, taskTitleWeight},
			{task.keywords, taskKeywordWeight},
			{task.steps, bodyWeight},
		})
		if score > bestScore {
			best, bestScore = task.task, score
		}
	}

	return best, bestScore
}

func (s *Service) Destination(
	ctx context.Context,
	req *serviceports.ProductGuideDestinationRequest,
) (*serviceports.ProductGuideDestination, error) {
	entity := strings.TrimSpace(req.Entity)
	pagePath := strings.TrimSpace(req.Page)

	var (
		path  string
		label string
	)
	switch {
	case entity != "":
		if strings.TrimSpace(req.RecordID) == "" {
			return nil, errortypes.NewValidationError(
				"recordId", errortypes.ErrRequired,
				"Give the record's id with its kind, from the tool that found it.",
			)
		}
		link, ok := s.catalog.Record(entity)
		if !ok {
			return nil, errortypes.NewValidationError(
				"entity", errortypes.ErrInvalid,
				fmt.Sprintf(
					"%q records do not open on a page of their own. Records that do: %s.",
					entity, strings.Join(s.recordEntities(), ", "),
				),
			)
		}
		path, _ = s.catalog.RecordPath(entity, req.RecordID, nil)
		label = link.Label
	case pagePath != "":
		page, ok := s.catalog.Page(pagePath)
		if !ok {
			return nil, errortypes.NewValidationError(
				"page", errortypes.ErrInvalid,
				fmt.Sprintf(
					"%q is not a page in Trenova. Find the page with find_in_trenova and pass its path.",
					pagePath,
				),
			)
		}
		path = page.Path
		label = page.Name
		if req.Create {
			if page.CreateAction == nil {
				return nil, errortypes.NewValidationError(
					"action", errortypes.ErrInvalid,
					fmt.Sprintf("%s has no create form to open; open the page instead.", page.Name),
				)
			}
			path = withQuery(page.Path, page.CreateAction.Query)
		}
	default:
		return nil, errortypes.NewValidationError(
			"page", errortypes.ErrRequired,
			"Say where to go: a page's path, or a record's kind and id.",
		)
	}

	page, ok := s.catalog.PageForPath(path)
	if !ok {
		return nil, fmt.Errorf("%s opens on %s, which the guide does not list", label, path)
	}

	access, err := s.access(ctx, req.Actor, req.TenantInfo, []*productguide.Page{page})
	if err != nil {
		return nil, err
	}
	if canOpen, missing := access.check(page); !canOpen {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"%s is not open to this person: it needs %s. Tell them who can grant it rather than sending them there.",
			page.Name, strings.Join(missing, " and "),
		))
	}

	return &serviceports.ProductGuideDestination{Path: path, Page: page, Label: label}, nil
}

func (s *Service) recordEntities() []string {
	entities := make([]string, 0, len(s.catalog.Records))
	for _, record := range s.catalog.Records {
		entities = append(entities, record.Entity)
	}

	return entities
}

func withQuery(path string, query map[string]string) string {
	if len(query) == 0 {
		return path
	}
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(path)
	for i, key := range keys {
		if i == 0 {
			b.WriteByte('?')
		} else {
			b.WriteByte('&')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(query[key])
	}

	return b.String()
}

// access is what one person may open, checked once per answer: every
// distinct permission the answer's pages demand, in one batch, and the
// organization's capabilities when one of them depends on one.
type access struct {
	allowed      map[string]bool
	capabilities repositories.OrganizationCapabilities
	resources    *permission.Registry
}

func (s *Service) access(
	ctx context.Context,
	actor *serviceports.RequestActor,
	tenant pagination.TenantInfo,
	pages []*productguide.Page,
) (*access, error) {
	if actor == nil {
		return nil, errors.New("the product guide answers for a person, and none was given")
	}

	requirements := make([]productguide.Requirement, 0, len(pages))
	seen := make(map[string]struct{}, len(pages))
	gated := false
	for _, page := range pages {
		gated = gated || len(page.Capabilities) > 0
		for _, requirement := range page.Requires {
			key := requirementKey(requirement)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			requirements = append(requirements, requirement)
		}
	}

	allowed, err := s.allowed(ctx, actor, requirements)
	if err != nil {
		return nil, err
	}

	result := &access{allowed: allowed, resources: s.resources}
	if !gated {
		return result, nil
	}
	capabilities, err := s.organizations.GetCapabilities(ctx, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("read what this organization operates: %w", err)
	}
	result.capabilities = *capabilities

	return result, nil
}

// allowed checks every requirement in one batch.
func (s *Service) allowed(
	ctx context.Context,
	actor *serviceports.RequestActor,
	requirements []productguide.Requirement,
) (map[string]bool, error) {
	allowed := make(map[string]bool, len(requirements))
	if len(requirements) == 0 {
		return allowed, nil
	}

	checks := make([]serviceports.ResourceOperationCheck, 0, len(requirements))
	for _, requirement := range requirements {
		checks = append(checks, serviceports.ResourceOperationCheck{
			Resource:  requirement.Resource,
			Operation: permission.Operation(requirement.Operation),
		})
	}
	result, err := s.permissions.CheckBatch(ctx, &serviceports.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Checks:         checks,
	})
	if err != nil {
		return nil, fmt.Errorf("check which pages this person may open: %w", err)
	}
	for i, requirement := range requirements {
		allowed[requirementKey(requirement)] = i < len(result.Results) && result.Results[i].Allowed
	}

	return allowed, nil
}

func requirementKey(requirement productguide.Requirement) string {
	return requirement.Resource + ":" + requirement.Operation
}

// check is whether the person may open a page and, when not, what they lack
// in words: the permission's display name, or the part of the product the
// organization has switched off.
func (a *access) check(page *productguide.Page) (bool, []string) {
	var missing []string
	for _, capability := range page.Capabilities {
		switch capability {
		case capabilityBrokerage:
			if !a.capabilities.BrokerageEnabled {
				missing = append(missing, "brokerage to be turned on for the organization")
			}
		case capabilityAssetOperations:
			if !a.capabilities.AssetOperationsEnabled {
				missing = append(missing, "asset operations to be turned on for the organization")
			}
		}
	}
	for _, requirement := range page.Requires {
		if a.allowed[requirementKey(requirement)] {
			continue
		}
		name := requirement.Resource
		if definition, ok := a.resources.Get(requirement.Resource); ok && definition.DisplayName != "" {
			name = definition.DisplayName
		}
		missing = append(missing, fmt.Sprintf("%s %s permission", name, requirement.Operation))
	}

	return len(missing) == 0, missing
}
