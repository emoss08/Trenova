import z from "zod";

export type GenericLimitOffsetResponse<T> = {
  results: T[];
  count: number;
  next: string | null;
  prev: string | null;
};

export function createLimitOffsetResponse<ItemType extends z.ZodType>(itemSchema: ItemType) {
  return z.object({
    results: z.array(itemSchema),
    count: z.number(),
    next: z.string().optional(),
    prev: z.string().optional(),
  });
}

export type BaseEndpoint =
  | "/formula-templates/"
  | "/equipment-types/"
  | "/equipment-manufacturers/"
  | "/fleet-codes/"
  | "/roles/"
  | "/us-states/"
  | "/accessorial-charges/"
  | "/users/"
  | "/workers/"
  | "/worker-pto/"
  | "/tractors/"
  | "/trailers/"
  | "/service-types/"
  | "/sequence-configs/"
  | "/shipments/"
  | "/shipment-types/"
  | "/hazardous-materials/"
  | "/commodities/"
  | "/customers/"
  | "/account-types/"
  | "/custom-fields/definitions/"
  | "/custom-fields/resource-types/"
  | "/api-keys/"
  | "/gl-accounts/"
  | "/accounting-control/"
  | "/invoice-adjustment-controls/"
  | "/dispatch-controls/"
  | "/fiscal-years/"
  | "/fiscal-periods/"
  | "/location-categories/"
  | "/locations/"
  | "/document-types/"
  | "/documents/"
  | "/hold-reasons/"
  | "/hazmat-segregation-rules/"
  | "/distance-overrides/"
  | "/document-packet-rules/"
  | "/audit-entries/"
  | "/tca/subscriptions/"
  | "/billing-queue/"
  | "/billing-queue/stats/"
  | "/billing-queue/filter-presets/"
  | "/billing/invoices/"
  | "/billing/invoice-adjustments/approvals/"
  | "/billing/invoice-adjustments/reconciliation-exceptions/"
  | "/billing/invoice-adjustments/batches/"
  | "/billing/invoice-adjustments/summary/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;

export type SELECT_OPTIONS_ENDPOINTS = API_ENDPOINTS | `${API_ENDPOINTS}select-options/`;
