import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  rateConfirmationListSchema,
  rateConfirmationSchema,
  type RateConfirmation,
} from "@trenova/shared/types/rate-confirmation";

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

  public async get(rateConfirmationId: string): Promise<RateConfirmation> {
    const response = await api.get<RateConfirmation>(`/rate-confirmations/${rateConfirmationId}/`);

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
}
