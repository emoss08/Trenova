import {
  DeskCarrierSettlementCardDocument,
  type DeskCarrierSettlementCardQuery,
  DeskDriverSettlementCardDocument,
  type DeskDriverSettlementCardQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskDriverSettlement = NonNullable<DeskDriverSettlementCardQuery["driverSettlement"]>;
export type DeskDriverSettlementGroup = NonNullable<DeskDriverSettlement["lineSummary"]>[number];
export type DeskDriverSettlementDispute = DeskDriverSettlement["disputes"][number];
export type DeskSettlementLoadShare = NonNullable<DeskDriverSettlement["loadShares"]>[number];

export type DeskCarrierSettlement = NonNullable<
  DeskCarrierSettlementCardQuery["carrierSettlement"]
>;
export type DeskCarrierSettlementGroup = NonNullable<DeskCarrierSettlement["costSummary"]>[number];
export type DeskCarrierMoveInvoice = NonNullable<DeskCarrierSettlement["moveInvoices"]>[number];

/**
 * One worker's settlement as the Desk's card reads it: its lines summed by how
 * each was paid, each load's share and the worker's disputes. A settlement the
 * server does not return is an error rather than an empty card.
 */
export async function fetchDeskDriverSettlement(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskDriverSettlement> {
  const data = await requestGraphQL({
    document: DeskDriverSettlementCardDocument,
    operationName: "DeskDriverSettlementCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.driverSettlement) {
    throw new Error("Settlement not found");
  }
  return data.driverSettlement;
}

/**
 * One carrier's settlement as the Desk's card reads it: its costs summed by
 * kind, each move's invoice against what it costs here, and where the money
 * goes. A settlement the server does not return is an error.
 */
export async function fetchDeskCarrierSettlement(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskCarrierSettlement> {
  const data = await requestGraphQL({
    document: DeskCarrierSettlementCardDocument,
    operationName: "DeskCarrierSettlementCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.carrierSettlement) {
    throw new Error("Settlement not found");
  }
  return data.carrierSettlement;
}
