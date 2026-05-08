import { api } from "@/lib/api";
import type { AnalyticsData, AnalyticsParams } from "@/types/analytics";

export class AnalyticsService {
  async get(params: AnalyticsParams) {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params?.endDate) searchParams.set("endDate", String(params.endDate));
    if (params?.startDate) searchParams.set("startDate", String(params.startDate));
    if (params?.timezone) searchParams.set("timezone", params.timezone);
    if (params?.windowDays) searchParams.set("windowDays", String(params.windowDays));
    if (params?.include) searchParams.set("include", params.include);

    const queryString = searchParams.toString();

    const response = await api.get<AnalyticsData>(`/analytics/?${queryString}`);

    return response;
  }
}
