package productguideservice

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/rankfusion"
	"go.uber.org/zap"
)

const (
	vectorCandidateFactor = 4
	minVectorCandidates   = 24
)

func CatalogItems(catalog *productguide.Catalog) []serviceports.EmbeddingCatalogItem {
	items := make([]serviceports.EmbeddingCatalogItem, 0, len(catalog.Pages))
	for i := range catalog.Pages {
		page := &catalog.Pages[i]
		items = append(items, serviceports.NewEmbeddingCatalogItem(page.Path, PageText(page)))
	}

	return items
}

func PageText(page *productguide.Page) string {
	var b strings.Builder
	b.WriteString(page.Name)
	if location := page.Location(); location != "" {
		b.WriteString(" (")
		b.WriteString(location)
		b.WriteString(")")
	}
	writeLine(&b, page.Description)
	writeLine(&b, page.Summary)
	if len(page.Aliases) > 0 {
		writeLine(&b, "Also called: "+strings.Join(page.Aliases, ", "))
	}
	if len(page.Tasks) > 0 {
		titles := make([]string, 0, len(page.Tasks))
		for i := range page.Tasks {
			if title := strings.TrimSpace(page.Tasks[i].Title); title != "" {
				titles = append(titles, title)
			}
		}
		if len(titles) > 0 {
			writeLine(&b, "People do here: "+strings.Join(titles, "; "))
		}
	}

	return b.String()
}

func writeLine(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	b.WriteByte('\n')
	b.WriteString(text)
}

func NewOffline(catalog *productguide.Catalog) *Service {
	return &Service{
		l:         zap.NewNop(),
		catalog:   catalog,
		index:     buildIndex(catalog),
		resources: permission.NewRegistry(),
		items:     CatalogItems(catalog),
		floor:     serviceports.DefaultCatalogSimilarityFloor,
	}
}

func (s *Service) RankPaths(query string, similarity map[string]float64, limit int) []string {
	candidates := s.ranked(query, "", similarity, limit)
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.entry.page.Path)
	}

	return out
}

func (s *Service) ranked(
	query string,
	onlyPage string,
	similarity map[string]float64,
	limit int,
) []scored {
	if limit <= 0 {
		return nil
	}

	terms := agentsearch.Terms(query)
	candidates := s.rank(terms, onlyPage)
	if onlyPage == "" {
		candidates = s.fuse(candidates, terms, similarity, limit)
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates
}

func (s *Service) similarities(
	ctx context.Context,
	req *serviceports.ProductGuideSearchRequest,
) map[string]float64 {
	if s.vectorizer == nil || s.vectors == nil || len(s.items) == 0 ||
		strings.TrimSpace(req.Query) == "" ||
		req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil
	}

	query, err := s.vectorizer.Vectorize(ctx, &serviceports.QueryVectorRequest{
		TenantInfo:  req.TenantInfo,
		Text:        req.Query,
		Attribution: req.Attribution,
	})
	if err != nil {
		s.l.Warn("could not embed a product guide question; searching by keyword",
			zap.String("organization", req.TenantInfo.OrgID.String()), zap.Error(err))
		return nil
	}
	if !query.Usable() {
		return nil
	}

	found, err := s.vectors.Similarities(ctx, &serviceports.CatalogSimilarityRequest{
		TenantInfo: req.TenantInfo,
		Corpus:     airetrieval.CatalogCorpusProductGuide,
		Items:      s.items,
		Query:      query,
	})
	if err != nil {
		s.l.Warn("could not compare the product guide with a question; searching by keyword",
			zap.String("organization", req.TenantInfo.OrgID.String()), zap.Error(err))
		return nil
	}
	if !found.Available {
		return nil
	}

	return found.ByKey
}

func (s *Service) fuse(
	keyword []scored,
	terms map[string]struct{},
	similarity map[string]float64,
	limit int,
) []scored {
	if len(similarity) == 0 {
		return keyword
	}

	byPath := make(map[string]scored, len(keyword))
	keywordPaths := make([]string, 0, len(keyword))
	for _, candidate := range keyword {
		byPath[candidate.entry.page.Path] = candidate
		keywordPaths = append(keywordPaths, candidate.entry.page.Path)
	}

	vector := s.vectorRanked(similarity, limit)
	vectorPaths := make([]string, 0, len(vector))
	entries := make(map[string]*indexedPage, len(vector))
	for _, entry := range vector {
		vectorPaths = append(vectorPaths, entry.page.Path)
		entries[entry.page.Path] = entry
	}

	scores := rankfusion.Reciprocal(rankfusion.DefaultK, keywordPaths, vectorPaths)
	fused := make([]scored, 0, len(scores))
	for path := range scores {
		if candidate, ok := byPath[path]; ok {
			fused = append(fused, candidate)
			continue
		}
		if similarity[path] < s.floor {
			continue
		}
		entry := entries[path]
		task, _ := bestTask(entry, terms)
		fused = append(fused, scored{entry: entry, task: task})
	}

	sort.Slice(fused, func(a, b int) bool {
		left, right := scores[fused[a].entry.page.Path], scores[fused[b].entry.page.Path]
		if left != right {
			return left > right
		}
		return catalogOrder(fused[a].entry, fused[b].entry)
	})

	return fused
}

func (s *Service) vectorRanked(similarity map[string]float64, limit int) []*indexedPage {
	ranked := make([]*indexedPage, 0, len(similarity))
	for i := range s.index {
		entry := &s.index[i]
		if _, ok := similarity[entry.page.Path]; ok {
			ranked = append(ranked, entry)
		}
	}

	sort.Slice(ranked, func(a, b int) bool {
		left, right := similarity[ranked[a].page.Path], similarity[ranked[b].page.Path]
		if left != right {
			return left > right
		}
		return catalogOrder(ranked[a], ranked[b])
	})

	candidates := max(limit*vectorCandidateFactor, minVectorCandidates)
	if len(ranked) > candidates {
		ranked = ranked[:candidates]
	}

	return ranked
}

func catalogOrder(a, b *indexedPage) bool {
	// A page in the navigation is the one people are meant to use.
	if a.page.InNavigation != b.page.InNavigation {
		return a.page.InNavigation
	}

	return a.page.Path < b.page.Path
}
