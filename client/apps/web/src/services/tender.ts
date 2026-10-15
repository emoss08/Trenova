import { api, ApiRequestError } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  bounceTenderPayloadSchema,
  publicTenderCounterPayloadSchema,
  publicTenderOfferSchema,
  recordTenderResponsePayloadSchema,
  spotTenderPayloadSchema,
  tenderSchema,
  waterfallTenderPayloadSchema,
  waterfallTenderResultSchema,
  type BounceTenderPayload,
  type PublicTenderCounterPayload,
  type PublicTenderOffer,
  type RecordTenderResponsePayloadInput,
  type SpotTenderPayload,
  type Tender,
  type WaterfallTenderPayload,
  type WaterfallTenderResult,
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

type RecordTenderResponseRequestBody = {
  action: RecordTenderResponsePayloadInput["action"];
  declineReason: string;
  counterRate?: string;
  counterNote?: string;
};

/**
 * Only the fields the chosen action reads are sent: a decline reason left in the
 * form after switching to Accept, or a counter rate after switching to Decline,
 * must not reach the server as part of a different answer.
 */
export function toRecordTenderResponseRequestBody(
  payload: RecordTenderResponsePayloadInput,
): RecordTenderResponseRequestBody {
  const parsed = recordTenderResponsePayloadSchema.parse(payload);
  switch (parsed.action) {
    case "Decline":
      return { action: parsed.action, declineReason: parsed.declineReason.trim() };
    case "Counter":
      return {
        action: parsed.action,
        declineReason: "",
        counterRate: parsed.counterRate.trim(),
        counterNote: parsed.counterNote.trim(),
      };
    case "Accept":
      return { action: parsed.action, declineReason: "" };
  }
}

export class TenderService {
  public async createWaterfall(payload: WaterfallTenderPayload): Promise<WaterfallTenderResult> {
    const parsed = waterfallTenderPayloadSchema.parse(payload);
    const response = await api.post<WaterfallTenderResult>("/tenders/waterfall/", {
      shipmentMoveId: parsed.shipmentMoveId,
      routingGuideId: parsed.routingGuideId ?? undefined,
    });
    return safeParse(waterfallTenderResultSchema, response, "Tender");
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
    payload: RecordTenderResponsePayloadInput,
  ): Promise<void> {
    await api.post(
      `/tenders/offers/${offerId}/respond/`,
      toRecordTenderResponseRequestBody(payload),
    );
  }

  public async bounce(tenderId: string, payload: BounceTenderPayload): Promise<void> {
    const parsed = bounceTenderPayloadSchema.parse(payload);
    await api.post(`/tenders/${tenderId}/bounce/`, { reason: parsed.reason });
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

  public async counterPublicOffer(
    token: string,
    payload: PublicTenderCounterPayload,
  ): Promise<void> {
    const parsed = publicTenderCounterPayloadSchema.parse(payload);
    await api.post(`/tender-offers/${encodeURIComponent(token)}/counter/`, {
      rate: parsed.rate,
      note: parsed.note.trim(),
    });
  }
}
