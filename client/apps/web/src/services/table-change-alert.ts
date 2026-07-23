import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  tcaAllowlistedTableSchema,
  tcaSubscriptionResponseSchema,
  tcaSubscriptionSchema,
  type TCAAllowlistedTable,
  type TCASubscription,
  type TCASubscriptionFormValues,
  type TCASubscriptionResponse,
} from "@/types/table-change-alert";
import { z } from "zod";

export class TableChangeAlertService {
  public async listAllowlistedTables() {
    const response = await api.get<TCAAllowlistedTable[]>(
      "/tca/allowlisted-tables/",
    );

    return z.array(tcaAllowlistedTableSchema).parse(response);
  }

  public async listSubscriptions(params?: {
    limit?: number;
    offset?: number;
    query?: string;
  }) {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));
    if (params?.query) searchParams.set("query", params.query);

    const queryString = searchParams.toString();
    const response = await api.get<TCASubscriptionResponse>(
      `/tca/subscriptions/?${queryString}`,
    );

    return safeParse(
      tcaSubscriptionResponseSchema,
      response,
      "TCA Subscription List",
    );
  }

  public async getSubscription(id: TCASubscription["id"]) {
    const response = await api.get<TCASubscription>(
      `/tca/subscriptions/${id}`,
    );

    return safeParse(tcaSubscriptionSchema, response, "TCA Subscription");
  }

  public async createSubscription(data: TCASubscriptionFormValues) {
    const response = await api.post<TCASubscription>(
      "/tca/subscriptions/",
      data,
    );

    return safeParse(tcaSubscriptionSchema, response, "TCA Subscription");
  }

  public async updateSubscription(
    id: TCASubscription["id"],
    data: Partial<TCASubscriptionFormValues>,
  ) {
    const response = await api.put<TCASubscription>(
      `/tca/subscriptions/${id}`,
      data,
    );

    return safeParse(tcaSubscriptionSchema, response, "TCA Subscription");
  }

  public async deleteSubscription(id: TCASubscription["id"]) {
    await api.delete(`/tca/subscriptions/${id}`);
  }

  public async pauseSubscription(id: TCASubscription["id"]) {
    const response = await api.patch<TCASubscription>(
      `/tca/subscriptions/${id}/pause`,
    );

    return safeParse(tcaSubscriptionSchema, response, "TCA Subscription");
  }

  public async resumeSubscription(id: TCASubscription["id"]) {
    const response = await api.patch<TCASubscription>(
      `/tca/subscriptions/${id}/resume`,
    );

    return safeParse(tcaSubscriptionSchema, response, "TCA Subscription");
  }

}
