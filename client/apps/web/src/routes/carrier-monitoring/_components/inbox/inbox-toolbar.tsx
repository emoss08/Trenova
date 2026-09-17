import {
  AddFilterMenu,
  FilterChip,
  type FilterDefinition,
} from "@/components/carrier-intelligence/filter-chip";
import { StatusDot, severityTone } from "@/components/carrier-intelligence/status-dot";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useSelectOption } from "@/hooks/use-select-option";
import { CARRIER_INTEL_SECTIONS, CARRIER_INTEL_SEVERITIES } from "@/lib/carrier-intelligence";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { SearchIcon, SlidersHorizontalIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  CARRIER_INTEL_EVENT_SOURCES,
  INBOX_GROUPINGS,
  type InboxFilterId,
  type InboxFilterState,
  type InboxGrouping,
  type InboxScope,
} from "./inbox-filters";

const SEARCH_DEBOUNCE_MS = 300;
const CARRIER_OPTION_LIMIT = 20;

export type InboxFilterPatch = Partial<{
  scope: InboxScope;
  q: string;
  severity: InboxFilterState["severity"];
  category: InboxFilterState["category"];
  source: InboxFilterState["source"];
  carrier: string | null;
}>;

export type InboxToolbarProps = {
  state: InboxFilterState;
  grouping: InboxGrouping;
  labels: CarrierIntelLabels;
  scopeCounts: Partial<Record<InboxScope, number>>;
  selectedCount: number;
  canUpdate: boolean;
  acknowledging: boolean;
  onChange: (patch: InboxFilterPatch) => void;
  onGroupingChange: (grouping: InboxGrouping) => void;
  onClearFilters: () => void;
  onAcknowledgeSelected: () => void;
  onClearSelection: () => void;
};

function summarizeValues(values: readonly string[]): string {
  if (values.length <= 2) {
    return values.join(", ");
  }
  return `${values.slice(0, 2).join(", ")} +${values.length - 2}`;
}

function useCarrierFilterOptions(enabled: boolean, search: string) {
  const query = useDebounce(search.trim(), SEARCH_DEBOUNCE_MS);
  return useQuery({
    queryKey: ["select-option", "CARRIER", "inbox-filter", query],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: "CARRIER", query: query || undefined, initialLimit: CARRIER_OPTION_LIMIT },
        { signal },
      ),
    enabled,
    staleTime: 60_000,
  });
}

