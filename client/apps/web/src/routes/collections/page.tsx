import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { collection } from "@/lib/queries/collection";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useDeferredValue } from "react";
import { CollectionAccountSheet } from "./_components/collection-account-sheet";
import { CollectionsTable } from "./_components/collections-table";
import { COLLECTOR_SCOPES, collectionsSearchParamsParser } from "./use-collections-state";

const PAGE_SIZE = 50;

const COLUMNS = [
  { label: "Customer" },
  { label: "Overdue", numeric: true },
  { label: "Oldest", numeric: true },
  { label: "Promise" },
  { label: "Follow-up" },
  { label: "Collector" },
  { label: "Last contact" },
] as const;

const SCOPE_LABEL: Record<(typeof COLLECTOR_SCOPES)[number], string> = defineLabels({
  All: "Everyone's",
  Mine: "Mine",
  Unassigned: "Unassigned",
});

/**
 * Collections: every customer with overdue receivables, ranked so the first
 * row is the call to make next, and each account worked from its panel.
 */
export function CollectionsPage() {
  const t = useT();
  const [state, setState] = useQueryStates(collectionsSearchParamsParser);
  const search = useDeferredValue(state.query);

  const { data, isLoading, isError, refetch } = useQuery({
    ...collection.worklist({
      scope: state.scope,
      dueOnly: state.due,
      search,
      limit: PAGE_SIZE,
      offset: (state.page - 1) * PAGE_SIZE,
    }),
    placeholderData: keepPreviousData,
  });

  const summary = data?.summary;
  const pages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;
  const hasFilters = state.scope !== "All" || state.due || state.query !== "";
  const clearFilters = () => void setState({ scope: "All", due: false, query: "", page: 1 });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Collections"),
        description: t(
          "Customers with overdue receivables, ranked by who to call first. Open one to log a call, record a promise to pay, or schedule the next follow-up.",
        ),
      }}
    >
      <>
        <KpiStrip aria-label={t("Collections summary")}>
          <KpiStripItem
            label={t("Overdue")}
            value={
              summary && summary.overdue.length > 0
                ? formatMinor(summary.overdue[0].amountMinor, summary.overdue[0].currencyCode)
                : formatMinor(0, "USD")
            }
            sub={
              summary && summary.overdue.length > 1
                ? summary.overdue
                    .slice(1)
                    .map((total) => formatMinor(total.amountMinor, total.currencyCode))
                    .join(" · ")
                : t("{0, plural, one {# account} other {# accounts}}", summary?.accounts ?? 0)
            }
            tone="danger"
          />
          <KpiStripItem
            label={t("Follow-ups due")}
            value={summary?.followUpsDue ?? 0}
            tone="warning"
            active={state.due}
            onClick={() => void setState({ due: !state.due, page: 1 })}
          />
          <KpiStripItem
            label={t("Broken promises")}
            value={summary?.brokenPromises ?? 0}
            tone={summary?.brokenPromises ? "danger" : "muted"}
          />
          <KpiStripItem
            label={t("Unassigned")}
            value={summary?.unassigned ?? 0}
            tone="muted"
            active={state.scope === "Unassigned"}
            onClick={() =>
              void setState({
                scope: state.scope === "Unassigned" ? "All" : "Unassigned",
                page: 1,
              })
            }
          />
        </KpiStrip>

        <div className="flex flex-wrap items-center gap-3">
          <SegmentedControl
            aria-label={t("Collector")}
            value={state.scope}
            onValueChange={(scope) => void setState({ scope, page: 1 })}
            items={COLLECTOR_SCOPES.map((value) => ({ value, label: SCOPE_LABEL[value] }))}
          />
          <div className="flex items-center gap-2">
            <Switch
              id="collections-due-only"
              checked={state.due}
              onCheckedChange={(checked) => void setState({ due: checked, page: 1 })}
            />
            <Label htmlFor="collections-due-only" className="text-sm font-normal">
              {t("Due today only")}
            </Label>
          </div>
          <Input
            className="h-8 w-64"
            placeholder={t("Search customers")}
            leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
            value={state.query}
            onChange={(event) => void setState({ query: event.target.value, page: 1 })}
          />
        </div>

        {isLoading ? (
          <Skeleton className="h-64 w-full rounded-md" />
        ) : isError ? (
          <EmptyTable
            title="The worklist could not be loaded"
            description="Something went wrong reading receivables. Try again in a moment."
            columns={COLUMNS}
            action={
              <Button variant="outline" size="sm" onClick={() => void refetch()}>
                {t("Try again")}
              </Button>
            }
          />
        ) : !data || data.rows.length === 0 ? (
          <EmptyTable
            title={hasFilters ? "Nothing matches" : "Nothing overdue"}
            description={
              hasFilters
                ? "No account fits those filters. Widen them, or clear them to see everyone who owes."
                : "No customer has an overdue invoice. An account appears here the day one of its invoices passes its due date."
            }
            columns={COLUMNS}
            onClearFilters={hasFilters ? clearFilters : undefined}
          />
        ) : (
          <>
            <CollectionsTable
              rows={data.rows}
              selectedCustomerId={state.customerId}
              onSelect={(customerId) => void setState({ customerId })}
            />
            {pages > 1 ? (
              <div className="flex items-center justify-end gap-2 text-sm">
                <span className="text-muted-foreground tabular-nums">
                  {t("Page {0} of {1}", state.page, pages)}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={state.page <= 1}
                  onClick={() => void setState({ page: state.page - 1 })}
                >
                  {t("Previous")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={state.page >= pages}
                  onClick={() => void setState({ page: state.page + 1 })}
                >
                  {t("Next")}
                </Button>
              </div>
            ) : null}
          </>
        )}

        <CollectionAccountSheet
          customerId={state.customerId}
          onClose={() => void setState({ customerId: null })}
        />
      </>
    </PageLayout>
  );
}
