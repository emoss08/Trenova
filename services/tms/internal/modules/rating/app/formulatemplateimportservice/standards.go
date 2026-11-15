package formulatemplateimportservice

import "github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"

type StandardTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

func (s *Service) ListStandards() ([]StandardTemplate, error) {
	catalog, err := standardcatalog.Load()
	if err != nil {
		return nil, err
	}

	standards := make([]StandardTemplate, 0, len(catalog))
	for _, entry := range catalog {
		standards = append(standards, StandardTemplate{
			Name:        entry.Name,
			Description: entry.Description,
			Source:      entry.Source,
		})
	}
	return standards, nil
}
