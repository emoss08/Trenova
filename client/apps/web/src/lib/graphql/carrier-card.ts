import {
  DeskCarrierCardDocument,
  DeskCarrierEligibilityDocument,
  DeskCarrierIntelDocument,
  DeskCarrierSettlementsDocument,
  type DeskCarrierCardQuery,
  type DeskCarrierEligibilityQuery,
  type DeskCarrierIntelQuery,
  type DeskCarrierSettlementsQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskCarrier = NonNullable<DeskCarrierCardQuery["carrier"]>;
export type CarrierEligibility = DeskCarrierEligibilityQuery["dispatchCarrierAssignmentPreview"];
export type CarrierIntel = NonNullable<
  NonNullable<DeskCarrierIntelQuery["carrier"]>["intelligence"]
>;
export type CarrierSettlementRow =
  DeskCarrierSettlementsQuery["carrierSettlements"]["edges"][number]["node"];

/** One page of a carrier's settlements. `total` is read with the first page only. */
export type CarrierSettlementsPage = {
  rows: CarrierSettlementRow[];
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

/**
 * One carrier as the Desk's card reads it. A carrier the server does not
 * return is an error rather than an empty card, so the card says it could not
 * be read.
 */
export async function fetchDeskCarrier(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskCarrier> {
  const data = await requestGraphQL({
    document: DeskCarrierCardDocument,
    operationName: "DeskCarrierCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.carrier) {
    throw new Error("Carrier not found");
  }
  return data.carrier;
}

/** Whether the carrier can be tendered a load now, from its record and its intelligence. */
export async function fetchCarrierEligibility(
  carrierId: string,
  options?: { signal?: AbortSignal },
): Promise<CarrierEligibility> {
  const data = await requestGraphQL({
    document: DeskCarrierEligibilityDocument,
    operationName: "DeskCarrierEligibility",
    variables: { carrierId },
    signal: options?.signal,
  });
  return data.dispatchCarrierAssignmentPreview;
}

/** The carrier's current intelligence snapshot, or null when it has never been vetted. */
export async function fetchCarrierIntel(
  carrierId: string,
  options?: { signal?: AbortSignal },
): Promise<CarrierIntel | null> {
  const data = await requestGraphQL({
    document: DeskCarrierIntelDocument,
    operationName: "DeskCarrierIntel",
    variables: { id: carrierId },
    signal: options?.signal,
  });
  return data.carrier?.intelligence ?? null;
}

/** A page of the carrier's settlements, newest pay date first. */
export async function fetchCarrierSettlements(
  carrierId: string,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<CarrierSettlementsPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskCarrierSettlementsDocument,
    operationName: "DeskCarrierSettlements",
    variables: {
      input: {
        first: page.first,
        after: page.after,
        fieldFilters: [{ field: "carrierId", operator: "eq", value: carrierId }],
        sort: [{ field: "payDate", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const rows = data.carrierSettlements.edges.map((edge) => edge.node);
  return {
    rows,
    total: firstPage ? (data.carrierSettlements.totalCount ?? rows.length) : null,
    endCursor: data.carrierSettlements.pageInfo.endCursor ?? null,
    hasNextPage: data.carrierSettlements.pageInfo.hasNextPage,
  };
}
