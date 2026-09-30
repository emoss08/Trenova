import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { captureBatchesQuery } from "@/lib/queries/capture";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ScanLineIcon, SlidersHorizontalIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { BatchList } from "./_components/batch-list";
import { BatchWorkspace } from "./_components/batch-workspace";
import { IntakeRail } from "./_components/intake-rail";
import { useUrlSearch } from "./_components/use-url-search";
import {
  INTAKE_VIEWS,
  batchFilter,
  parseIntakeFilter,
  viewLabel,
  writeIntakeFilter,
  type IntakeFilter,
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
  return countFilter({ ...filter, view: "waiting", query: "", received: "any" });
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
  const [filtersOpen, setFiltersOpen] = useState(false);

  const setFilter = useCallback(
    (next: IntakeFilter, replace = false) =>
      setSearchParams((current) => writeIntakeFilter(current, next), { replace }),
    [setSearchParams],
  );
  const commitSearch = useCallback(
    (query: string, replace: boolean) =>
      setSearchParams(
        (current) => writeIntakeFilter(current, { ...parseIntakeFilter(current), query }),
        { replace },
      ),
    [setSearchParams],
  );
  const [search, setSearch] = useUrlSearch({
    query: filter.query,
    onCommit: commitSearch,
    delay: SEARCH_DEBOUNCE_MS,
  });

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
  const searching = filter.query.trim() !== "";
  const narrowedBy = (filter.source === null ? 0 : 1) + (filter.mine ? 1 : 0);
  const viewItems = INTAKE_VIEWS.map((view) => ({ value: view, label: viewLabel(t, view) }));

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
          <div className="border-border flex items-center gap-2 border-b px-3 py-2 md:hidden">
            <div className="min-w-0 flex-1 overflow-x-auto">
              <SegmentedControl
                aria-label={t("Intake views")}
                items={viewItems}
                value={filter.view}
                onValueChange={(view) => setFilter({ ...filter, view })}
              />
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="shrink-0"
              onClick={() => setFiltersOpen(true)}
            >
              <SlidersHorizontalIcon aria-hidden />
              {t("Filters")}
              {narrowedBy > 0 && (
                <Badge variant="brand" className="tabular-nums">
                  {narrowedBy}
                </Badge>
              )}
            </Button>
          </div>
          <BatchList
            title={title}
            list={{
              batches,
              total: totalQuery.data,
              isLoading: batchesQuery.isLoading,
              isError: batchesQuery.isError,
              error: batchesQuery.error,
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
            received={filter.received}
            onReceivedChange={(received) => setFilter({ ...filter, received })}
            onOpen={openBatch}
            empty={
              searching
                ? {
                    title: t("No stack matches “{0}”", filter.query.trim()),
                    description: t(
                      "Try other words, or clear the search to see every stack in this view.",
                    ),
                    action: (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => setFilter({ ...filter, query: "" })}
                      >
                        {t("Clear the search")}
                      </Button>
                    ),
                  }
                : filter.view === "waiting"
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

      <Sheet open={filtersOpen} onOpenChange={setFiltersOpen}>
        <SheetContent side="left" className="gap-0 p-0">
          <SheetHeader className="border-border border-b">
            <SheetTitle>{t("Views and filters")}</SheetTitle>
            <SheetDescription>{t("Choose which stacks the queue shows.")}</SheetDescription>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col">
            <IntakeRail
              filter={filter}
              waiting={waitingQuery.data}
              onChange={(next) => {
                setFilter(next);
                if (next.mine === filter.mine) {
                  setFiltersOpen(false);
                }
              }}
            />
          </div>
        </SheetContent>
      </Sheet>
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
