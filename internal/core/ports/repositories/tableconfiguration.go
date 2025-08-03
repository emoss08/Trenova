/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// TableConfigurationFilters defines filters for querying configurations
type TableConfigurationFilters struct {
	Base       *ports.FilterQueryOptions
	Resource   string
	CreatedBy  pulid.ID
	Visibility *tableconfiguration.Visibility
	IsDefault  *bool
	Search     string
	UserID     pulid.ID
	// Include relationships
	IncludeShares  bool
	IncludeCreator bool
}

// CopyTableConfigurationRequest defines a request for copying a table configuration
type CopyTableConfigurationRequest struct {
	ConfigID pulid.ID
	UserID   pulid.ID
	OrgID    pulid.ID
	BuID     pulid.ID
}

// ListUserConfigurationRequest defines a request for listing user configurations
type ListUserConfigurationRequest struct {
	Filter   *ports.LimitOffsetQueryOptions `query:"filter"`
	Resource string
}

type DeleteUserConfigurationRequest struct {
	ConfigID pulid.ID `json:"configId"`
	UserID   pulid.ID `json:"userId"`
	OrgID    pulid.ID `json:"orgId"`
	BuID     pulid.ID `json:"buId"`
}

type TableConfigurationRepository interface {
	GetByID(
		ctx context.Context,
		id pulid.ID,
		opts *TableConfigurationFilters,
	) (*tableconfiguration.Configuration, error)
	List(
		ctx context.Context,
		filters *TableConfigurationFilters,
	) (*ports.ListResult[*tableconfiguration.Configuration], error)
	ListPublicConfigurations(
		ctx context.Context,
		opts *TableConfigurationFilters,
	) (*ports.ListResult[*tableconfiguration.Configuration], error)
	Create(
		ctx context.Context,
		config *tableconfiguration.Configuration,
	) (*tableconfiguration.Configuration, error)
	Update(ctx context.Context, config *tableconfiguration.Configuration) error
	Delete(ctx context.Context, req DeleteUserConfigurationRequest) error
	GetUserConfigurations(
		ctx context.Context,
		tableID string,
		opts *TableConfigurationFilters,
	) ([]*tableconfiguration.Configuration, error)
	ListUserConfigurations(
		ctx context.Context,
		opts *ListUserConfigurationRequest,
	) (*ports.ListResult[*tableconfiguration.Configuration], error)
	GetDefaultOrLatestConfiguration(
		ctx context.Context,
		resource string,
		opts *TableConfigurationFilters,
	) (*tableconfiguration.Configuration, error)
	Copy(ctx context.Context, req *CopyTableConfigurationRequest) error
	ShareConfiguration(ctx context.Context, share *tableconfiguration.ConfigurationShare) error
	RemoveShare(ctx context.Context, configID pulid.ID, sharedWithID pulid.ID) error
}
