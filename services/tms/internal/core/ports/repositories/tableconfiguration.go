package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type TableConfigurationFilters struct {
	Filter         *pagination.QueryOptions       `json:"filter"         form:"filter"`
	Visibility     *tableconfiguration.Visibility `json:"visibility"     form:"visibility"`
	Resource       string                         `json:"resource"       form:"resource"`
	Search         string                         `json:"search"         form:"search"`
	CreatedBy      pulid.ID                       `json:"createdBy"      form:"createdBy"`
	UserID         pulid.ID                       `json:"userId"         form:"userId"`
	IncludeShares  bool                           `json:"includeShares"  form:"includeShares"`
	IncludeCreator bool                           `json:"includeCreator" form:"includeCreator"`
	IsDefault      bool                           `json:"isDefault"      form:"isDefault"`
}

type CopyTableConfigurationRequest struct {
	ConfigID pulid.ID `json:"configId" form:"configId"`
	UserID   pulid.ID `json:"userId"   form:"userId"`
	OrgID    pulid.ID `json:"orgId"    form:"orgId"`
	BuID     pulid.ID `json:"buId"     form:"buId"`
}

type ListUserConfigurationRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"   form:"filter"`
	Resource string                   `json:"resource" form:"resource"`
}

type DeleteUserConfigurationRequest struct {
	ConfigID pulid.ID `json:"configId" form:"configId"`
	UserID   pulid.ID `json:"userId"   form:"userId"`
	OrgID    pulid.ID `json:"orgId"    form:"orgId"`
	BuID     pulid.ID `json:"buId"     form:"buId"`
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
	) (*pagination.ListResult[*tableconfiguration.Configuration], error)
	ListPublicConfigurations(
		ctx context.Context,
		opts *TableConfigurationFilters,
	) (*pagination.ListResult[*tableconfiguration.Configuration], error)
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
	) (*pagination.ListResult[*tableconfiguration.Configuration], error)
	GetDefaultOrLatest(
		ctx context.Context,
		resource string,
		opts *TableConfigurationFilters,
	) (*tableconfiguration.Configuration, error)
	Copy(ctx context.Context, req *CopyTableConfigurationRequest) error
	Share(ctx context.Context, share *tableconfiguration.ConfigurationShare) error
	RemoveShare(ctx context.Context, configID pulid.ID, sharedWithID pulid.ID) error
}
