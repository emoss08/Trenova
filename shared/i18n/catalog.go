package i18n

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sync"

	"github.com/bytedance/sonic"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

var ErrCatalogsLoaded = errors.New(
	"i18n: catalogs are already loaded; register edition catalogs from init",
)

var (
	catalogsOnce   sync.Once
	catalogs       map[Locale]map[string]string
	catalogErr     error
	editionMu      sync.Mutex
	editionSources []fs.FS
	catalogsLoaded bool
)

func RegisterCatalogFS(fsys fs.FS) error {
	if fsys == nil {
		return errors.New("i18n: edition catalog filesystem is nil")
	}

	editionMu.Lock()
	defer editionMu.Unlock()

	if catalogsLoaded {
		return ErrCatalogsLoaded
	}
	editionSources = append(editionSources, fsys)
	return nil
}

func loadCatalogs() {
	editionMu.Lock()
	catalogsLoaded = true
	sources := append([]fs.FS(nil), editionSources...)
	editionMu.Unlock()

	loaded, err := readCatalogs(catalogFS, "catalogs", sources)
	catalogs = loaded
	catalogErr = err
}

func readCatalogs(
	base fs.FS,
	baseDir string,
	editions []fs.FS,
) (map[Locale]map[string]string, error) {
	loaded := make(map[Locale]map[string]string, len(supported))

	for _, locale := range supported {
		messages, err := readCatalog(base, path.Join(baseDir, string(locale)+".json"))
		if err != nil {
			return loaded, err
		}

		for _, edition := range editions {
			extra, editionErr := readCatalog(edition, string(locale)+".json")
			if errors.Is(editionErr, fs.ErrNotExist) {
				continue
			}
			if editionErr != nil {
				return loaded, editionErr
			}
			for key, value := range extra {
				messages[key] = value
			}
		}

		loaded[locale] = messages
	}

	return loaded, nil
}

func readCatalog(fsys fs.FS, name string) (map[string]string, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("i18n: reading %s: %w", name, err)
	}

	messages := make(map[string]string)
	if err = sonic.Unmarshal(raw, &messages); err != nil {
		return nil, fmt.Errorf("i18n: parsing %s: %w", name, err)
	}

	return messages, nil
}

func catalog(locale Locale) map[string]string {
	catalogsOnce.Do(loadCatalogs)
	if catalogErr != nil {
		return nil
	}
	return catalogs[locale]
}

func CatalogError() error {
	catalogsOnce.Do(loadCatalogs)
	return catalogErr
}

func Loaded() map[Locale]int {
	catalogsOnce.Do(loadCatalogs)
	counts := make(map[Locale]int, len(catalogs))
	for locale, messages := range catalogs {
		counts[locale] = len(messages)
	}
	return counts
}
