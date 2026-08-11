import { api, ApiRequestError } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  publicTenderOfferSchema,
  spotTenderPayloadSchema,
  tenderSchema,
  waterfallTenderPayloadSchema,
  type PublicTenderOffer,
  type RecordTenderResponsePayload,
  type SpotTenderPayload,
  type Tender,
  type WaterfallTenderPayload,
} from "@trenova/shared/types/tender";

type SpotTenderLineRequestBody = {
  carrierId: string;
  rateMethod: SpotTenderPayload["lines"][number]["rateMethod"];
  rate: string;
  offerTtlSeconds: number;
  channel: SpotTenderPayload["lines"][number]["channel"];
  email?: string;
};

type SpotTenderRequestBody = {
  shipmentMoveId: string;
  mode: SpotTenderPayload["mode"];
  lines: SpotTenderLineRequestBody[];
  overrideInsuranceWarnings: boolean;
};

export function toSpotTenderRequestBody(payload: SpotTenderPayload): SpotTenderRequestBody {
  return {
    shipmentMoveId: payload.shipmentMoveId,
    mode: payload.mode,
    lines: payload.lines.map((line) => ({
      carrierId: line.carrierId,
      rateMethod: line.rateMethod,
      rate: line.rate.toString(),
      offerTtlSeconds: line.offerTtlSeconds,
      channel: line.channel,
      email: line.channel === "EDI" ? undefined : line.email || undefined,
    })),
    overrideInsuranceWarnings: payload.overrideInsuranceWarnings,
  };
}

/**
 * The backend flags insurance-warning rejections with an `overridable` param so the
 * dialog can offer a confirm-and-resubmit instead of a dead-end toast.
 */
export function isOverridableInsuranceWarningError(error: unknown): error is ApiRequestError {
  return (
    error instanceof ApiRequestError &&
    error.isBusinessError() &&
    error.getParams().overridable === "true"
  );
}

export class TenderService {
  public async createWaterfall(payload: WaterfallTenderPayload): Promise<Tender> {
    const parsed = waterfallTenderPayloadSchema.parse(payload);
    const response = await api.post<Tender>("/tenders/waterfall/", {
      shipmentMoveId: parsed.shipmentMoveId,
      routingGuideId: parsed.routingGuideId ?? undefined,
    });
    return safeParse(tenderSchema, response, "Tender");
  }

  public async createSpot(payload: SpotTenderPayload): Promise<Tender> {
    const response = await api.post<Tender>(
      "/tenders/spot/",
      toSpotTenderRequestBody(spotTenderPayloadSchema.parse(payload)),
    );
    return safeParse(tenderSchema, response, "Tender");
  }

  public async cancel(tenderId: string, reason: string): Promise<void> {
    await api.post(`/tenders/${tenderId}/cancel/`, { reason });
  }

  public async recordResponse(
    offerId: string,
    payload: RecordTenderResponsePayload,
  ): Promise<void> {
    await api.post(`/tenders/offers/${offerId}/respond/`, {
      action: payload.action,
      declineReason: payload.action === "Decline" ? payload.declineReason : "",
    });
  }

  public async getPublicOffer(token: string): Promise<PublicTenderOffer> {
    const response = await api.get<PublicTenderOffer>(
      `/tender-offers/${encodeURIComponent(token)}/`,
    );
    return safeParse(publicTenderOfferSchema, response, "TenderOffer");
  }

  public async acceptPublicOffer(token: string): Promise<void> {
    await api.post(`/tender-offers/${encodeURIComponent(token)}/accept/`);
  }

  public async declinePublicOffer(token: string, reason: string): Promise<void> {
    await api.post(`/tender-offers/${encodeURIComponent(token)}/decline/`, {
      reason,
    });
  }
}
