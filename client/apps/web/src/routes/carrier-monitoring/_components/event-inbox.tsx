import { useT } from "@trenova/shared/i18n/use-t";
import { ControlledCarrierAutocompleteField } from "@/components/autocomplete-fields";
import { ResolveEventDialog } from "@/components/carrier-intelligence/resolve-event-dialog";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { DataTable } from "@/components/data-table/data-table";
import { CARRIER_INTEL_SECTIONS, CARRIER_INTEL_SEVERITIES } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTELLIGENCE_KEY,
  CARRIER_INTEL_EVENTS_KEY,
} from "@/lib/graphql/carrier-intelligence";
import {
  CARRIER_INTEL_EVENT_LIST_KEY,
  carrierIntelEventTableGraphQLConfig,
  type CarrierIntelEventRow,
} from "@/lib/graphql/carrier-monitoring-table";
import { queries } from "@/lib/queries";
import type { CarrierIntelSection, CarrierIntelSeverity } from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTableEmptyStateRenderProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { FilterXIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import { getEventColumns } from "./event-columns";
import {
  buildEventFilter,
  eventInboxSearchParams,
  hasNarrowingEventFilters,
  type EventScope,
} from "./event-inbox-filter";
import { useEventInboxActions } from "./use-event-inbox-actions";

const ALL_CATEGORIES = "__all__";

const SEVERITY_TOGGLE_CLASSES: Record<CarrierIntelSeverity, string> = {
  Critical: "data-[pressed=true]:border-red-600 data-[pressed=true]:bg-red-600/10",
  High: "data-[pressed=true]:border-orange-500 data-[pressed=true]:bg-orange-500/10",
  Medium: "data-[pressed=true]:border-yellow-500 data-[pressed=true]:bg-yellow-500/10",
  Low: "data-[pressed=true]:border-blue-500 data-[pressed=true]:bg-blue-500/10",
  Info: "data-[pressed=true]:border-muted-foreground data-[pressed=true]:bg-muted",
};

export type EventInboxProps = {
  canUpdate: boolean;
};

export function EventInbox({ canUpdate }: EventInboxProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useQueryStates(eventInboxSearchParams);

  const filter = useMemo(
    () =>
      buildEventFilter({
        scope: filters.scope,
        severity: filters.severity,
        category: filters.category,
        carrier: filters.carrier,
      }),
    [filters.carrier, filters.category, filters.scope, filters.severity],
  );
  const graphql = useMemo(() => carrierIntelEventTableGraphQLConfig(filter), [filter]);
  const columns = useMemo(() => getEventColumns(t, labels), [labels, t]);
  const narrowed = hasNarrowingEventFilters(filters);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_EVENT_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_EVENTS_KEY] }),
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY] }),
      queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      }),
    ]);
  }, [queryClient]);

  const { dockActions, contextMenuActions, resolving, setResolving } = useEventInboxActions({
    canUpdate,
    onChanged: invalidate,
  });

  const resetFilters = useCallback(() => {
    void setFilters({ scope: null, severity: null, category: null, carrier: null });
  }, [setFilters]);

  const toggleSeverity = (severity: CarrierIntelSeverity) => {
    const next = filters.severity.includes(severity)
      ? filters.severity.filter((value) => value !== severity)
      : [...filters.severity, severity];
    void setFilters({ severity: next.length > 0 ? next : null });
  };

  const scopeItems: { value: EventScope; label: string }[] = [
    { value: "attention", label: t("Needs attention") },
    { value: "Open", label: labels.eventStatus.Open },
    { value: "Acknowledged", label: labels.eventStatus.Acknowledged },
    { value: "Resolved", label: labels.eventStatus.Resolved },
    { value: "Dismissed", label: labels.eventStatus.Dismissed },
    { value: "all", label: t("All") },
  ];

  const categoryItems = [
    { value: ALL_CATEGORIES, label: t("All categories") },
    ...CARRIER_INTEL_SECTIONS.map((section) => ({
      value: section,
      label: labels.section[section],
    })),
  ];

  const renderEmptyState = ({
    hasActiveFilters,
    onClearFilters,
  }: DataTableEmptyStateRenderProps) => {
    const filtered = hasActiveFilters || narrowed;
    return (
      <EmptyTable
        className="py-10"
        title={
          filtered
            ? t("Nothing matches")
            : filters.scope === "attention"
              ? t("Nothing needs attention")
              : t("No carrier changes yet")
        }
        description={
          filtered
            ? t(
                "No carrier change fits these filters. Widen them, or clear them to see every event.",
              )
            : t(
                "Authority, insurance and safety changes on monitored carriers land here as they are detected. Acknowledge them to show someone is on it, and resolve them once handled.",
              )
        }
        columns={[
          { label: t("Severity") },
          { label: t("Status") },
          { label: t("Carrier") },
          { label: t("Change") },
          { label: t("Detected") },
        ]}
        onClearFilters={
          filtered
            ? () => {
                onClearFilters();
                resetFilters();
              }
            : undefined
        }
      />
    );
  };

  return (
    <div className="flex flex-col gap-3">
      <div
        className="flex flex-wrap items-end gap-x-4 gap-y-2"
        role="group"
        aria-label={t("Event filters")}
      >
        <SegmentedControl<EventScope>
          items={scopeItems}
          value={filters.scope}
          onValueChange={(scope) =>
            void setFilters({ scope: scope === "attention" ? null : scope })
          }
          aria-label={t("Event status")}
        />
        <div className="flex items-center gap-1" role="group" aria-label={t("Severity")}>
          {CARRIER_INTEL_SEVERITIES.map((severity) => {
            const pressed = filters.severity.includes(severity);
            return (
              <Button
                key={severity}
                type="button"
                size="xs"
                variant="outline"
                aria-pressed={pressed}
                data-pressed={pressed}
                className={cn(SEVERITY_TOGGLE_CLASSES[severity])}
                onClick={() => toggleSeverity(severity)}
              >
                {labels.severity[severity]}
              </Button>
            );
          })}
        </div>
        <Select
          items={categoryItems}
          value={filters.category ?? ALL_CATEGORIES}
          onValueChange={(value) =>
            void setFilters({
              category: value && value !== ALL_CATEGORIES ? (value as CarrierIntelSection) : null,
            })
          }
        >
          <SelectTrigger size="sm" className="min-w-40" aria-label={t("Category")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {categoryItems.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="w-64">
          <ControlledCarrierAutocompleteField
            label={t("Carrier")}
            value={filters.carrier ?? ""}
            onValueChange={(value) => void setFilters({ carrier: value || null })}
            placeholder={t("Any carrier")}
          />
        </div>
        {narrowed ? (
          <Button type="button" size="sm" variant="ghost" onClick={resetFilters}>
            <FilterXIcon className="size-3.5" />
            {t("Reset")}
          </Button>
        ) : null}
      </div>
      <DataTable<CarrierIntelEventRow>
        name="Carrier Intelligence Event"
        queryKey={CARRIER_INTEL_EVENT_LIST_KEY}
        resource={Resource.CarrierIntelligence}
        columns={columns}
        graphql={graphql}
        enableCreateAction={false}
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        renderEmptyState={renderEmptyState}
      />
      <ResolveEventDialog
        event={resolving}
        open={resolving !== null}
        onOpenChange={(open) => {
          if (!open) {
            setResolving(null);
          }
        }}
        onResolved={() => void invalidate()}
      />
    </div>
  );
}
