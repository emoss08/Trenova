package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetTableConfigurationByIDRequest struct {
	ConfigurationID pulid.ID
	TenantInfo      pagination.TenantInfo
}

type ListTableConfigurationsRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	Resource   string                   `json:"resource"`
	Visibility string                   `json:"visibility"`
}

type GetDefaultTableConfigurationRequest struct {
	Resource   string
	TenantInfo pagination.TenantInfo
}

type TableConfigurationRepository interface {
	Create(
		ctx context.Context,
		entity *tableconfiguration.TableConfiguration,
	) (*tableconfiguration.TableConfiguration, error)

	Update(
		ctx context.Context,
		entity *tableconfiguration.TableConfiguration,
	) (*tableconfiguration.TableConfiguration, error)

	GetByID(
		ctx context.Context,
		req GetTableConfigurationByIDRequest,
	) (*tableconfiguration.TableConfiguration, error)

	List(
		ctx context.Context,
		req *ListTableConfigurationsRequest,
	) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error)

	Delete(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error

	GetDefaultForResource(
		ctx context.Context,
		req GetDefaultTableConfigurationRequest,
	) (*tableconfiguration.TableConfiguration, error)

	ClearDefaultForResource(
		ctx context.Context,
		userID pulid.ID,
		resource string,
		tenantInfo pagination.TenantInfo,
	) error
}
