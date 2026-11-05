import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  tcaAllowlistedTableSchema,
  tcaSubscriptionResponseSchema,
  type TCAAllowlistedTable,
  type TCASubscriptionResponse,
} from "@/types/table-change-alert";
import { z } from "zod";

export class TableChangeAlertService {
  public async listAllowlistedTables() {
    const response = await api.get<TCAAllowlistedTable[]>("/tca/allowlisted-tables/");

    return z.array(tcaAllowlistedTableSchema).parse(response);
  }

  public async listSubscriptions(params?: { limit?: number; offset?: number; query?: string }) {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));
    if (params?.query) searchParams.set("query", params.query);

    const queryString = searchParams.toString();
    const response = await api.get<TCASubscriptionResponse>(`/tca/subscriptions/?${queryString}`);

    return safeParse(tcaSubscriptionResponseSchema, response, "TCA Subscription List");
  }
}

export const tableChangeAlertService = new TableChangeAlertService();
