import { updateSearch } from "@trenova/shared/hooks/use-search-state";
import { parseSearch, stringifySearch } from "@trenova/shared/lib/router/search";
import { useLocation, useRouter } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";

/** What the query becomes: a whole new query, or a change to the one the address holds now. */
export type QueryParamsUpdate =
  | URLSearchParams
  | Readonly<Record<string, string>>
  | ((current: URLSearchParams) => URLSearchParams);

export type SetQueryParams = (
  update: QueryParamsUpdate,
  options?: { replace?: boolean },
) => Promise<void>;

/**
 * The address's query as URLSearchParams, for a page whose parsers and writers work on the
 * whole query (Triage's location, Intake's filter, the sign-in notices); per-key state is
 * useSearchState. A change made from a function starts from the query the address holds when
 * it runs, not the one this render saw, so two quick changes keep each other. A change of the
 * query alone keeps the page where it is scrolled and keeps what the history entry carried.
 */
export function useQueryParams(): [URLSearchParams, SetQueryParams] {
  const router = useRouter();
  const searchStr = useLocation({ select: (location) => location.searchStr });
  const params = useMemo(() => new URLSearchParams(searchStr), [searchStr]);

  const setQueryParams = useCallback<SetQueryParams>(
    (update, options) =>
      updateSearch(router, (current) => {
        const next =
          typeof update === "function"
            ? update(new URLSearchParams(stringifySearch(current)))
            : new URLSearchParams(update);
        return {
          search: parseSearch(next.toString()),
          push: !(options?.replace ?? false),
          resetScroll: false,
        };
      }),
    [router],
  );

  return [params, setQueryParams];
}
