package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/tableconfiguration"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type ListTableConfigurationResult struct {
	Configurations []*tableconfiguration.Configuration
	Total          int
}

type TableConfigurationRepository interface {
	GetByID(ctx context.Context, id pulid.ID, opts *TableConfigurationFilters) (*tableconfiguration.Configuration, error)
	List(ctx context.Context, filters *TableConfigurationFilters) (*ListTableConfigurationResult, error)
	Create(ctx context.Context, config *tableconfiguration.Configuration) error
	Update(ctx context.Context, config *tableconfiguration.Configuration) error
	Delete(ctx context.Context, id pulid.ID) error
	GetUserConfigurations(ctx context.Context, tableID string, opts *TableConfigurationFilters) ([]*tableconfiguration.Configuration, error)
	ShareConfiguration(ctx context.Context, share *tableconfiguration.ConfigurationShare) error
	RemoveShare(ctx context.Context, configID pulid.ID, sharedWithID pulid.ID) error
}

// TableConfigurationFilters defines filters for querying configurations
type TableConfigurationFilters struct {
	Base            *ports.FilterQueryOptions
	TableIdentifier string
	CreatedBy       pulid.ID
	Visibility      *tableconfiguration.Visibility
	IsDefault       *bool
	Search          string
	UserID          pulid.ID
	// Include relationships
	IncludeShares  bool
	IncludeCreator bool
}
