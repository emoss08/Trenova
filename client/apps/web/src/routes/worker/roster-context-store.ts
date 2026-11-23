import {
  buildDataTableQueryKey,
  fetchDataTablePage,
} from "@/hooks/data-table/use-data-table-query";
import type { WorkerRow } from "@/lib/graphql/worker-table";
import { PREFETCH_STALE_TIME_MS } from "@/lib/route-prefetch";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { rosterViews } from "@trenova/shared/lib/worker-roster-views";
import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
  FieldFilter,
} from "@trenova/shared/types/data-table";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { useQuery, useQueryClient, type QueryClient, type QueryKey } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { create } from "zustand";

export const ROSTER_QUERY_KEY = "worker-list";

export type RosterPage = GenericLimitOffsetResponse<WorkerRow>;

export type RosterScope = {
  graphql: DataTableGraphQLSource<WorkerRow>;
  pageSize: number;
  options: Omit<DataTableQueryOptions, "cursor">;
};

export type RosterContext = {
  scope: RosterScope;
  cursors: Readonly<Record<number, string | null>>;
  pageIndex: number;
  label: string;
  search: string;
};

export type RosterDirection = "previous" | "next";

export type RosterNeighbors = {
  index: number;
  position: number;
  total: number | null;
  previous: string | null;
  next: string | null;
  canPrevious: boolean;
  canNext: boolean;
};

type RosterContextState = {
  context: RosterContext | null;
  setContext: (context: RosterContext | null) => void;
  clear: () => void;
};

export const useRosterContextStore = create<RosterContextState>()((set) => ({
  context: null,
  setContext: (context) => set({ context }),
  clear: () => set({ context: null }),
}));

type CachedPage = {
  queryKey: QueryKey;
  pageIndex: number;
  pageSize: number;
  options: DataTableQueryOptions;
  variables: unknown;
  data: RosterPage | undefined;
  observed: boolean;
};

function isRosterPage(value: unknown): value is RosterPage {
  return (
    typeof value === "object" &&
    value !== null &&
    Array.isArray((value as { results?: unknown }).results)
  );
}

function cachedRosterPages(queryClient: QueryClient): CachedPage[] {
  const pages: CachedPage[] = [];
  for (const query of queryClient.getQueryCache().findAll({ queryKey: [ROSTER_QUERY_KEY] })) {
    const [, pagination, options, variables] = query.queryKey as readonly unknown[];
    if (typeof pagination !== "object" || pagination === null) continue;
    const { pageIndex, pageSize } = pagination as { pageIndex?: unknown; pageSize?: unknown };
    if (typeof pageIndex !== "number" || typeof pageSize !== "number") continue;
    const data = query.state.data;
    pages.push({
      queryKey: query.queryKey,
      pageIndex,
      pageSize,
      options: (options ?? {}) as DataTableQueryOptions,
      variables,
      data: isRosterPage(data) ? data : undefined,
      observed: query.getObserversCount() > 0,
    });
  }
  return pages;
}

function withoutCursor(options: DataTableQueryOptions): Omit<DataTableQueryOptions, "cursor"> {
  return Object.fromEntries(Object.entries(options).filter(([key]) => key !== "cursor")) as Omit<
    DataTableQueryOptions,
    "cursor"
  >;
}

function sameValue(a: unknown, b: unknown): boolean {
  return JSON.stringify(sortedValue(a)) === JSON.stringify(sortedValue(b));
}

function sortedValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortedValue);
  if (typeof value === "object" && value !== null) {
    return Object.fromEntries(
      Object.entries(value)
        .filter(([, entry]) => entry !== undefined)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, entry]) => [key, sortedValue(entry)]),
    );
  }
  return value;
}

function sameScope(page: CachedPage, origin: CachedPage): boolean {
  return (
    page.pageSize === origin.pageSize &&
    sameValue(withoutCursor(page.options), withoutCursor(origin.options)) &&
    sameValue(page.variables, origin.variables)
  );
}

export function rosterPageKey(context: RosterContext, pageIndex: number): QueryKey {
  return buildDataTableQueryKey(
    ROSTER_QUERY_KEY,
    context.scope.graphql,
    { pageIndex, pageSize: context.scope.pageSize },
    { ...context.scope.options, cursor: context.cursors[pageIndex] ?? null },
  );
}

export function rosterPageOptions(context: RosterContext, pageIndex: number) {
  return {
    queryKey: rosterPageKey(context, pageIndex),
    queryFn: ({ signal }: { signal: AbortSignal }): Promise<RosterPage | null> =>
      fetchDataTablePage<WorkerRow>({
        pageSize: context.scope.pageSize,
        options: { ...context.scope.options, cursor: context.cursors[pageIndex] ?? null },
        graphql: context.scope.graphql,
        signal,
      }),
  };
}

export function captureRosterContext(
  queryClient: QueryClient,
  {
    workerId,
    graphql,
    search,
    label,
  }: {
    workerId: string;
    graphql: DataTableGraphQLSource<WorkerRow>;
    search: string;
    label: (options: Omit<DataTableQueryOptions, "cursor">) => string;
  },
): RosterContext | null {
  const pages = cachedRosterPages(queryClient);
  const holding = pages.filter((page) => page.data?.results.some((row) => row.id === workerId));
  const origin = holding.find((page) => page.observed) ?? holding[0];
  if (!origin) return null;

  const cursors: Record<number, string | null> = { 0: null };
  for (const page of pages) {
    if (!sameScope(page, origin)) continue;
    cursors[page.pageIndex] = page.options.cursor ?? null;
  }
  cursors[origin.pageIndex] = origin.options.cursor ?? null;

  const options = withoutCursor(origin.options);
  return {
    scope: { graphql, pageSize: origin.pageSize, options },
    cursors,
    pageIndex: origin.pageIndex,
    label: label(options),
    search,
  };
}

