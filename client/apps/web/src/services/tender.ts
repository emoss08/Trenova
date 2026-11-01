import {
  bounceTenderGraphQL,
  cancelTenderGraphQL,
  createSpotTenderGraphQL,
  createWaterfallTenderGraphQL,
  respondToTenderOfferGraphQL,
} from "@/lib/graphql/tender";
import type {
  CreateSpotTenderInput,
  RespondToTenderOfferInput,
} from "@trenova/graphql/generated/graphql";
import { api } from "@trenova/shared/lib/api";
import { errorParams } from "@trenova/shared/lib/error-params";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  bounceTenderPayloadSchema,
  publicTenderCounterPayloadSchema,
  publicTenderOfferSchema,
  recordTenderResponsePayloadSchema,
  spotTenderPayloadSchema,
  waterfallTenderPayloadSchema,
  type BounceTenderPayload,
  type PublicTenderCounterPayload,
  type PublicTenderOffer,
  type RecordTenderResponsePayloadInput,
  type SpotTenderPayload,
  type WaterfallTenderPayload,
} from "@trenova/shared/types/tender";

export function toSpotTenderInput(payload: SpotTenderPayload): CreateSpotTenderInput {
  return {
    shipmentMoveId: payload.shipmentMoveId,
    mode: payload.mode,
    lines: payload.lines.map((line) => ({
      carrierId: line.carrierId,
      rateMethod: line.rateMethod,
      rate: line.rate.toString(),
      offerTtlSeconds: line.offerTtlSeconds,
      channel: line.channel,
      email: line.channel === "EDI" ? null : line.email || null,
    })),
    overrideInsuranceWarnings: payload.overrideInsuranceWarnings,
  };
}

/**
 * The backend flags insurance-warning rejections with an `overridable` param so the
 * dialog can offer a confirm-and-resubmit instead of a dead-end toast.
 */
export function isOverridableInsuranceWarningError(error: unknown): error is GraphQLRequestError {
  return (
    error instanceof GraphQLRequestError &&
    error.isBusinessError() &&
    errorParams(error).overridable === "true"
  );
}

/**
 * Only the fields the chosen action reads are sent: a decline reason left in the
 * form after switching to Accept, or a counter rate after switching to Decline,
 * must not reach the server as part of a different answer.
 */
export function toRespondToTenderOfferInput(
  payload: RecordTenderResponsePayloadInput,
): RespondToTenderOfferInput {
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
  /** Starts a routing-guide waterfall; the screening names guide carriers left out or warned. */
  public async createWaterfall(payload: WaterfallTenderPayload) {
    const parsed = waterfallTenderPayloadSchema.parse(payload);
    const result = await createWaterfallTenderGraphQL({
      shipmentMoveId: parsed.shipmentMoveId,
      routingGuideId: parsed.routingGuideId ?? null,
    });
    return {
      ...result.tender,
      screening: result.screening
        ? { skipped: result.screening.skipped, warned: result.screening.warned }
        : null,
    };
  }

  public async createSpot(payload: SpotTenderPayload) {
    return createSpotTenderGraphQL(toSpotTenderInput(spotTenderPayloadSchema.parse(payload)));
  }

  public async cancel(tenderId: string, reason: string): Promise<void> {
    await cancelTenderGraphQL(tenderId, reason);
  }

  public async recordResponse(
    offerId: string,
    payload: RecordTenderResponsePayloadInput,
  ): Promise<void> {
    await respondToTenderOfferGraphQL(offerId, toRespondToTenderOfferInput(payload));
  }

  public async bounce(tenderId: string, payload: BounceTenderPayload): Promise<void> {
    const parsed = bounceTenderPayloadSchema.parse(payload);
    await bounceTenderGraphQL(tenderId, parsed.reason);
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
