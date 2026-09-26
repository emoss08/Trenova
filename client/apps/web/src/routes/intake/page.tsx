import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { captureBatchesQuery } from "@/lib/queries/capture";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ScanLineIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { BatchList } from "./_components/batch-list";
import { BatchWorkspace } from "./_components/batch-workspace";
import { IntakeRail } from "./_components/intake-rail";
import {
  INTAKE_VIEWS,
  batchFilter,
  parseIntakeFilter,
  viewLabel,
  writeIntakeFilter,
  type IntakeFilter,
  type IntakeView,
} from "./_components/queue-filter";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const SEARCH_DEBOUNCE_MS = 250;
const CLOCK_TICK_MS = 60_000;

function countFilter(filter: IntakeFilter) {
  const { sort: _sort, ...rest } = batchFilter(filter);
  return rest;
}

/**
 * What is waiting to be filed under the rail's filters, whatever view is open
 * and whatever is typed in the search: the rail's loud count.
 */
function waitingFilter(filter: IntakeFilter) {
  return countFilter({ ...filter, view: "waiting", query: "" });
}

export const prefetch: RoutePrefetch = ({ request }) => {
  const filter = parseIntakeFilter(new URL(request.url).searchParams);

  return [
    captureBatchesQuery(batchFilter(filter)),
    queries.capture.batchCount(countFilter(filter)),
  ];
};

function useNow(): number {
  const [now, setNow] = useState(nowInSeconds);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(nowInSeconds()), CLOCK_TICK_MS);
    return () => window.clearInterval(timer);
  }, []);

  return now;
}

/**
 * Intake: everything scanned or printed into Trenova that is not filed yet,
 * as one queue.
 *
 * Three panes, as the inbox has: the views, the stacks, and the open stack
 * with its documents. The address holds all of it, so a link to a stack opens
 * that stack and the back button walks back through what was opened.
 */
export function IntakePage() {
  const t = useT();
  const now = useNow();
  const [searchParams, setSearchParams] = useSearchParams();

  const filter = useMemo(() => parseIntakeFilter(searchParams), [searchParams]);
  const openId = searchParams.get("batch");
  const [search, setSearch] = useState(filter.query);
  const debouncedSearch = useDebounce(search, SEARCH_DEBOUNCE_MS);

  const setFilter = useCallback(
    (next: IntakeFilter, replace = false) =>
      setSearchParams((current) => writeIntakeFilter(current, next), { replace }),
    [setSearchParams],
  );

  useEffect(() => {
    if (debouncedSearch.trim() !== filter.query.trim()) {
      setFilter({ ...filter, query: debouncedSearch }, true);
    }
  }, [debouncedSearch, filter, setFilter]);

  const openBatch = useCallback(
    (id: string | null) =>
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        if (id === null) {
          next.delete("batch");
        } else {
          next.set("batch", id);
        }
        return next;
      }),
    [setSearchParams],
  );

  const batchesQuery = useInfiniteQuery(captureBatchesQuery(batchFilter(filter)));
  const totalQuery = useQuery(queries.capture.batchCount(countFilter(filter)));
  const waitingQuery = useQuery(queries.capture.batchCount(waitingFilter(filter)));

  const batches = useMemo(
    () => batchesQuery.data?.pages.flatMap((page) => page.batches) ?? [],
    [batchesQuery.data],
  );
  const title = viewLabel(t, filter.view);

  return (
    <PageLayout
      fill
      className="p-0"
      pageHeaderProps={{
        title: t("Intake"),
        description: t(
          "Paper scanned and documents printed into Trenova, split into documents and waiting to be filed onto their records",
        ),
      }}
    >
      <div className="flex min-h-0 flex-1">
        <aside className="border-border bg-sunken hidden w-60 shrink-0 flex-col border-r md:flex">
          <IntakeRail filter={filter} waiting={waitingQuery.data} onChange={setFilter} />
        </aside>

        <div
          className={cn(
            "border-border min-w-0 flex-col lg:flex lg:w-96 lg:shrink-0 lg:border-r xl:w-[28rem]",
            openId === null ? "flex flex-1 lg:flex-none" : "hidden",
          )}
        >
          <div className="border-border flex gap-1 overflow-x-auto border-b px-3 py-2 md:hidden">
            {INTAKE_VIEWS.map((view: IntakeView) => {
              const active = filter.view === view;
              return (
                <button
                  key={view}
                  type="button"
                  aria-pressed={active}
                  onClick={() => setFilter({ ...filter, view })}
                  className={cn(
                    "ui-focus-ring shrink-0 rounded-full px-2.5 py-1 text-xs transition-colors",
                    active
                      ? "bg-nav-active text-nav-active-foreground"
                      : "text-foreground-muted ring-foreground/10 ring-1",
                  )}
                >
                  {viewLabel(t, view)}
                </button>
              );
            })}
          </div>
          <BatchList
            title={title}
            list={{
              batches,
              total: totalQuery.data,
              isLoading: batchesQuery.isLoading,
              isError: batchesQuery.isError,
              hasNextPage: batchesQuery.hasNextPage,
              isFetchingNextPage: batchesQuery.isFetchingNextPage,
              fetchNextPage: () => void batchesQuery.fetchNextPage(),
              retry: () => void batchesQuery.refetch(),
            }}
            openId={openId}
            now={now}
            search={search}
            onSearchChange={setSearch}
            sort={filter.sort}
            onSortChange={(sort) => setFilter({ ...filter, sort })}
            onOpen={openBatch}
            empty={
              filter.view === "waiting"
                ? {
                    title: t("Nothing is waiting to be filed"),
                    description: t(
                      "Scan from a record's Documents tab, or print into Trenova from any program, and the pages land here.",
                    ),
                  }
                : {
                    title: t("Nothing here"),
                    description: t("Stacks appear here as they are scanned or printed."),
                  }
            }
          />
        </div>

        <main
          className={cn("min-w-0 flex-1 flex-col", openId === null ? "hidden lg:flex" : "flex")}
        >
          {openId === null ? (
            <NothingOpen waiting={waitingQuery.data ?? 0} />
          ) : (
            <BatchWorkspace
              key={openId}
              batchId={openId}
              now={now}
              onClose={() => openBatch(null)}
            />
          )}
        </main>
      </div>
    </PageLayout>
  );
}

function NothingOpen({ waiting }: { waiting: number }) {
  const t = useT();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      <ScanLineIcon className="text-foreground-subtle size-8" aria-hidden />
      <p className="text-sm font-medium">
        {waiting > 0
          ? t(
              "{0, plural, one {# stack is waiting to be filed} other {# stacks are waiting to be filed}}",
              waiting,
            )
          : t("Choose a stack to file")}
      </p>
      <p className="text-foreground-subtle max-w-xs text-xs leading-relaxed">
        {t(
          "Each stack is split into documents. Check where each one goes, then file them together.",
        )}
      </p>
    </div>
  );
}
