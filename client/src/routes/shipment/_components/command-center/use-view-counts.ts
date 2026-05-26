import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { shipmentSavedViewCountsAnalyticsSchema } from "@/types/analytics";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { SAVED_VIEWS, type SavedViewId } from "./saved-views";

const browserTimezone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;

function resolveSavedViewCountsTimezone(timezone: string | undefined) {
  if (!timezone || timezone === "auto") return browserTimezone();
  return timezone;
}

export function useSavedViewCounts(enabled = true): Record<SavedViewId, number | undefined> {
  const userTimezone = useAuthStore((state) => state.user?.timezone);
  const timezone = resolveSavedViewCountsTimezone(userTimezone);

  const { data } = useQuery({
    queryKey: ["analytics", "shipment-management", "saved-view-counts", timezone],
    queryFn: async () => {
      const response = await apiService.analyticService.get({
        page: "shipment-management",
        include: "savedViewCounts",
        timezone,
      });
      return shipmentSavedViewCountsAnalyticsSchema.parse(response).savedViewCounts;
    },
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });

  return useMemo(() => {
    const map = {} as Record<SavedViewId, number | undefined>;
    SAVED_VIEWS.forEach((view) => {
      map[view.id] = data?.[view.id];
    });
    return map;
  }, [data]);
}
