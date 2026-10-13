import {
  DEFAULT_SIDEBAR_PREFERENCES_QUERY,
  updateSidebarPreferences,
} from "@/lib/graphql/sidebar-preferences";
import { sidebarPreferences } from "@/lib/queries/sidebar-preferences";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const SIDEBAR_PREFERENCES_STALE_TIME = 60_000;

export function useSidebarPreferences() {
  return useQuery({
    ...sidebarPreferences.effective(),
    placeholderData: DEFAULT_SIDEBAR_PREFERENCES_QUERY,
    staleTime: SIDEBAR_PREFERENCES_STALE_TIME,
    select: (data) => data.sidebarPreferences,
  });
}

export function useSidebarCustomizationOptions(enabled = true) {
  return useQuery({
    ...sidebarPreferences.options(),
    enabled,
    select: (data) => data.sidebarCustomizationOptions,
  });
}

export function useUpdateSidebarPreferences() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateSidebarPreferences,
    onSuccess: (updated) => {
      queryClient.setQueryData(sidebarPreferences.effective().queryKey, {
        sidebarPreferences: updated,
      });
      void queryClient.invalidateQueries({
        queryKey: sidebarPreferences.effective().queryKey,
      });
    },
  });
}
