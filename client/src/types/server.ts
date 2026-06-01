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
  | "/organizations/"
  | "/document-types/"
  | "/documents/"
  | "/hold-reasons/"
  | "/service-failures/"
  | "/service-failure-reason-codes/"
  | "/hazmat-segregation-rules/"
  | "/distance-overrides/"
  | "/distance-profiles/"
  | "/distance-controls/"
  | "/stored-mileages/"
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
  | "/billing/invoice-adjustments/summary/"
  | "/accounting/manual-journals/"
  | "/accounting/journal-reversals/"
  | "/accounting/bank-receipts/"
  | "/accounting/bank-receipt-batches/"
  | "/accounting/bank-receipt-batches/select-options/sources/"
  | "/accounting/bank-receipt-work-items/"
  | "/edi/partners/"
  | "/edi/communication-profiles/"
  | "/edi/mapping-profiles/"
  | "/edi/catalog/document-types/"
  | "/edi/catalog/source-context/fields/"
  | "/edi/catalog/partner-settings/fields/"
  | "/edi/templates/"
  | "/edi/document-profiles/"
  | "/edi/documents/preview/"
  | "/edi/documents/generate/"
  | "/edi/messages/"
  | "/edi/test-cases/"
  | "/edi/transfers/"
  | "/edi/load-tenders/"
  | "/email-profiles/"
  | "/email-logs/"
  | "/email-suppressions/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;

export type SELECT_OPTIONS_ENDPOINTS = API_ENDPOINTS | `${API_ENDPOINTS}select-options/`;
