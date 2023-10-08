/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

export interface ApiResponse<T> extends Record<string, unknown> {
  count: number;
  next?: string;
  previous?: string;
  results: T[];
}

type APIAttrs = "nonFieldErrors" | "validationError" | "All" | "";

export type APIError = {
  code: string;
  detail: string;
  attr: APIAttrs;
};

export type API_ENDPOINTS =
  | "/shipment_types/"
  | "/order_types/"
  | "/users/"
  | "/job_titles/"
  | "/groups/"
  | "/permissions/"
  | "/gl_accounts/"
  | "/revenue_codes/"
  | "/division_codes/"
  | "/organizations/"
  | "/organizations/depots"
  | "/departments/"
  | "/email_control/"
  | "/email_profiles/"
  | "/email_log/"
  | "/tax_rates/"
  | "/table_change_alerts/"
  | "/notification_types/"
  | "/notification_settings/"
  | "/workers/"
  | "/worker_profiles/"
  | "/worker_comments/"
  | "/worker_contacts/"
  | "/billing_control/"
  | "/billing_queue/"
  | "/billing_history/"
  | "/charge_types/"
  | "/accessorial_charges/"
  | "/document_classifications/"
  | "/billing_log_entry/"
  | "/hazardous_materials/"
  | "/commodities/"
  | "/customers/"
  | "/customer_fuel_tables/"
  | "/customer_email_profiles/"
  | "/customer_rule_profiles/"
  | "/delivery_slots/"
  | "/equipment_types/"
  | "/tractors/"
  | "/trailers/"
  | "/equipment_manufacturers/"
  | "/equipment_maintenance_plans/"
  | "/locations_categories/"
  | "/locations/"
  | "/location_contacts/"
  | "/location_comments/"
  | "/comment_types/"
  | "/delay_codes/"
  | "/fleet_codes/"
  | "/dispatch_control/"
  | "/rates/"
  | "/feasibility_tool_control/"
  | "/integration_vendors/"
  | "/integrations/"
  | "/google_api/"
  | "/routes/"
  | "/route_control/"
  | "/qualifier_codes/"
  | "/stop_comments/"
  | "/service_incidents/"
  | "/stops/"
  | "/shipment_control/"
  | "/reason_codes/"
  | "/shipments/"
  | "/shipment_documents/"
  | "/shipment_comments/"
  | "/additional_charges/"
  | "/movements/"
  | "/invoice_control/"
  | "/custom_reports/"
  | "/user_reports/"
  | "/log_entries/"
  | "/schema/"
  | "/docs/"
  | "/change_password/"
  | "/reset_password/"
  | "/change_email/"
  | "/schema/redoc/"
  | "/login/"
  | "/logout/"
  | "/me/"
  | "/system_health/"
  | "/bill_invoice/"
  | "/active_triggers/"
  | "/mass_bill_shipments/"
  | "/active_sessions/"
  | "/active_threads/"
  | "/table_columns/"
  | "/transfer_to_billing/"
  | "/generate_excel_report/"
  | "/plugin_list/"
  | "/plugin_install/"
  | "/cache_manager/"
  | "/untransfer_invoice/"
  | "/get_columns/"
  | "/generate_report/"
  | "/user/notifications/"
  | "/billing/shipments_ready/"
  | "/tags/"
  | "/finance_transactions/"
  | "/reconciliation_queue/"
  | "/accounting_control/";
