import {
  LiveTenderByMoveDocument,
  TendersByShipmentDocument,
  type LiveTenderByMoveQuery,
  type TendersByShipmentQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type LiveTender = NonNullable<LiveTenderByMoveQuery["liveTenderByMove"]>;
export type LiveTenderOffer = NonNullable<LiveTender["offers"]>[number];
export type ShipmentTender = TendersByShipmentQuery["tendersByShipment"][number];

export async function getTendersByShipmentGraphQL(shipmentId: string): Promise<ShipmentTender[]> {
  const data = await requestGraphQL({
    document: TendersByShipmentDocument,
    operationName: "TendersByShipment",
    variables: { shipmentId },
  });
  return data.tendersByShipment;
}

export async function getLiveTenderByMoveGraphQL(moveId: string): Promise<LiveTender | null> {
  const data = await requestGraphQL({
    document: LiveTenderByMoveDocument,
    operationName: "LiveTenderByMove",
    variables: { moveId },
  });
  return data.liveTenderByMove ?? null;
}
