import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeftIcon, ChevronRightIcon, LightbulbIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { InsightDetailCard } from "./_components/insight-detail-card";
import { InsightDetailPanel } from "./_components/insight-detail-panel";
import { InsightFilterBar } from "./_components/insight-filter-bar";
import {
  DEFAULT_FILTERS,
  hasActiveFilters,
  pageCount,
  parseFilters,
  serializeFilters,
  statusesFor,
  type InsightFilterState,
} from "./_components/insight-filters";

const PAGE_SIZE = 20;

export function InsightsPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  // The URL is the filter state, not a copy of it. A filtered view can then be
  // linked, reloaded and reached with the back button, and there is no second
  // source of truth to drift.
  const filters = useMemo(() => parseFilters(searchParams), [searchParams]);

  // Read once at mount: staleness is measured against a horizon over a day out,
  // so a fresher reading changes no card, and reading the clock during render
  // makes output depend on when React happened to re-run.
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const browseParams = useMemo(
    () => ({
      statuses: statusesFor(filters.status),
      categories: filters.categories,
      severities: filters.severities,
      limit: PAGE_SIZE,
      offset: (filters.page - 1) * PAGE_SIZE,
    }),
    [filters],
  );

  const insightsQuery = useQuery(queries.insight.browse(browseParams));
  const insights = insightsQuery.data?.results ?? [];
  const total = insightsQuery.data?.total ?? 0;
  const pages = pageCount(total, PAGE_SIZE);

  const applyFilters = useCallback(
    (next: InsightFilterState) => setSearchParams(serializeFilters(next)),
    [setSearchParams],
  );

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queries.insight._def });
  }, [queryClient]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Insights"),
        description: t(
          "Findings computed from your own records. The figures come from your data; where an explanation was written by AI, the card says so.",
        ),
      }}
    >
      <InsightFilterBar filters={filters} total={total} onChange={applyFilters} />

      {insightsQuery.isLoading ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-40" />
          <Skeleton className="h-40" />
        </div>
      ) : insights.length === 0 ? (
        <EmptyState
          filtered={hasActiveFilters(filters)}
          onClear={() => applyFilters(DEFAULT_FILTERS)}
        />
      ) : (
        <div className="flex flex-col gap-3">
          {insights.map((insight) => (
            <InsightDetailCard
              key={insight.id}
              insight={insight}
              now={now}
              onChanged={refresh}
              onOpen={() => applyFilters({ ...filters, selected: insight.id })}
            />
          ))}
        </div>
      )}

      {filters.selected !== null && (
        <InsightDetailPanel
          insightId={filters.selected}
          onClose={() => applyFilters({ ...filters, selected: null })}
        />
      )}

      {pages > 1 && (
        <Pager
          page={filters.page}
          pages={pages}
          onPage={(page) => applyFilters({ ...filters, page })}
        />
      )}
    </PageLayout>
  );
}

/**
 * Two different emptinesses, said differently.
 *
 * "Nothing matches these filters" tells someone to widen them. "Nothing needs
 * your attention" is good news. Rendering the same sentence for both leaves a
 * reader unsure whether the operation is healthy or the page is broken.
 */
function EmptyState({ filtered, onClear }: { filtered: boolean; onClear: () => void }) {
  const t = useT();

  return (
    <div className="border-border flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16 text-center">
      <span className="border-border bg-muted/50 flex size-10 items-center justify-center rounded-full border">
        <LightbulbIcon className="text-muted-foreground size-5" />
      </span>
      <p className="text-muted-foreground max-w-md text-sm">
        {filtered
          ? t("No findings match these filters.")
          : t(
              "Nothing needs your attention. Findings appear here when service, billing, or compliance moves in the wrong direction.",
            )}
      </p>
      {filtered && (
        <Button variant="outline" size="sm" onClick={onClear}>
          {t("Clear filters")}
        </Button>
      )}
    </div>
  );
}

function Pager({
  page,
  pages,
  onPage,
}: {
  page: number;
  pages: number;
  onPage: (page: number) => void;
}) {
  const t = useT();

  return (
    <div className="flex items-center justify-center gap-3">
      <Button
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
        aria-label={t("Previous page")}
      >
        <ChevronLeftIcon className="size-4" />
      </Button>
      <span className="text-muted-foreground text-xs tabular-nums">
        {t("Page {0} of {1}", String(page), String(pages))}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={page >= pages}
        onClick={() => onPage(page + 1)}
        aria-label={t("Next page")}
      >
        <ChevronRightIcon className="size-4" />
      </Button>
    </div>
  );
}
