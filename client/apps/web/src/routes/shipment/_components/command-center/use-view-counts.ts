import { getShipmentSavedViewCountsGraphQL } from "@/lib/graphql/shipment";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
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
      const counts = await getShipmentSavedViewCountsGraphQL(timezone);
      return {
        all: counts?.all ?? 0,
        transit: counts?.transit ?? 0,
        "at-risk": counts?.atRisk ?? 0,
        unassigned: counts?.unassigned ?? 0,
        "delivering-today": counts?.deliveringToday ?? 0,
      };
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
