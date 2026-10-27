import { useLocalStorage } from "@/hooks/use-local-storage";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { DisplayColumn } from "@/components/assistant/readable-values";
import { isRecordEntityType } from "@/config/record-links";
import type { ArtifactRows } from "@/types/assistant";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useEffect, useMemo, useState } from "react";
import { resumesAfterAgent, tableRowsFrom, type TablePaging } from "../../artifact-payloads";
import {
  isPageSize,
  PAGE_SIZES,
  pageCount,
  pageSlice,
  positionAfter,
  type PageSize,
  type RowsPosition,
} from "./table-pages";

export type TableSort = { key: string; direction: 1 | -1 };

/** A table's rows as the body draws them; the shape gridOf reads every table into. */
export type PagedRow = {
  key: string;
  values: Record<string, unknown>;
  path: string;
  recordId?: string;
};

const SEARCH_DELAY_MS = 300;
const NO_PAGES: readonly ArtifactRows[] = [];

/**
 * What the list lets a person do, from its first page: search it, sort it by
 * which columns, and which columns it withholds. Kept across readings, so a
 * header stays sortable while the page it asked for is read.
 */
type ListAbilities = { searchable: boolean; sortable: string[]; withheld: string[] };

function sameAbilities(known: ListAbilities | null, page: ListAbilities): boolean {
  return (
    known !== null &&
    known.searchable === page.searchable &&
    known.sortable.join("\n") === page.sortable.join("\n") &&
    known.withheld.join("\n") === page.withheld.join("\n")
  );
}
const PAGE_SIZE_KEY = "desk.table.pageSize";

/**
 * A list a table pages through beside the conversation.
 *
 * The rows the agent read come first, as the table holds them. Past them the
 * list is read again from its source a page at a time, with the person's own
 * access, from where the agent's read stopped, so the next page is read as
 * soon as the table opens and Next is instant. Searching or sorting reads the
 * list from its start instead, as does a table cut to fit once the person
 * pages past the rows it holds. A list the person may no longer read stops at
 * the agent's rows. A list that no longer takes the place the agent's read
 * stopped at is read again from its start. The person can read the whole list
 * again, live, from its start in place of the agent's rows, and go back to
 * them; each time they ask is a read of its own, never one kept from before.
 */