export function InboxToolbar({
  state,
  grouping,
  labels,
  scopeCounts,
  selectedCount,
  canUpdate,
  acknowledging,
  onChange,
  onGroupingChange,
  onClearFilters,
  onAcknowledgeSelected,
  onClearSelection,
}: InboxToolbarProps) {
  const t = useT();
  const [search, setSearch] = useState(state.q);
  const debouncedSearch = useDebounce(search, SEARCH_DEBOUNCE_MS);
  const [menuOpen, setMenuOpen] = useState(false);
  const [editing, setEditing] = useState<InboxFilterId | null>(null);
  const [carrierSearch, setCarrierSearch] = useState("");

  const sentQuery = useRef(state.q);
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    if (state.q !== sentQuery.current) {
      sentQuery.current = state.q;
      setSearch(state.q);
    }
  }, [state.q]);

  useEffect(() => {
    if (debouncedSearch !== sentQuery.current) {
      sentQuery.current = debouncedSearch;
      onChangeRef.current({ q: debouncedSearch });
    }
  }, [debouncedSearch]);

  const carrierOptions = useCarrierFilterOptions(menuOpen, carrierSearch);
  const { option: selectedCarrier } = useSelectOption("CARRIER", state.carrier);

  const values = useMemo<Record<InboxFilterId, string[]>>(
    () => ({
      severity: [...state.severity],
      category: [...state.category],
      source: [...state.source],
      carrier: state.carrier ? [state.carrier] : [],
    }),
    [state.carrier, state.category, state.severity, state.source],
  );

  const definitions = useMemo<FilterDefinition[]>(
    () => [
      {
        id: "severity",
        label: t("Severity"),
        multiple: true,
        options: CARRIER_INTEL_SEVERITIES.map((severity) => ({
          value: severity,
          label: labels.severity[severity],
          keywords: labels.severity[severity],
          icon: <StatusDot tone={severityTone(severity)} />,
        })),
      },
      {
        id: "category",
        label: t("Category"),
        multiple: true,
        options: CARRIER_INTEL_SECTIONS.map((section) => ({
          value: section,
          label: labels.section[section],
          keywords: labels.section[section],
        })),
      },
      {
        id: "source",
        label: t("Source"),
        multiple: true,
        options: CARRIER_INTEL_EVENT_SOURCES.map((source) => ({
          value: source,
          label: labels.eventSource[source],
          keywords: labels.eventSource[source],
        })),
      },
      {
        id: "carrier",
        label: t("Carrier"),
        onSearchChange: setCarrierSearch,
        searching: carrierOptions.isFetching,
        options: (carrierOptions.data?.results ?? []).map((option) => ({
          value: option.id,
          label: option.label,
          keywords: option.label,
        })),
      },
    ],
    [carrierOptions.data, carrierOptions.isFetching, labels, t],
  );

  const applyFilter = (filterId: string, next: string[]) => {
    switch (filterId as InboxFilterId) {
      case "severity":
        onChange({ severity: next as InboxFilterState["severity"] });
        break;
      case "category":
        onChange({ category: next as InboxFilterState["category"] });
        break;
      case "source":
        onChange({ source: next as InboxFilterState["source"] });
        break;
      case "carrier":
        onChange({ carrier: next[0] ?? null });
        break;
    }
  };

  const chips: { id: InboxFilterId; label: string; value: ReactNode }[] = [];
  if (state.severity.length > 0) {
    chips.push({
      id: "severity",
      label: t("Severity"),
      value: summarizeValues(state.severity.map((value) => labels.severity[value])),
    });
  }
  if (state.category.length > 0) {
    chips.push({
      id: "category",
      label: t("Category"),
      value: summarizeValues(state.category.map((value) => labels.section[value])),
    });
  }
  if (state.source.length > 0) {
    chips.push({
      id: "source",
      label: t("Source"),
      value: summarizeValues(state.source.map((value) => labels.eventSource[value])),
    });
  }
  if (state.carrier) {
    chips.push({
      id: "carrier",
      label: t("Carrier"),
      value: selectedCarrier?.label ?? t("Selected carrier"),
    });
  }

  const hasFilters = chips.length > 0 || state.q.trim() !== "";

  const scopeItems = (
    [
      ["attention", t("Needs attention")],
      ["acknowledged", t("Acknowledged")],
      ["resolved", t("Resolved")],
      ["all", t("All")],
    ] as const
  ).map(([value, label]) => {
    const count = scopeCounts[value];
    return {
      value,
      label:
        count !== undefined && count > 0 ? (
          <span className="flex items-center gap-1.5">
            {label}
            <span className="text-muted-foreground tabular-nums">{count.toLocaleString()}</span>
          </span>
        ) : (
          label
        ),
    };
  });

  return (
    <div
      className="flex flex-wrap items-center gap-2"
      role="toolbar"
      aria-label={t("Inbox filters")}
    >
      <SegmentedControl<InboxScope>
        items={scopeItems}
        value={state.scope}
        onValueChange={(scope) => onChange({ scope })}
        aria-label={t("Event status")}
        className="h-8"
      />
      <Input
        type="search"
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        placeholder={t("Search carrier, USDOT or change")}
        aria-label={t("Search events")}
        leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
        inputContainerClassName="w-full sm:w-64"
        className="h-8 text-xs md:text-xs"
      />
      <AddFilterMenu
        key={editing ?? "all-filters"}
        filters={definitions}
        values={values}
        onChange={applyFilter}
        initialFilterId={editing}
        open={menuOpen}
        onOpenChange={(open) => {
          setMenuOpen(open);
          if (!open) {
            setEditing(null);
            setCarrierSearch("");
          }
        }}
        className="h-8 text-xs"
      />
      {chips.map((chip) => (
        <FilterChip
          key={chip.id}
          label={chip.label}
          value={chip.value}
          onEdit={() => {
            setEditing(chip.id);
            setMenuOpen(true);
          }}
          onClear={() => applyFilter(chip.id, [])}
        />
      ))}
      {hasFilters ? (
        <Button
          type="button"
          variant="ghost"
          className="text-muted-foreground h-8 text-xs"
          onClick={() => {
            setSearch("");
            onClearFilters();
          }}
        >
          {t("Clear")}
        </Button>
      ) : null}
      <div className="ml-auto flex items-center gap-2">
        {selectedCount > 0 ? (
          <div className="flex items-center gap-1" aria-live="polite">
            <span className="text-muted-foreground px-1 text-xs tabular-nums">
              {t("{0} selected", selectedCount)}
            </span>
            {canUpdate ? (
              <Button
                type="button"
                variant="outline"
                className="h-8 text-xs"
                isLoading={acknowledging}
                onClick={onAcknowledgeSelected}
              >
                {t("Acknowledge")}
                <Kbd>E</Kbd>
              </Button>
            ) : null}
            <Button
              type="button"
              variant="ghost"
              className="h-8 text-xs"
              onClick={onClearSelection}
            >
              {t("Clear selection")}
            </Button>
          </div>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button type="button" variant="ghost" size="icon" aria-label={t("Display options")} />
            }
          >
            <SlidersHorizontalIcon className="size-3.5" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuGroup>
              <DropdownMenuLabel>{t("Group by")}</DropdownMenuLabel>
              <DropdownMenuRadioGroup
                value={grouping}
                onValueChange={(value) => {
                  if ((INBOX_GROUPINGS as readonly string[]).includes(value as string)) {
                    onGroupingChange(value as InboxGrouping);
                  }
                }}
              >
                <DropdownMenuRadioItem value="day">{t("Day")}</DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="carrier">{t("Carrier")}</DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
