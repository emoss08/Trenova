import { useLocation, useRouter } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";

/** What the query becomes: a whole new query, or a change to the one the address holds now. */
export type QueryParamsUpdate =
  | URLSearchParams
  | Readonly<Record<string, string>>
  | ((current: URLSearchParams) => URLSearchParams);

export type SetQueryParams = (
  update: QueryParamsUpdate,
  options?: { replace?: boolean; viewTransition?: boolean },
) => void;

/**
 * The address's query as URLSearchParams, for a page whose parsers and writers work on the
 * whole query (Triage's location, Intake's filter, the sign-in notices). Per-key state reads
 * better through nuqs. A change made from a function starts from the query the address holds
 * when it runs, not the one this render saw, so two quick changes do not drop each other, and
 * nuqs, which writes through the router too, merges what it still holds on top. A change of the
 * query alone keeps the page where it is scrolled and keeps what the history entry carried
 * (why the page was opened), as a filter changes the entry rather than starting another.
 */
export function useQueryParams(): [URLSearchParams, SetQueryParams] {
  const router = useRouter();
  const searchStr = useLocation({ select: (location) => location.searchStr });
  const params = useMemo(() => new URLSearchParams(searchStr), [searchStr]);

  const setQueryParams = useCallback<SetQueryParams>(
    (update, options) => {
      const { pathname, searchStr: currentSearch, hash } = router.latestLocation;
      const next =
        typeof update === "function"
          ? update(new URLSearchParams(currentSearch))
          : new URLSearchParams(update);
      const query = next.toString();
      void router.navigate({
        href: `${pathname}${query ? `?${query}` : ""}${hash ? `#${hash}` : ""}`,
        replace: options?.replace,
        viewTransition: options?.viewTransition,
        resetScroll: false,
        state: (current) => current,
      });
    },
    [router],
  );

  return [params, setQueryParams];
}
