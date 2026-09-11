package i18n

import (
	"embed"
	"fmt"
	"path"
	"sync"

	"github.com/bytedance/sonic"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

var (
	catalogsOnce sync.Once
	catalogs     map[Locale]map[string]string
	catalogErr   error
)

func loadCatalogs() {
	catalogs = make(map[Locale]map[string]string, len(supported))

	for _, locale := range supported {
		name := path.Join("catalogs", string(locale)+".json")

		raw, err := catalogFS.ReadFile(name)
		if err != nil {
			catalogErr = fmt.Errorf("i18n: reading %s: %w", name, err)
			return
		}

		messages := make(map[string]string)
		if err = sonic.Unmarshal(raw, &messages); err != nil {
			catalogErr = fmt.Errorf("i18n: parsing %s: %w", name, err)
			return
		}

		catalogs[locale] = messages
	}
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
