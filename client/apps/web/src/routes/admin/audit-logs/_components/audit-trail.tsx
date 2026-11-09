import { DataTable } from "@/components/data-table/data-table";
import { tableFilterSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { dayGroupScope, dayKey } from "@/lib/day-grouping";
import { auditLogTableGraphQLConfig, type AuditEntryRow } from "@/lib/graphql/audit-log-table";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { Switch } from "@trenova/shared/components/ui/switch";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { dateTimeFormatter } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  DataTableEmptyStateRenderProps,
  DataTableGroup,
  DataTableGroupKey,
  FieldFilter,
} from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { parseAsArrayOf, parseAsString, useQueryState, useQueryStates } from "nuqs";
import { useId, useMemo, useState, type ReactNode } from "react";
import { RequestEntries } from "./audit-entry-parts";
import { AuditInspector } from "./audit-inspector";
import { entriesByDay } from "./audit-overview-model";
import { ONE_PER_REQUEST_STORAGE_KEY, type AuditQuickView } from "./audit-page-state";
import { getTrailColumns, TRAIL_HIDDEN_COLUMNS } from "./audit-trail-columns";
import { useAuditOverview } from "./use-audit-overview";

/** Days back the trail names in its group headers; older rows fall under their date. */
const NAMED_DAYS = 400;

const QUICK_VIEW_FILTERS: Record<AuditQuickView, FieldFilter[]> = {
  all: [],
  changes: [{ field: "operation", operator: "notin", value: ["read", "export"] }],
  security: [{ field: "category", operator: "eq", value: "Security" }],
  agents: [{ field: "channel", operator: "in", value: ["agent_conversation", "agent_run"] }],
  deletes: [{ field: "operation", operator: "in", value: ["delete", "export"] }],
};

const EMPTY_COLUMNS = [
  { label: "Time" },
  { label: "Who" },
  { label: "What happened" },
  { label: "Record" },
  { label: "Via" },
] as const;

const collapsedParser = parseAsArrayOf(parseAsString).withDefault([]);

function dayGroups(t: TranslateFn, timezone: string, entries: ReadonlyMap<string, number>) {
  const groups: DataTableGroup[] = [];
  const now = Math.floor(Date.now() / 1000);
  const thisYear = dateTimeFormatter({ year: "numeric", timeZone: timezone }).format(new Date());
  const short = dateTimeFormatter({
    weekday: "short",
    month: "short",
    day: "numeric",
    timeZone: timezone,
  });
  const long = dateTimeFormatter({
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: timezone,
  });
  const yearOf = dateTimeFormatter({ year: "numeric", timeZone: timezone });
  for (let back = 0; back < NAMED_DAYS; back++) {
    const at = now - back * 86_400;
    const key = dayKey(at, timezone);
    if (groups.length > 0 && groups[groups.length - 1].key === key) continue;
    const day = new Date(at * 1000);
    const date = (yearOf.format(day) === thisYear ? short : long).format(day);
    const count = entries.get(key);
    groups.push({
      key,
      label: back === 0 ? t("Today · {0}", date) : back === 1 ? t("Yesterday · {0}", date) : date,
      aggregate:
        count === undefined ? undefined : (
          <span className="text-muted-foreground text-xs font-normal">
            {t("{0, plural, one {# entry} other {# entries}}", count)}
          </span>
        ),
    });
  }
  return groups;
}

