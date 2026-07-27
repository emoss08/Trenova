import { api } from "@trenova/shared/lib/api";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { useQuery } from "@tanstack/react-query";

export type RoleOption = {
  id: string;
  name: string;
  description?: string | null;
  coreResponsibility?: string | null;
};

const ROLE_OPTIONS_LIMIT = 100;
const ROLE_OPTIONS_STALE_TIME = 5 * 60_000;

/**
 * The roles a preset can be assigned to. Presets are authored rarely and every
 * role in an organization has to be visible at once for an audience choice to
 * make sense, so this fetches the whole list rather than paging it.
 */
export function useRoleOptions() {
  return useQuery({
    queryKey: ["home-layout", "role-options"],
    queryFn: () =>
      api.get<GenericLimitOffsetResponse<RoleOption>>(
        `/roles/select-options/?limit=${ROLE_OPTIONS_LIMIT}&offset=0`,
      ),
    staleTime: ROLE_OPTIONS_STALE_TIME,
    select: (data) => data.results,
  });
}
