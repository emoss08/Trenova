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
  | "/finance-transactions/"
  | "/reconciliation-queue/"
  | "/organizations/me/"
  | "/roles/"
  | "/accounting-control/";

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