export function useTablePages({
  threadId,
  artifactId,
  stamp,
  rows: agentRows,
  columns,
  entity,
  paging,
  rereadable,
  arrange,
}: {
  threadId: string;
  artifactId: string;
  /** When the table last changed, so a page the agent reads into it starts the reading over. */
  stamp: number;
  rows: readonly PagedRow[];
  columns: readonly DisplayColumn[];
  entity: string;
  /** Where the list goes on from the agent's rows; null for a table that is all there is. */
  paging: TablePaging | null;
  /** Whether the list can be read again, live, from its start, in place of the agent's rows. */
  rereadable: boolean;
  /** Filters and sorts rows where they are, for a list the server does not search or sort. */
  arrange: (rows: readonly PagedRow[], needle: string, sort: TableSort | null) => PagedRow[];
}) {
  const [storedSize, setStoredSize] = useLocalStorage<number>(PAGE_SIZE_KEY, PAGE_SIZES[0]);
  const size: PageSize = isPageSize(storedSize) ? storedSize : PAGE_SIZES[0];
  const [page, setPage] = useState(0);
  const [search, setSearchText] = useState("");
  const [sort, setSortState] = useState<TableSort | null>(null);
  const [restarted, setRestarted] = useState(false);
  const [readAt, setReadAt] = useState<number | null>(null);
  const [refused, setRefused] = useState(false);
  const live = readAt !== null;
  const typed = search.trim();
  const q = useDebounce(typed, SEARCH_DELAY_MS);

  const paged = paging !== null && paging.hasMore;
  const remote = paged || live;
  const [abilities, setAbilities] = useState<ListAbilities | null>(null);
  const listSearches = remote && abilities?.searchable === true;
  const remoteSearch = listSearches && q !== "";
  const fromStart = remote && (remoteSearch || sort !== null || restarted || live);
  const start: RowsPosition | null = !remote
    ? null
    : fromStart
      ? { first: true }
      : paging === null
        ? null
        : paging.nextCursor !== ""
          ? { after: paging.nextCursor, first: true }
          : paging.nextOffset !== null
            ? { offset: paging.nextOffset, first: true }
            : null;

  const read = useInfiniteQuery({
    queryKey: assistant.artifactRows(threadId, artifactId, {
      stamp,
      size,
      q: remoteSearch ? q : "",
      sortBy: fromStart ? (sort?.key ?? "") : "",
      sortDirection: sort?.direction === -1 ? "desc" : "asc",
      fromStart,
      readAt: readAt ?? 0,
    }).queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.artifactRows(
        threadId,
        artifactId,
        {
          limit: size,
          after: pageParam.after,
          offset: pageParam.offset,
          q: remoteSearch ? q : undefined,
          sortBy: fromStart ? sort?.key : undefined,
          sortDirection: sort?.direction === -1 ? "desc" : "asc",
          total: pageParam.first,
        },
        { signal },
      ),
    initialPageParam: start ?? { first: true },
    getNextPageParam: (last) => positionAfter(last) ?? undefined,
    enabled: start !== null,
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
  });

  const failure = read.error;
  const locked = failure instanceof ApiRequestError && failure.status === 403;
  const stale = failure instanceof ApiRequestError && failure.getFieldErrors("after").length > 0;
  const pages = read.data?.pages ?? NO_PAGES;
  const first = pages[0];
  const recordEntity = isRecordEntityType(entity) ? entity : null;
  const heldRows = fromStart ? 0 : agentRows.length;

  const buffer = useMemo(() => {
    const out: PagedRow[] = fromStart ? [] : [...agentRows];
    for (const chunk of pages) {
      for (const row of tableRowsFrom(chunk.rows, columns, recordEntity, out.length)) {
        out.push(row);
      }
    }
    return out;
  }, [agentRows, columns, fromStart, pages, recordEntity]);

  if (first !== undefined && !sameAbilities(abilities, first)) {
    setAbilities({
      searchable: first.searchable,
      sortable: first.sortable,
      withheld: first.withheld,
    });
  }

  const hasMore = remote && !locked && (pages.length === 0 || (pages.at(-1)?.hasMore ?? false));
  const filtering = typed !== "" && !listSearches;
  const view = useMemo(
    () =>
      filtering || !remote ? arrange(buffer, filtering ? typed : "", remote ? null : sort) : buffer,
    [arrange, buffer, filtering, remote, sort, typed],
  );
  const total = filtering
    ? view.length
    : !remote
      ? agentRows.length
      : (first?.total ?? (fromStart ? null : (paging?.total ?? null)));
  const totalEstimated =
    !filtering &&
    remote &&
    (first?.total != null ? first.totalEstimated : !fromStart && (paging?.totalEstimated ?? false));
  const count = pageCount({
    total: locked || totalEstimated ? null : total,
    read: view.length,
    hasMore: !filtering && (hasMore || (locked && remote)),
    size,
  });
  const lastPage =
    locked && !filtering ? Math.max(0, Math.ceil(view.length / size) - 1) : count.pages - 1;
  const shownPage = Math.min(page, Math.max(0, lastPage));
  const need = (shownPage + 1) * size;
  const waiting = !filtering && buffer.length < need && hasMore && failure === null;

  if (
    paging !== null &&
    remote &&
    !fromStart &&
    (stale || (!resumesAfterAgent(paging) && need > agentRows.length))
  ) {
    setRestarted(true);
  }
  if (live && locked) {
    setReadAt(null);
    setRefused(true);
  }

  const { hasNextPage, isFetching, fetchNextPage } = read;
  useEffect(() => {
    if (waiting && hasNextPage && !isFetching) {
      void fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetching, waiting]);

  const rows = pageSlice(view, shownPage, size);

  return {
    rows,
    page: shownPage,
    size,
    pages: count.pages,
    exact: count.exact,
    lastPage,
    total,
    totalEstimated: total !== null && totalEstimated,
    readRows: buffer.length,
    shownRows: view.length,
    agentRows: heldRows,
    remote,
    fromStart,
    loading: waiting || (!filtering && read.isFetching && buffer.length < need && failure === null),
    fetching: read.isFetching,
    failure: locked ? null : failure,
    locked: locked || refused,
    live,
    rereadable: rereadable && !locked && !refused,
    filtering,
    searchesList: listSearches,
    sortable: (key: string) => (remote ? (abilities?.sortable.includes(key) ?? false) : true),
    withheld: abilities?.withheld ?? [],
    search,
    searching: remoteSearch,
    sort,
    setPage: (next: number) => setPage(Math.max(0, Math.min(next, lastPage))),
    setSize: (next: PageSize) => {
      setStoredSize(next);
      setPage(Math.floor((shownPage * size) / next));
    },
    setSearch: (text: string) => {
      setSearchText(text);
      setPage(0);
    },
    setSort: (next: TableSort | null) => {
      setSortState(next);
      setPage(0);
    },
    readAgain: () => {
      setReadAt(Date.now());
      if (!paged) {
        setSortState(null);
      }
      setPage(0);
    },
    backToAgent: () => {
      setReadAt(null);
      setPage(0);
    },
    retry: () => {
      void (pages.length === 0 ? read.refetch() : read.fetchNextPage());
    },
  };
}

export type TablePages = ReturnType<typeof useTablePages>;
