import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import { previewShipmentCustomerTracking } from "@/lib/graphql/customer-tracking";
import {
  publicShipmentViewSchema,
  type PublicShipmentView,
} from "@trenova/shared/types/shipment-tracking";

type CustomerTrackingPreview = Awaited<ReturnType<typeof previewShipmentCustomerTracking>>;

function shown(value: string | null | undefined): string | undefined {
  return value ? value : undefined;
}

/**
 * The public page leaves out every part the customer's settings hide; GraphQL sends
 * those parts as empty strings or null, so they are dropped to give the preview the
 * same shape the public link reads.
 */
export function publicViewFromPreview(preview: CustomerTrackingPreview): unknown {
  return {
    companyName: preview.companyName,
    shipmentProNumber: preview.shipmentProNumber,
    fields: preview.fields,
    expiresAt: preview.expiresAt,
    closed: preview.closed,
    status: preview.status ?? undefined,
    bol: shown(preview.bol),
    customerReference: shown(preview.customerReference),
    carrierName: shown(preview.carrierName),
    equipment: shown(preview.equipment),
    pieces: preview.pieces ?? undefined,
    weight: preview.weight ?? undefined,
    lastLocation: preview.lastLocation ?? undefined,
    stops: preview.stops.map((stop) => ({
      sequence: stop.sequence,
      type: stop.type,
      city: stop.city,
      stateCode: stop.stateCode,
      locationName: shown(stop.locationName),
      addressLine: shown(stop.addressLine),
      windowStart: stop.windowStart ?? undefined,
      windowEnd: stop.windowEnd ?? undefined,
      arrivedAt: stop.arrivedAt ?? undefined,
      departedAt: stop.departedAt ?? undefined,
      estimatedArrival: stop.estimatedArrival ?? undefined,
      running: shown(stop.running),
    })),
    holds: preview.holds.length
      ? preview.holds.map((hold) => ({ ...hold, label: shown(hold.label) }))
      : undefined,
    notes: preview.notes.length ? preview.notes : undefined,
    proofOfDelivery: preview.proofOfDelivery ?? undefined,
  };
}

function linkPath(token: string): string {
  return `/shipment-tracking-links/${encodeURIComponent(token)}/`;
}

export class ShipmentTrackingService {
  public async getPublicView(token: string): Promise<PublicShipmentView> {
    const response = await api.get<PublicShipmentView>(linkPath(token));

    return safeParse(publicShipmentViewSchema, response, "PublicShipmentView");
  }

  /**
   * What the shipment's customer would see through a link sent now, for staff
   * to check before sharing. No link is created and no view is counted.
   */
  public async previewForShipment(
    shipmentId: string,
    options?: { signal?: AbortSignal },
  ): Promise<PublicShipmentView> {
    const preview = await previewShipmentCustomerTracking(shipmentId, { signal: options?.signal });

    return safeParse(
      publicShipmentViewSchema,
      publicViewFromPreview(preview),
      "PublicShipmentView",
    );
  }

  /** Where the proof of delivery downloads from: a plain link the browser follows. */
  public proofOfDeliveryUrl(token: string): string {
    return `${API_BASE_URL}${linkPath(token)}proof-of-delivery/`;
  }
}
