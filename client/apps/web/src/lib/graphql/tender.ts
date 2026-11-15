import type {
  LiveTenderByMoveQuery,
  TendersByShipmentQuery,
} from "@trenova/graphql/generated/graphql";
import {
  LiveTenderByMoveDocument,
  TendersByShipmentDocument,
} from "@trenova/graphql/generated/documents/tender/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type LiveTender = NonNullable<LiveTenderByMoveQuery["liveTenderByMove"]>;
export type LiveTenderOffer = NonNullable<LiveTender["offers"]>[number];
export type ShipmentTender = TendersByShipmentQuery["tendersByShipment"][number];

export async function getTendersByShipmentGraphQL(
  shipmentId: string,
  options?: { signal?: AbortSignal },
): Promise<ShipmentTender[]> {
  const data = await requestGraphQL({
    document: TendersByShipmentDocument,
    operationName: "TendersByShipment",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.tendersByShipment;
}

export async function getLiveTenderByMoveGraphQL(
  moveId: string,
  options?: { signal?: AbortSignal },
): Promise<LiveTender | null> {
  const data = await requestGraphQL({
    document: LiveTenderByMoveDocument,
    operationName: "LiveTenderByMove",
    variables: { moveId },
    signal: options?.signal,
  });
  return data.liveTenderByMove ?? null;
}
