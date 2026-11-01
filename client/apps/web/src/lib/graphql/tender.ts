import {
  BounceTenderDocument,
  CancelTenderDocument,
  CreateSpotTenderDocument,
  CreateWaterfallTenderDocument,
  LiveTenderByMoveDocument,
  RespondToTenderOfferDocument,
  TendersByShipmentDocument,
  type CreateSpotTenderInput,
  type CreateWaterfallTenderInput,
  type LiveTenderByMoveQuery,
  type RespondToTenderOfferInput,
  type TendersByShipmentQuery,
} from "@trenova/graphql/generated/graphql";
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

export async function createWaterfallTenderGraphQL(input: CreateWaterfallTenderInput) {
  const data = await requestGraphQL({
    document: CreateWaterfallTenderDocument,
    operationName: "CreateWaterfallTender",
    variables: { input },
  });
  return data.createWaterfallTender;
}

export async function createSpotTenderGraphQL(input: CreateSpotTenderInput) {
  const data = await requestGraphQL({
    document: CreateSpotTenderDocument,
    operationName: "CreateSpotTender",
    variables: { input },
  });
  return data.createSpotTender;
}

export async function cancelTenderGraphQL(id: string, reason: string) {
  const data = await requestGraphQL({
    document: CancelTenderDocument,
    operationName: "CancelTender",
    variables: { id, reason },
  });
  return data.cancelTender;
}

export async function respondToTenderOfferGraphQL(
  offerId: string,
  input: RespondToTenderOfferInput,
) {
  const data = await requestGraphQL({
    document: RespondToTenderOfferDocument,
    operationName: "RespondToTenderOffer",
    variables: { offerId, input },
  });
  return data.respondToTenderOffer;
}

export async function bounceTenderGraphQL(id: string, reason: string) {
  const data = await requestGraphQL({
    document: BounceTenderDocument,
    operationName: "BounceTender",
    variables: { id, reason },
  });
  return data.bounceTender;
}