function TrailEmpty({ hasActiveFilters, onClearFilters }: DataTableEmptyStateRenderProps) {
  const t = useT();
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? t("Nothing matches") : t("No audit entries yet")}
      description={
        hasActiveFilters
          ? t(
              "No entry fits the search, the filters and this view. Widen them, or clear the filters.",
            )
          : t("Entries appear here as people, agents, keys and jobs change records.")
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

/**
 * Every entry the reader can see, newest first, grouped by day: each row a sentence with the
 * fields it changed, opening beside the list. Quick views narrow it on the server; the person's
 * own filters, the chart's span and a picked actor live in its address.
 */
export default function AuditTrail({
  days,
  quick,
  onQuickChange,
  entryId,
  onEntryChange,
  canSeeReviews,
  onShowRecordHistory,
  headerActions,
}: {
  /** The overview's window: its rollups count each day's entries while nothing narrows the trail. */
  days: number;
  quick: AuditQuickView;
  onQuickChange: (view: AuditQuickView) => void;
  entryId: string | null;
  onEntryChange: (id: string | null) => void;
  canSeeReviews: boolean;
  onShowRecordHistory: (resourceId: string) => void;
  /** Controls the page puts at the end of the trail's header, such as folding the overview. */
  headerActions?: ReactNode;
}) {
  const t = useT();
  const timezone = useUserTimezone();
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [collapsed, setCollapsed] = useQueryState("collapsed", collapsedParser);
  const [onePerRequest, setOnePerRequest] = useLocalStorage(ONE_PER_REQUEST_STORAGE_KEY, true);
  const [filters] = useQueryStates(tableFilterSearchParamsParser);
  const { data: overview } = useAuditOverview(days);
  const offHoursZone = overview?.timezone ?? null;
  const switchId = useId();
  const narrowed =
    quick !== "all" ||
    Boolean(filters.query) ||
    (filters.fieldFilters?.length ?? 0) > 0 ||
    (filters.filterGroups?.length ?? 0) > 0;
  const grouped = onePerRequest && !narrowed;
  const graphql = useMemo(
    () => ({ ...auditLogTableGraphQLConfig, extraVariables: { onePerRequest: grouped } }),
    [grouped],
  );

  const columns = useMemo(
    () =>
      getTrailColumns(t, {
        offHoursZone,
        expandedId,
        onToggleRequest: (id) => setExpandedId((current) => (current === id ? null : id)),
      }),
    [t, offHoursZone, expandedId],
  );
  const scopeFilters = QUICK_VIEW_FILTERS[quick];
  const activity = narrowed ? undefined : overview?.activity;
  const groups = useMemo(
    () => dayGroups(t, timezone, activity ? entriesByDay(activity, timezone) : new Map()),
    [t, timezone, activity],
  );

  const views: SegmentedControlItem<AuditQuickView>[] = [
    { value: "all", label: t("All") },
    { value: "changes", label: t("Changes") },
    ...(canSeeReviews ? [{ value: "security" as const, label: t("Security") }] : []),
    { value: "agents", label: t("By agents") },
    { value: "deletes", label: t("Deletes and exports") },
  ];

  return (
    <div className="flex min-w-0 items-start gap-4">
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-base font-semibold">{t("Trail")}</h2>
          <div className="max-w-full overflow-x-auto">
            <SegmentedControl
              aria-label={t("Quick views")}
              items={views}
              value={quick}
              onValueChange={onQuickChange}
            />
          </div>
          <div className="ml-auto flex items-center gap-2">
            {onePerRequest && narrowed ? (
              <span className="text-foreground-subtle text-xs">
                {t("Every match is shown while you search or filter")}
              </span>
            ) : null}
            <Switch
              id={switchId}
              size="sm"
              checked={onePerRequest}
              onCheckedChange={(checked) => setOnePerRequest(checked)}
            />
            <label htmlFor={switchId} className="text-muted-foreground text-sm">
              {t("One row per request")}
            </label>
            {headerActions}
          </div>
        </div>
        <DataTable<AuditEntryRow>
          name="Audit Entry"
          queryKey="audit-entry-list"
          graphql={graphql}
          resource={Resource.AuditLog}
          columns={columns}
          scopeFilters={scopeFilters}
          enableCreateAction={false}
          enableExport={false}
          initialColumnVisibility={TRAIL_HIDDEN_COLUMNS}
          onRowClick={(row) => onEntryChange(row.original.id)}
          getRowClassName={(row) =>
            cn(row.original.id === entryId && "bg-surface-selected data-cursor:bg-surface-selected")
          }
          keyboard={{
            enabled: true,
            cursorRowId: entryId,
            onCursorRowIdChange: (id) => onEntryChange(id),
          }}
          grouping={{
            field: "timestamp",
            direction: "desc",
            tieBreakers: [{ field: "id", direction: "desc" }],
            groups,
            getGroupKey: (row) => dayKey(row.timestamp, timezone),
            collapsedKeys: collapsed,
            collapsedScope: (keys: readonly DataTableGroupKey[]) =>
              dayGroupScope("timestamp", keys, timezone),
            onToggleGroup: (key) => {
              const value = String(key);
              void setCollapsed(
                collapsed.includes(value)
                  ? collapsed.filter((entry) => entry !== value)
                  : [...collapsed, value],
              );
            },
          }}
          expansion={{
            expandedRowId: expandedId,
            onExpandedRowIdChange: setExpandedId,
            renderExpandedRow: (row) =>
              row.original.correlationId ? (
                <div className="border-l-border ml-28 border-l py-1 pl-3">
                  <RequestEntries
                    correlationId={row.original.correlationId}
                    entryId={row.original.id}
                    onOpen={onEntryChange}
                  />
                </div>
              ) : null,
          }}
          renderEmptyState={(state) => <TrailEmpty {...state} />}
        />
      </div>
      {entryId ? (
        <div className="sticky top-4 w-104 shrink-0">
          <AuditInspector
            key={entryId}
            entryId={entryId}
            canSeeReviews={canSeeReviews}
            offHoursZone={offHoursZone}
            onOpenEntry={onEntryChange}
            onShowRecordHistory={onShowRecordHistory}
            onClose={() => onEntryChange(null)}
          />
        </div>
      ) : null}
    </div>
  );
}
