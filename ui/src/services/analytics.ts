import { http } from "@/lib/http-client";
import { AnalyticsData, AnalyticsPage } from "@/types/analytics";

export type AnalyticsParams = {
  page: AnalyticsPage;
  startDate?: number;
  endDate?: number;
  limit?: number;
};

export class AnalyticsAPI {
  async getAnalytics(params: AnalyticsParams) {
    const { page, startDate, endDate, limit } = params;

    // Build query parameters
    const queryParams: Record<string, string> = {
      page: page.toString(),
    };

    if (startDate) {
      queryParams.startDate = startDate.toString();
    }

    if (endDate) {
      queryParams.endDate = endDate.toString();
    }

    if (limit) {
      queryParams.limit = limit.toString();
    }

    const response = await http.get<AnalyticsData>("/analytics/", {
      params: queryParams,
    });

    return response.data;
  }
}
