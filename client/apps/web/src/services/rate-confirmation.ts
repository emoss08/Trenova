import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  publicRateConfirmationSchema,
  rateConfirmationListSchema,
  rateConfirmationSchema,
  type PublicRateConfirmation,
  type RateConfirmation,
} from "@trenova/shared/types/rate-confirmation";

export type ConfirmPublicRateConfirmationPayload = {
  signerName: string;
  signerTitle: string;
};

export class RateConfirmationService {
  public async listByMove(moveId: string): Promise<RateConfirmation[]> {
    const response = await api.get<RateConfirmation[]>(
      `/shipment-moves/${moveId}/rate-confirmations/`,
    );

    return safeParse(rateConfirmationListSchema, response ?? [], "RateConfirmationList");
  }

  public async generate(moveId: string): Promise<RateConfirmation> {
    const response = await api.post<RateConfirmation>(
      `/shipment-moves/${moveId}/rate-confirmations/`,
    );

    return safeParse(rateConfirmationSchema, response, "RateConfirmation");
  }

  public async send(rateConfirmationId: string): Promise<RateConfirmation> {
    const response = await api.post<RateConfirmation>(
      `/rate-confirmations/${rateConfirmationId}/send/`,
    );

    return safeParse(rateConfirmationSchema, response, "RateConfirmation");
  }

  public async confirm(
    rateConfirmationId: string,
    confirmedByName: string,
  ): Promise<RateConfirmation> {
    const response = await api.post<RateConfirmation>(
      `/rate-confirmations/${rateConfirmationId}/confirm/`,
      { confirmedByName },
    );

    return safeParse(rateConfirmationSchema, response, "RateConfirmation");
  }

  public async void(rateConfirmationId: string, reason: string): Promise<RateConfirmation> {
    const response = await api.post<RateConfirmation>(
      `/rate-confirmations/${rateConfirmationId}/void/`,
      { reason },
    );

    return safeParse(rateConfirmationSchema, response, "RateConfirmation");
  }

  public async getPublicRateConfirmation(token: string): Promise<PublicRateConfirmation> {
    const response = await api.get<PublicRateConfirmation>(
      `/rate-confirmation-links/${encodeURIComponent(token)}/`,
    );

    return safeParse(publicRateConfirmationSchema, response, "PublicRateConfirmation");
  }

  public async confirmPublicRateConfirmation(
    token: string,
    payload: ConfirmPublicRateConfirmationPayload,
  ): Promise<void> {
    await api.post(`/rate-confirmation-links/${encodeURIComponent(token)}/confirm/`, {
      signerName: payload.signerName,
      signerTitle: payload.signerTitle,
    });
  }
}
