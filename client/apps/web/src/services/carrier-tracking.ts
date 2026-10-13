import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  publicCheckCallResultSchema,
  publicTrackingViewSchema,
  type PublicCheckCallPayload,
  type PublicCheckCallResult,
  type PublicTrackingView,
} from "@trenova/shared/types/carrier-tracking";

export class CarrierTrackingService {
  public async getPublicTracking(token: string): Promise<PublicTrackingView> {
    const response = await api.get<PublicTrackingView>(
      `/carrier-tracking-links/${encodeURIComponent(token)}/`,
    );

    return safeParse(publicTrackingViewSchema, response, "PublicTrackingView");
  }

  public async postCheckCall(
    token: string,
    payload: PublicCheckCallPayload,
  ): Promise<PublicCheckCallResult> {
    const response = await api.post<PublicCheckCallResult>(
      `/carrier-tracking-links/${encodeURIComponent(token)}/check-calls/`,
      payload,
    );

    return safeParse(publicCheckCallResultSchema, response, "PublicCheckCallResult");
  }
}
