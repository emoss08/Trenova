import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { recordOptionLabel } from "@/lib/select-option-meta";
import type { SelectOptionResource } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";

/**
 * The names of records, looked up in one request and shared with every other reader
 * asking for the same ones. A record that no longer exists is left out of the map.
 */
export function useRecordNames(record: string | undefined, ids: readonly string[]) {
  const { data, isPending } = useQuery({
    queryKey: ["selectOptions", "byIds", record, ids],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: record as SelectOptionResource, ids: [...ids], initialLimit: ids.length },
        { signal },
      ),
    enabled: !!record && ids.length > 0,
    staleTime: 5 * 60_000,
  });

  const names = new Map<string, string>();
  for (const option of data?.results ?? []) names.set(option.id, recordOptionLabel(option));
  return { names, isPending: !!record && ids.length > 0 && isPending };
}
