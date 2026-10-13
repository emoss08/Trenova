import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  publicShipmentViewSchema,
  type PublicShipmentView,
} from "@trenova/shared/types/shipment-tracking";

function linkPath(token: string): string {
  return `/shipment-tracking-links/${encodeURIComponent(token)}/`;
}

export class ShipmentTrackingService {
  public async getPublicView(token: string): Promise<PublicShipmentView> {
    const response = await api.get<PublicShipmentView>(linkPath(token));

    return safeParse(publicShipmentViewSchema, response, "PublicShipmentView");
  }

  /** Where the proof of delivery downloads from: a plain link the browser follows. */
  public proofOfDeliveryUrl(token: string): string {
    return `${API_BASE_URL}${linkPath(token)}proof-of-delivery/`;
  }
}
