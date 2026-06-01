package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListServiceFailureReasonCodesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetServiceFailureReasonCodeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ServiceFailureReasonCodeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest     `json:"-"`
	AppliesTo          servicefailure.ReasonCodeAppliesTo `json:"appliesTo"`
}

type ReorderServiceFailureReasonCodesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ReasonIDs  []pulid.ID            `json:"reasonIds"`
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

func (r *ReorderServiceFailureReasonCodesRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Reason code reorder request is required")
		return multiErr
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if len(r.ReasonIDs) == 0 {
		multiErr.Add("reasonIds", errortypes.ErrRequired, "Reason code IDs are required")
	}
	seen := make(map[pulid.ID]struct{}, len(r.ReasonIDs))
	for _, id := range r.ReasonIDs {
		if id.IsNil() {
			multiErr.Add("reasonIds", errortypes.ErrInvalid, "Reason code IDs cannot be empty")
			continue
		}
		if _, ok := seen[id]; ok {
			multiErr.Add("reasonIds", errortypes.ErrInvalid, "Reason code IDs must be unique")
			continue
		}
		seen[id] = struct{}{}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type ServiceFailureReasonCodeRepository interface {
	List(
		ctx context.Context,
		req *ListServiceFailureReasonCodesRequest,
	) (*pagination.ListResult[*servicefailure.ReasonCode], error)
	GetByID(
		ctx context.Context,
		req GetServiceFailureReasonCodeByIDRequest,
	) (*servicefailure.ReasonCode, error)
	FindDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		appliesTo servicefailure.ReasonCodeAppliesTo,
	) (*servicefailure.ReasonCode, error)
	Create(ctx context.Context, entity *servicefailure.ReasonCode) (*servicefailure.ReasonCode, error)
	Update(ctx context.Context, entity *servicefailure.ReasonCode) (*servicefailure.ReasonCode, error)
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
	Reorder(
		ctx context.Context,
		req *ReorderServiceFailureReasonCodesRequest,
	) ([]*servicefailure.ReasonCode, error)
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
