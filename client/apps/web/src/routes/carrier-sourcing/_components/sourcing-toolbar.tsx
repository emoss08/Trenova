import {
  AddFilterMenu,
  FilterChip,
  type FilterDefinition,
} from "@/components/carrier-intelligence/filter-chip";
import {
  AUTHORITY_AGE_PRESETS,
  POWER_UNIT_RANGES,
  SOURCING_SCREENS,
  SOURCING_SORTS,
  type AuthorityAgePreset,
  type PowerUnitRange,
  type SourcingFilters,
  type SourcingScreen,
} from "@/lib/carrier-sourcing";
import { usStateAbbreviationChoices } from "@/lib/choices";
import type { CarrierSourcingSort } from "@trenova/graphql/generated/graphql";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowDownWideNarrowIcon } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { useSourcingLabels } from "./use-sourcing-labels";

type FilterId =
  | "state"
  | "originState"
  | "destinationState"
  | "powerUnits"
  | "authorityAge"
  | "screens";

export type SourcingToolbarProps = {
  filters: SourcingFilters;
  onFiltersChange: (filters: SourcingFilters) => void;
  sort: CarrierSourcingSort;
  onSortChange: (sort: CarrierSourcingSort) => void;
  summary: ReactNode;
  showFilters: boolean;
};

function isOneOf<T extends string>(values: readonly T[], value: string | undefined): value is T {
  return value !== undefined && (values as readonly string[]).includes(value);
}

export function SourcingToolbar({
  filters,
  onFiltersChange,
  sort,
  onSortChange,
  summary,
  showFilters,
}: SourcingToolbarProps) {
  const t = useT();
  const labels = useSourcingLabels();
  const [menuOpen, setMenuOpen] = useState(false);
  const [editing, setEditing] = useState<FilterId | null>(null);

  const definitions = useMemo<FilterDefinition[]>(() => {
    const states = usStateAbbreviationChoices.map((choice) => ({
      value: choice.value,
      label: choice.label,
      keywords: choice.label,
    }));
    return [
      { id: "state", label: t("Home state"), options: states },
      { id: "originState", label: t("Lane origin"), options: states },
      { id: "destinationState", label: t("Lane destination"), options: states },
      {
        id: "powerUnits",
        label: t("Power units"),
        options: POWER_UNIT_RANGES.map((range) => ({
          value: range,
          label: labels.powerUnits[range],
        })),
      },
      {
        id: "authorityAge",
        label: t("Authority age"),
        options: AUTHORITY_AGE_PRESETS.map((preset) => ({
          value: preset,
          label: labels.authorityAge[preset],
        })),
      },
      {
        id: "screens",
        label: t("Screening"),
        multiple: true,
        options: SOURCING_SCREENS.map((screen) => ({
          value: screen,
          label: labels.screen[screen],
        })),
      },
    ];
  }, [labels, t]);

  const values: Record<FilterId, string[]> = {
    state: filters.state ? [filters.state] : [],
    originState: filters.originState ? [filters.originState] : [],
    destinationState: filters.destinationState ? [filters.destinationState] : [],
    powerUnits: filters.powerUnits ? [filters.powerUnits] : [],
    authorityAge: filters.authorityAge ? [filters.authorityAge] : [],
    screens: [...filters.screens],
  };

  const handleChange = (filterId: string, next: string[]) => {
    const first = next[0];
    switch (filterId as FilterId) {
      case "state":
      case "originState":
      case "destinationState":
        onFiltersChange({ ...filters, [filterId]: first ?? null });
        break;
      case "powerUnits":
        onFiltersChange({
          ...filters,
          powerUnits: isOneOf<PowerUnitRange>(POWER_UNIT_RANGES, first) ? first : null,
        });
        break;
      case "authorityAge":
        onFiltersChange({
          ...filters,
          authorityAge: isOneOf<AuthorityAgePreset>(AUTHORITY_AGE_PRESETS, first) ? first : null,
        });
        break;
      case "screens":
        onFiltersChange({
          ...filters,
          screens: SOURCING_SCREENS.filter((screen: SourcingScreen) => next.includes(screen)),
        });
        break;
      default:
        break;
    }
  };

  const edit = (filterId: FilterId) => {
    setEditing(filterId);
    setMenuOpen(true);
  };

  const any = t("Any");

  return (
    <div className="flex flex-wrap items-center gap-2">
      {showFilters ? (
        <>
          <AddFilterMenu
            key={editing ?? "root"}
            filters={definitions}
            values={values}
            onChange={handleChange}
            initialFilterId={editing}
            open={menuOpen}
            onOpenChange={(open) => {
              setMenuOpen(open);
              if (!open) {
                setEditing(null);
              }
            }}
          />
          {filters.state ? (
            <FilterChip
              label={t("Home state")}
              value={filters.state}
              onEdit={() => edit("state")}
              onClear={() => onFiltersChange({ ...filters, state: null })}
            />
          ) : null}
          {filters.originState || filters.destinationState ? (
            <FilterChip
              label={t("Lane")}
              value={`${filters.originState ?? any} → ${filters.destinationState ?? any}`}
              onEdit={() => edit(filters.originState ? "originState" : "destinationState")}
              onClear={() =>
                onFiltersChange({ ...filters, originState: null, destinationState: null })
              }
            />
          ) : null}
          {filters.powerUnits ? (
            <FilterChip
              label={t("Fleet")}
              value={labels.powerUnits[filters.powerUnits]}
              onEdit={() => edit("powerUnits")}
              onClear={() => onFiltersChange({ ...filters, powerUnits: null })}
            />
          ) : null}
          {filters.authorityAge ? (
            <FilterChip
              label={t("Authority")}
              value={labels.authorityAge[filters.authorityAge]}
              onEdit={() => edit("authorityAge")}
              onClear={() => onFiltersChange({ ...filters, authorityAge: null })}
            />
          ) : null}
          {filters.screens.map((screen) => (
            <FilterChip
              key={screen}
              label={t("Screening")}
              value={labels.screen[screen]}
              onEdit={() => edit("screens")}
              onClear={() =>
                onFiltersChange({
                  ...filters,
                  screens: filters.screens.filter((current) => current !== screen),
                })
              }
            />
          ))}
        </>
      ) : null}
      <div className="ml-auto flex items-center gap-2">
        <span className="text-muted-foreground text-xs tabular-nums" aria-live="polite">
          {summary}
        </span>
        {showFilters ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" className="gap-1.5" aria-label={t("Sort carriers")} />
              }
            >
              <ArrowDownWideNarrowIcon className="size-3.5" aria-hidden />
              {labels.sort[sort]}
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuGroup>
                <DropdownMenuLabel>{t("Sort by")}</DropdownMenuLabel>
                <DropdownMenuRadioGroup
                  value={sort}
                  onValueChange={(next: string) => {
                    if (isOneOf<CarrierSourcingSort>(SOURCING_SORTS, next)) {
                      onSortChange(next);
                    }
                  }}
                >
                  {SOURCING_SORTS.map((option) => (
                    <DropdownMenuRadioItem key={option} value={option}>
                      {labels.sort[option]}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </div>
    </div>
  );
}