export function rosterNeighbors(
  context: RosterContext,
  page: RosterPage | null | undefined,
  workerId: string,
  total: number | null,
): RosterNeighbors | null {
  if (!page) return null;
  const index = page.results.findIndex((row) => row.id === workerId);
  if (index === -1) return null;
  const previousCursorKnown =
    context.pageIndex > 0 && Object.hasOwn(context.cursors, context.pageIndex - 1);
  const hasNextPage = page.pageInfo?.hasNextPage === true && Boolean(page.pageInfo.endCursor);
  return {
    index,
    position: context.pageIndex * context.scope.pageSize + index + 1,
    total,
    previous: page.results[index - 1]?.id ?? null,
    next: page.results[index + 1]?.id ?? null,
    canPrevious: index > 0 || previousCursorKnown,
    canNext: index < page.results.length - 1 || hasNextPage,
  };
}

export async function stepRoster(
  queryClient: QueryClient,
  context: RosterContext,
  workerId: string,
  direction: RosterDirection,
): Promise<{ workerId: string; context: RosterContext } | null> {
  const page = queryClient.getQueryData<RosterPage>(rosterPageKey(context, context.pageIndex));
  const neighbors = rosterNeighbors(context, page, workerId, null);
  if (!page || !neighbors) return null;

  const inPage = direction === "next" ? neighbors.next : neighbors.previous;
  if (inPage) return { workerId: inPage, context };

  const canCross = direction === "next" ? neighbors.canNext : neighbors.canPrevious;
  if (!canCross) return null;

  const pageIndex = context.pageIndex + (direction === "next" ? 1 : -1);
  const cursors =
    direction === "next"
      ? { ...context.cursors, [pageIndex]: page.pageInfo?.endCursor ?? null }
      : context.cursors;
  const crossed: RosterContext = { ...context, cursors, pageIndex };
  const fetched = await queryClient.fetchQuery({
    ...rosterPageOptions(crossed, pageIndex),
    staleTime: PREFETCH_STALE_TIME_MS,
  });
  if (!fetched) return null;
  const landed =
    direction === "next" ? fetched.results[0] : fetched.results[fetched.results.length - 1];
  if (!landed) return null;
  return { workerId: landed.id, context: crossed };
}

function sameFilters(applied: readonly FieldFilter[], view: readonly FieldFilter[]): boolean {
  if (applied.length !== view.length) return false;
  return view.every((wanted) =>
    applied.some(
      (filter) =>
        filter.field === wanted.field &&
        filter.operator === wanted.operator &&
        (typeof wanted.value === "number" || sameValue(filter.value, wanted.value)),
    ),
  );
}

export function rosterLabel(
  t: TranslateFn,
  {
    options,
    missingCredential,
    now,
  }: {
    options: Omit<DataTableQueryOptions, "cursor">;
    missingCredential: { name: string | null } | null;
    now: number;
  },
): string {
  if (missingCredential) {
    return missingCredential.name
      ? t("Workers missing {0}", missingCredential.name)
      : t("Workers missing a credential");
  }
  const query = options.query?.trim() ?? "";
  if (query !== "") return t("Workers matching “{0}”", query);
  const filters = options.fieldFilters ?? [];
  const groups = options.filterGroups ?? [];
  if (filters.length === 0 && groups.length === 0) return t("All workers");
  const view = rosterViews(now).find(
    (candidate) =>
      candidate.filters.length > 0 &&
      groups.length === 0 &&
      sameFilters(filters, candidate.filters),
  );
  return view ? view.label : t("Filtered workers");
}

const IDLE_KEY = [ROSTER_QUERY_KEY, "stepper-idle"] as const;

function rosterPageQuery(context: RosterContext | null, pageIndex: number) {
  if (!context) {
    return {
      queryKey: IDLE_KEY as QueryKey,
      queryFn: () => Promise.resolve<RosterPage | null>(null),
      enabled: false,
    };
  }
  return { ...rosterPageOptions(context, pageIndex), enabled: false };
}

export type RosterStepper = {
  available: boolean;
  position: number | null;
  total: number | null;
  label: string | null;
  rosterSearch: string;
  canPrevious: boolean;
  canNext: boolean;
  stepping: RosterDirection | null;
  step: (direction: RosterDirection) => Promise<string | null>;
};

export function useRosterStepper(workerId: string): RosterStepper {
  const queryClient = useQueryClient();
  const context = useRosterContextStore((state) => state.context);
  const setContext = useRosterContextStore((state) => state.setContext);
  const page = useQuery(rosterPageQuery(context, context?.pageIndex ?? 0));
  const first = useQuery(rosterPageQuery(context, 0));
  const busy = useRef(false);
  const [stepping, setStepping] = useState<RosterDirection | null>(null);

  const totalCount = first.data?.pageInfo?.totalCount;
  const total = typeof totalCount === "number" ? totalCount : null;
  const neighbors = context ? rosterNeighbors(context, page.data, workerId, total) : null;

  const step = async (direction: RosterDirection): Promise<string | null> => {
    if (busy.current || !context) return null;
    busy.current = true;
    setStepping(direction);
    try {
      const result = await stepRoster(queryClient, context, workerId, direction);
      if (!result) return null;
      setContext(result.context);
      return result.workerId;
    } finally {
      busy.current = false;
      setStepping(null);
    }
  };

  return {
    available: neighbors !== null,
    position: neighbors?.position ?? null,
    total: neighbors?.total ?? null,
    label: context?.label ?? null,
    rosterSearch: context?.search ?? "",
    canPrevious: neighbors?.canPrevious ?? false,
    canNext: neighbors?.canNext ?? false,
    stepping,
    step,
  };
}
