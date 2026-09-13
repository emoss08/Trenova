import {
  fetchGraphQLSelectOptions,
  fetchGraphQLSelectedOption,
  type SelectOption,
} from "@/lib/graphql/select-options";
import type { SelectOptionResource } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";

export const SELECT_OPTION_KEY = "select-option";

export function selectOptionQueryOptions(
  resource: SelectOptionResource,
  id: string | null | undefined,
  filters?: Record<string, unknown>,
) {
  return {
    queryKey: [SELECT_OPTION_KEY, resource, id ?? null, filters ?? null] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      id ? fetchGraphQLSelectedOption(resource, id, filters, { signal }) : null,
    enabled: Boolean(id),
    staleTime: 60_000,
  };
}

/**
 * Reads back the one option a form currently holds, for the fields that decide
 * something from the selection itself — whether a course is scored, whether a
 * pay code is taxable. An autocomplete only reports an option when the user
 * picks one, so a form opened on a saved record would otherwise know nothing
 * about the value it is already showing.
 */
export function useSelectOption(
  resource: SelectOptionResource,
  id: string | null | undefined,
  filters?: Record<string, unknown>,
): { option: SelectOption | null; isLoading: boolean } {
  const query = useQuery(selectOptionQueryOptions(resource, id, filters));

  return { option: query.data ?? null, isLoading: query.isLoading };
}

/**
 * The first option a resource offers, which is the one its provider orders
 * first — the default template, the default policy. Fetches a single row rather
 * than the whole list, so a form can preselect without holding the options it
 * no longer renders.
 */
export function useFirstSelectOption(
  resource: SelectOptionResource,
  options?: { enabled?: boolean; filters?: Record<string, unknown> },
): { option: SelectOption | null; isLoading: boolean } {
  const filters = options?.filters;
  const query = useQuery({
    queryKey: [SELECT_OPTION_KEY, resource, "first", filters ?? null] as const,
    queryFn: async ({ signal }) => {
      const response = await fetchGraphQLSelectOptions(
        { resource, initialLimit: 1, filters },
        { signal },
      );
      return response.results[0] ?? null;
    },
    enabled: options?.enabled ?? true,
    staleTime: 60_000,
  });

  return { option: query.data ?? null, isLoading: query.isLoading };
}
