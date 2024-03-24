/*
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

export type API_ENDPOINTS =
  | "/shipment_types/"
  | "/me/favorites/"
  | "/order_types/"
  | "/users/"
  | "me"
  | "/job_titles/"
  | "/groups/"
  | "/permissions/"
  | "/gl_accounts/"
  | "/revenue-codes/"
  | "/division_codes/"
  | "/organizations/"
  | "/depots/"
  | "/departments/"
  | "/email-control/"
  | "/email-profiles/"
  | "/email-log/"
  | "/tax_rates/"
  | "/table-change-alerts/"
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
  | "/location_categories/"
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
  | "/qualifier-codes/"
  | "/stop_comments/"
  | "/service_incidents/"
  | "/stops/"
  | "/shipment_control/"
  | "/reason_codes/"
  | "/shipments/"
  | "/shipments/get_shipment_count_by_status/"
  | "/shipment_documents/"
  | "/shipment_comments/"
  | "/service_types/"
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
  | "/auth/login"
  | "/auth/logout"
  | "/me/"
  | "/system_health/"
  | "/bill_invoice/"
  | "/active_triggers/"
  | "/mass_bill_shipments/"
  | "/active_sessions/"
  | "/active_threads/"
  | "/table-columns/"
  | "/transfer_to_billing/"
  | "/generate_excel_report/"
  | "/plugin_list/"
  | "/plugin_install/"
  | "/cache_manager/"
  | "/untransfer_invoice/"
  | "/get-columns/"
  | "/generate-report/"
  | "/users/notifications/"
  | "/billing/shipments_ready/"
  | "/tags/"
  | "/hazardous_material_segregation/"
  | "/finance_transactions/"
  | "/reconciliation_queue/"
  | "/organization/me/"
  | "/accounting_control/";

// HTTP Status Codes Ripped from Django Rest Framework
export const HTTP_100_CONTINUE = 100;
export const HTTP_101_SWITCHING_PROTOCOLS = 101;
export const HTTP_102_PROCESSING = 102;
export const HTTP_103_EARLY_HINTS = 103;
export const HTTP_200_OK = 200;
export const HTTP_201_CREATED = 201;
export const HTTP_202_ACCEPTED = 202;
export const HTTP_203_NON_AUTHORITATIVE_INFORMATION = 203;
export const HTTP_204_NO_CONTENT = 204;
export const HTTP_205_RESET_CONTENT = 205;
export const HTTP_206_PARTIAL_CONTENT = 206;
export const HTTP_207_MULTI_STATUS = 207;
export const HTTP_208_ALREADY_REPORTED = 208;
export const HTTP_226_IM_USED = 226;
export const HTTP_300_MULTIPLE_CHOICES = 300;
export const HTTP_301_MOVED_PERMANENTLY = 301;
export const HTTP_302_FOUND = 302;
export const HTTP_303_SEE_OTHER = 303;
export const HTTP_304_NOT_MODIFIED = 304;
export const HTTP_305_USE_PROXY = 305;
export const HTTP_306_RESERVED = 306;
export const HTTP_307_TEMPORARY_REDIRECT = 307;
export const HTTP_308_PERMANENT_REDIRECT = 308;
export const HTTP_400_BAD_REQUEST = 400;
export const HTTP_401_UNAUTHORIZED = 401;
export const HTTP_402_PAYMENT_REQUIRED = 402;
export const HTTP_403_FORBIDDEN = 403;
export const HTTP_404_NOT_FOUND = 404;
export const HTTP_405_METHOD_NOT_ALLOWED = 405;
export const HTTP_406_NOT_ACCEPTABLE = 406;
export const HTTP_407_PROXY_AUTHENTICATION_REQUIRED = 407;
export const HTTP_408_REQUEST_TIMEOUT = 408;
export const HTTP_409_CONFLICT = 409;
export const HTTP_410_GONE = 410;
export const HTTP_411_LENGTH_REQUIRED = 411;
export const HTTP_412_PRECONDITION_FAILED = 412;
export const HTTP_413_REQUEST_ENTITY_TOO_LARGE = 413;
export const HTTP_414_REQUEST_URI_TOO_LONG = 414;
export const HTTP_415_UNSUPPORTED_MEDIA_TYPE = 415;
export const HTTP_416_REQUESTED_RANGE_NOT_SATISFIABLE = 416;
export const HTTP_417_EXPECTATION_FAILED = 417;
export const HTTP_418_IM_A_TEAPOT = 418;
export const HTTP_421_MISDIRECTED_REQUEST = 421;
export const HTTP_422_UNPROCESSABLE_ENTITY = 422;
export const HTTP_423_LOCKED = 423;
export const HTTP_424_FAILED_DEPENDENCY = 424;
export const HTTP_425_TOO_EARLY = 425;
export const HTTP_426_UPGRADE_REQUIRED = 426;
export const HTTP_428_PRECONDITION_REQUIRED = 428;
export const HTTP_429_TOO_MANY_REQUESTS = 429;
export const HTTP_431_REQUEST_HEADER_FIELDS_TOO_LARGE = 431;
export const HTTP_451_UNAVAILABLE_FOR_LEGAL_REASONS = 451;
export const HTTP_500_INTERNAL_SERVER_ERROR = 500;
export const HTTP_501_NOT_IMPLEMENTED = 501;
export const HTTP_502_BAD_GATEWAY = 502;
export const HTTP_503_SERVICE_UNAVAILABLE = 503;
export const HTTP_504_GATEWAY_TIMEOUT = 504;
export const HTTP_505_HTTP_VERSION_NOT_SUPPORTED = 505;
export const HTTP_506_VARIANT_ALSO_NEGOTIATES = 506;
export const HTTP_507_INSUFFICIENT_STORAGE = 507;
export const HTTP_508_LOOP_DETECTED = 508;
export const HTTP_509_BANDWIDTH_LIMIT_EXCEEDED = 509;
export const HTTP_510_NOT_EXTENDED = 510;
export const HTTP_511_NETWORK_AUTHENTICATION_REQUIRED = 511;
