package servicefailureports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListServiceFailureReasonCodeConnectionRequest struct {
	Filter                          *pagination.QueryOptions `json:"filter"`
	Cursor                          pagination.CursorInfo    `json:"-"`
	ServiceFailureReasonCodeColumns []string                 `json:"-"`
}

type GetServiceFailureReasonCodeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

func (r *GetServiceFailureReasonCodeByIDRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Reason code request is required")
		return multiErr
	}
	if r.ID.IsNil() {
		multiErr.Add("id", errortypes.ErrRequired, "Reason code ID is required")
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type GetServiceFailureReasonCodesByIDsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	ReasonCodeIDs []pulid.ID            `json:"reasonCodeIds"`
}

func (r *GetServiceFailureReasonCodesByIDsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	validateTenantInfo(multiErr, r.TenantInfo)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type ServiceFailureReasonCodeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest     `json:"-"`
	AppliesTo          servicefailure.ReasonCodeAppliesTo `json:"appliesTo"`
}

type ServiceFailureReasonCodeRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListServiceFailureReasonCodeConnectionRequest,
	) (*pagination.CursorListResult[*servicefailure.ReasonCode], error)
	GetByID(
		ctx context.Context,
		req GetServiceFailureReasonCodeByIDRequest,
	) (*servicefailure.ReasonCode, error)
	GetByIDs(
		ctx context.Context,
		req GetServiceFailureReasonCodesByIDsRequest,
	) ([]*servicefailure.ReasonCode, error)
	FindDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		appliesTo servicefailure.ReasonCodeAppliesTo,
	) (*servicefailure.ReasonCode, error)
	Create(
		ctx context.Context,
		entity *servicefailure.ReasonCode,
	) (*servicefailure.ReasonCode, error)
	Update(
		ctx context.Context,
		entity *servicefailure.ReasonCode,
	) (*servicefailure.ReasonCode, error)
	Archive(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		actorID pulid.ID,
	) (*servicefailure.ReasonCode, error)
	Activate(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		actorID pulid.ID,
	) (*servicefailure.ReasonCode, error)
	SelectOptions(
		ctx context.Context,
		req *ServiceFailureReasonCodeSelectOptionsRequest,
	) (*pagination.ListResult[*servicefailure.ReasonCode], error)
}

func validateTenantInfo(multiErr *errortypes.MultiError, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if tenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}
}
