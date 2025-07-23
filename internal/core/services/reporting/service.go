// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package reporting

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger *logger.Logger
}

type Service struct {
	l *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "reporting").
		Logger()

	return &Service{
		l: &log,
	}
}

func (s *Service) GetReportTemplate(entity string) (string, error) {
	path, err := s.getReportTemplatePath(entity)
	if err != nil {
		return "", eris.Wrap(err, "get report template path")
	}

	return path, nil
}

func (s *Service) getReportTemplatePath(entity string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", eris.Wrap(err, "get working directory")
	}

	projectRoot, err := fileutils.FindProjectRoot(root)
	if err != nil {
		return "", eris.Wrap(err, "find project root")
	}

	templatesDir := filepath.Join(projectRoot, "web", "report-templates")

	if err = fileutils.EnsureDirExists(templatesDir); err != nil {
		return "", eris.Wrap(err, "ensure templates directory exists")
	}

	return filepath.Join(templatesDir, fmt.Sprintf("%s.csv", entity)), nil
}
