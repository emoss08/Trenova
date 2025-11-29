/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { SelectOption } from "./fields";

export type ListResult<TData> = {
  items: TData[];
  total: number;
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
  | "/audit-entries/"
  | "/accounting-control/"
  | "/table-configurations/"
  | "/dedicated-lanes/"
  | "/consolidations/"
  | "/document-types/"
  | "/hold-reasons/"
  | "/ai-logs/"
  | "/workers/pto/"
  | "/variables/"
  | "/distance-overrides/"
  | "/fiscal-years/"
  | "/fiscal-periods/"
  | "/gl-accounts/"
  | "/variable-formats/"
  | "/document-templates/"
  | "/account-types/"
  | "/workers/pto/create/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;

export type SELECT_OPTIONS_ENDPOINTS =
  | API_ENDPOINTS
  | `${API_ENDPOINTS}select-options/`;
