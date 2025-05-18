import { SelectOption } from "./fields";

export type PaginationResponse<TData> = {
  data: TData[];
  meta: {
    totalCount: number;
    hasMore: boolean;
  };
  cursors: {
    next?: string;
    previous?: string;
  };
};

type TenantOptions = {
  orgId: string;
  buId: string;
  userId: string;
};

export type LimitOffsetOptions = {
  tenantOpts: TenantOptions;
  limit: number;
  offset: number;
};

export type LimitOffsetResponse<TData> = {
  results: TData[];
  count: number;
  next: string;
  prev: string;
};

export type SelectOptionResponse = {
  results: SelectOption[];
};

export type BaseEndpoint =
  | "/auth/change-password/"
  | "/auth/reset_password/"
  | "/auth/change-email/"
  | "/auth/login"
  | "/auth/logout"
  | "/shipment-types/"
  | "/me/favorites/"
  | "/users/"
  | "/users/me/"
  | "/job_titles/"
  | "/groups/"
  | "/permissions/"
  | "/general-ledger-accounts/"
  | "/revenue-codes/"
  | "/division-codes/"
  | "/organizations/"
  | "/depots/"
  | "/departments/"
  | "/email-control/"
  | "/email-profiles/"
  | "/email-log/"
  | "/tax-rates/"
  | "/table-change-alerts/"
  | "/notification-types/"
  | "/notification-settings/"
  | "/workers/"
  | "/billing_control/"
  | "/billing_queue/"
  | "/billing_history/"
  | "/charge-types/"
  | "/accessorial-charges/"
  | "/document-classifications/"
  | "/hazardous-materials/"
  | "/commodities/"
  | "/customers/"
  | "/equipment-types/"
  | "/tractors/"
  | "/trailers/"
  | "/equipment-manufacturers/"
  | "/equipment-maintenance_plans/"
  | "/location-categories/"
  | "/locations/"
  | "/location-contacts/"
  | "/location-comments/"
  | "/comment-types/"
  | "/delay-codes/"
  | "/fleet-codes/"
  | "/dispatch-control/"
  | "/rates/"
  | "/feasibility-tool-control/"
  | "/integration-vendors/"
  | "/integrations/"
  | "/google-api/"
  | "/routes/"
  | "/route-control/"
  | "/qualifier-codes/"
  | "/stop-comments/"
  | "/service-incidents/"
  | "/stops/"
  | "/shipment-control/"
  | "/reason-codes/"
  | "/shipments/"
  | "/service-types/"
  | "/additional-charges/"
  | "/invoice-control/"
  | "/custom-reports/"
  | "/user-reports/"
  | "/log-entries/"
  | "/bill-invoice/"
  | "/mass_bill_shipments/"
  | "/table-columns/"
  | "/transfer-to-billing/"
  | "/untransfer-invoice/"
  | "reports/column-names"
  | "/generate-report/"
  | "/users/notifications/"
  | "/billing/shipments-ready/"
  | "/tags/"
  | "/hazmat-segregation-rules/"
  | "/formula-templates/"
  | "/finance-transactions/"
  | "/reconciliation-queue/"
  | "/organizations/me/"
  | "/roles/"
  | "/us-states/"
  | "/analytics/new-shipment-count/"
  | "/audit-logs/"
  | "/accounting-control/"
  | "/table-configurations/"
  | "/document-types/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;
