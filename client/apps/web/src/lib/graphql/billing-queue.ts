import {
  AssignBillingQueueBillerDocument,
  BillingQueueFormulaTemplateDocument,
  BillingQueueItemsDocument,
  BillingQueueRateAgreementDocument,
  BillingQueueShipmentDocument,
  type BillingQueueFormulaTemplateQuery,
  type BillingQueueRateAgreementQuery,
  type BillingQueueShipmentQuery,
  UpdateBillingQueueStatusDocument,
  type BillingQueueAssignInput,
  type BillingQueueItemsFilterInput,
  type BillingQueueItemsQuery,
  type BillingQueueUpdateStatusInput,
  type FieldFilterInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingQueueItemSchema, type BillingQueueItem } from "@trenova/shared/types/billing-queue";

type UpdateBillingQueueStatusResponse = {
  updateBillingQueueStatus: unknown;
};

type AssignBillingQueueBillerResponse = {
  assignBillingQueueBiller: unknown;
};

export async function updateBillingQueueStatusGraphQL(
  id: BillingQueueItem["id"],
  input: BillingQueueUpdateStatusInput,
): Promise<BillingQueueItem> {
  const data = await requestGraphQL<UpdateBillingQueueStatusResponse>({
    document: UpdateBillingQueueStatusDocument,
    operationName: "UpdateBillingQueueStatus",
    variables: { id, input },
  });

  return safeParse(billingQueueItemSchema, data.updateBillingQueueStatus, "BillingQueueItem");
}

export async function assignBillingQueueBillerGraphQL(
  id: BillingQueueItem["id"],
  input: BillingQueueAssignInput,
): Promise<BillingQueueItem> {
  const data = await requestGraphQL<AssignBillingQueueBillerResponse>({
    document: AssignBillingQueueBillerDocument,
    operationName: "AssignBillingQueueBiller",
    variables: { id, input },
  });

  return safeParse(billingQueueItemSchema, data.assignBillingQueueBiller, "BillingQueueItem");
}

type RequestOptions = { signal?: AbortSignal };

export type BillingQueueItemConnection = BillingQueueItemsQuery["billingQueueItems"];

/** One row of a queue lane: the item with the issue it leads with and Nova's take. */
export type BillingQueueRow = BillingQueueItemConnection["edges"][number]["node"];

/** A page of the queue, oldest first; the total is counted on the first page only. */
export async function listBillingQueueItemsGraphQL(
  req: {
    first: number;
    after?: string | null;
    query?: string;
    fieldFilters: FieldFilterInput[];
    filter: BillingQueueItemsFilterInput;
  },
  options?: RequestOptions,
): Promise<BillingQueueItemConnection> {
  const data = await requestGraphQL({
    document: BillingQueueItemsDocument,
    operationName: "BillingQueueItems",
    variables: {
      input: {
        first: req.first,
        after: req.after ?? undefined,
        query: req.query || undefined,
        fieldFilters: req.fieldFilters.length > 0 ? req.fieldFilters : undefined,
      },
      filter: req.filter,
      includeTotalCount: !req.after,
    },
    signal: options?.signal,
  });
  return data.billingQueueItems;
}

/** The shipment as the queue's Shipment tab reads it: stops, load and rating. */
export type BillingQueueShipment = NonNullable<BillingQueueShipmentQuery["shipment"]>;

export async function getBillingQueueShipmentGraphQL(
  shipmentId: string,
  options?: RequestOptions,
): Promise<BillingQueueShipment | null> {
  const data = await requestGraphQL({
    document: BillingQueueShipmentDocument,
    operationName: "BillingQueueShipment",
    variables: { id: shipmentId },
    signal: options?.signal,
  });

  return data.shipment ?? null;
}

/** The rate agreement behind "Rated by": terms, accessorials and fuel program. */
export type BillingQueueRateAgreement = NonNullable<
  BillingQueueRateAgreementQuery["rateAgreement"]
>;

export async function getBillingQueueRateAgreementGraphQL(
  agreementId: string,
  options?: RequestOptions,
): Promise<BillingQueueRateAgreement | null> {
  const data = await requestGraphQL({
    document: BillingQueueRateAgreementDocument,
    operationName: "BillingQueueRateAgreement",
    variables: { id: agreementId },
    signal: options?.signal,
  });

  return data.rateAgreement ?? null;
}

/** The formula template behind "Rated by" when no contract priced the shipment. */
export type BillingQueueFormulaTemplate = NonNullable<
  BillingQueueFormulaTemplateQuery["formulaTemplate"]
>;

export async function getBillingQueueFormulaTemplateGraphQL(
  templateId: string,
  options?: RequestOptions,
): Promise<BillingQueueFormulaTemplate | null> {
  const data = await requestGraphQL({
    document: BillingQueueFormulaTemplateDocument,
    operationName: "BillingQueueFormulaTemplate",
    variables: { id: templateId },
    signal: options?.signal,
  });

  return data.formulaTemplate ?? null;
}
