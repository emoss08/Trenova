import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { useQuery } from "@tanstack/react-query";

export type RoleOption = {
  id: string;
  name: string;
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
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions({ resource: "ROLE", initialLimit: ROLE_OPTIONS_LIMIT }, { signal }),
    staleTime: ROLE_OPTIONS_STALE_TIME,
    select: (data): RoleOption[] =>
      data.results.map((option) => ({ id: option.id, name: option.label })),
  });
}
