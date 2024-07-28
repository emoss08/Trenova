/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

export type ApiResponse<K> = {
  count: number;
  next: string | null;
  previous: string | null;
  results: K[];
};

type APIAttrs =
  | "nonFieldErrors"
  | "validationError"
  | "All"
  | "databaseError"
  | "";

export type APIError = {
  code: string;
  detail: string;
  attr: APIAttrs;
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
  | "/hazardous-material-segregations/"
  | "/formula-templates/"
  | "/finance-transactions/"
  | "/reconciliation-queue/"
  | "/organizations/me/"
  | "/roles/"
  | "/us-states/"
  | "/analytics/new-shipment-count/"
  | "/audit-logs/"
  | "/accounting-control/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;

export const HTTP_200_OK = 200;
