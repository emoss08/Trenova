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
  | "/dispatch-controls/"
  | "/fiscal-years/"
  | "/fiscal-periods/"
  | "/location-categories/"
  | "/locations/"
  | "/document-types/"
  | "/hold-reasons/"
  | "/hazmat-segregation-rules/"
  | "/distance-overrides/"
  | "/audit-entries/"
  | "/tca/subscriptions/";

export type API_ENDPOINTS = `${BaseEndpoint}${"" | `?${string}`}`;

export type SELECT_OPTIONS_ENDPOINTS = API_ENDPOINTS | `${API_ENDPOINTS}select-options/`;
