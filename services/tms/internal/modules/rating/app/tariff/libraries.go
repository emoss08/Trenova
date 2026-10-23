package tariff

import (
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
)

const (
	MaxLibraries        = 32
	maxResolutionRounds = 8
)

var useOrgPattern = regexp.MustCompile(`(?m)^[ \t]*use[ \t]+org\.([A-Za-z_][A-Za-z0-9_]*)`)

func ImportedLibraryNames(sources ...string) []string {
	names := make([]string, 0, 4)
	for _, source := range sources {
		for _, match := range useOrgPattern.FindAllStringSubmatch(source, -1) {
			name := match[1]
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names
}

type LibrarySet []*formulalibrary.FormulaLibraryVersion

func (s LibrarySet) Context() []rules.TariffLibrary {
	out := make([]rules.TariffLibrary, 0, len(s))
	for _, version := range s {
		out = append(out, rules.TariffLibrary{
			Path:    version.Path(),
			Version: version.ID.String(),
			Source:  version.Source,
		})
	}
	return out
}

func (s LibrarySet) byPath() map[string]*formulalibrary.FormulaLibraryVersion {
	out := make(map[string]*formulalibrary.FormulaLibraryVersion, len(s))
	for _, version := range s {
		out[version.Path()] = version
	}
	return out
}

func (s LibrarySet) Pins(imports []rules.TariffImport) []tarifftypes.Import {
	byPath := s.byPath()
	pinned := make(map[string]struct{}, len(imports))
	pending := make([]string, 0, len(imports))
	for _, imp := range imports {
		if strings.HasPrefix(imp.Path, formulalibrary.PathPrefix) {
			pending = append(pending, imp.Path)
		}
	}

	for len(pending) > 0 {
		path := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, seen := pinned[path]; seen {
			continue
		}
		version, ok := byPath[path]
		if !ok {
			continue
		}
		pinned[path] = struct{}{}
		for _, name := range ImportedLibraryNames(version.Source) {
			pending = append(pending, formulalibrary.Path(name))
		}
	}

	out := make([]tarifftypes.Import, 0, len(pinned))
	for _, version := range s {
		if _, ok := pinned[version.Path()]; ok {
			out = append(out, version.Pin())
		}
	}
	slices.SortFunc(out, func(a, b tarifftypes.Import) int {
		return strings.Compare(a.Path, b.Path)
	})
	return out
}

type libraryVersions interface {
	LatestApprovedByNames(
		ctx context.Context,
		req *formulalibraryports.LatestApprovedByNamesRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	GetByPins(
		ctx context.Context,
		req *formulalibraryports.GetVersionsByPinsRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
}

type libraryLoader struct {
	versions libraryVersions
}

func (l libraryLoader) latest(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sources ...string,
) (LibrarySet, error) {
	looked := make(map[string]struct{}, 4)
	pending := ImportedLibraryNames(sources...)
	set := make(LibrarySet, 0, len(pending))

	for round := 0; len(pending) > 0 && round < maxResolutionRounds; round++ {
		batch := make([]string, 0, len(pending))
		for _, name := range pending {
			key := strings.ToLower(name)
			if _, done := looked[key]; done {
				continue
			}
			if len(looked) >= MaxLibraries {
				break
			}
			looked[key] = struct{}{}
			batch = append(batch, name)
		}
		if len(batch) == 0 {
			break
		}

		found, err := l.versions.LatestApprovedByNames(
			ctx,
			&formulalibraryports.LatestApprovedByNamesRequest{TenantInfo: tenant, Names: batch},
		)
		if err != nil {
			return nil, err
		}

		next := make([]string, 0, len(found))
		for _, version := range found {
			looked[strings.ToLower(version.Name)] = struct{}{}
			set = append(set, version)
			next = append(next, ImportedLibraryNames(version.Source)...)
		}
		pending = next
	}

	slices.SortFunc(set, func(a, b *formulalibrary.FormulaLibraryVersion) int {
		return strings.Compare(a.Name, b.Name)
	})
	return set, nil
}

func (l libraryLoader) pinned(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pins []tarifftypes.Import,
) (LibrarySet, error) {
	usable := make([]tarifftypes.Import, 0, len(pins))
	for _, pin := range pins {
		if pin.LibraryID.IsNotNil() && pin.VersionNumber > 0 {
			usable = append(usable, pin)
		}
	}
	if len(usable) == 0 {
		return LibrarySet{}, nil
	}

	versions, err := effectiveversioncache.GetLibraries(
		ctx,
		pinKey(usable),
		func(loadCtx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			return l.versions.GetByPins(loadCtx, &formulalibraryports.GetVersionsByPinsRequest{
				TenantInfo: tenant,
				Pins:       usable,
			})
		},
	)
	if err != nil {
		return nil, err
	}
	return LibrarySet(versions), nil
}

func pinKey(pins []tarifftypes.Import) string {
	parts := make([]string, 0, len(pins))
	for _, pin := range pins {
		parts = append(parts, pin.LibraryID.String()+"@"+strconv.FormatInt(pin.VersionNumber, 10))
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}
