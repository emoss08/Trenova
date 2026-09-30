package edihandler

import (
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func tenantInfoFromAuth(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func queryOptionsFromSelect(req *pagination.SelectQueryRequest) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: req.TenantInfo,
		Pagination: req.Pagination,
		Query:      req.Query,
	}
}
