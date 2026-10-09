import type { FilterableField } from "@/lib/data-table";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { FilterFunnel01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTableQueryOptions } from "@trenova/shared/types/data-table";
import { useState } from "react";
import { useRecordNames } from "./record-filter-input";

/** How many values a header lists; the server counts the most common first. */
const FACET_LIMIT = 50;

/** What every header needs to count and filter its column's values. */
export type DataTableHeaderFacets = {
  resource: string;
  /** The fields the server can count for this table. */
  facetable: ReadonlySet<string>;
  /** The filters the table is showing. */
  queryOptions: DataTableQueryOptions;
  /** The table's own list options beyond filters. */
  options: Record<string, unknown>;
  /** The values a field is filtered to now, from its "is any of" filter. */
  activeValues: (apiField: string) => readonly string[];
  apply: (field: FilterableField, values: readonly string[]) => void;
};

type DataTableHeaderFacetFilterProps = {
  field: FilterableField;
  facets: DataTableHeaderFacets;
};

/**
 * Filters a column by its values from its header, each with how many of the rows the
 * other filters leave hold it. Counting ignores this column's own filter, so ticking a
 * value never hides the others from the list.
 */
export function DataTableHeaderFacetFilter({ field, facets }: DataTableHeaderFacetFilterProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const active = facets.activeValues(field.apiField);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("Filter {0} by value", field.label)}
            data-active={active.length > 0 || undefined}
            className={cn(
              "text-muted-foreground size-5 opacity-0 group-hover/head:opacity-100 focus-visible:opacity-100",
              "data-active:text-brand data-active:opacity-100 data-popup-open:opacity-100",
            )}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <FilterFunnel01Icon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent align="start" className="w-72 p-0">
        {open ? <FacetList field={field} facets={facets} active={active} /> : null}
      </PopoverContent>
    </Popover>
  );
}

function FacetList({
  field,
  facets,
  active,
}: {
  field: FilterableField;
  facets: DataTableHeaderFacets;
  active: readonly string[];
}) {
  const t = useT();
  const [search, setSearch] = useState("");

  const otherFilters = (facets.queryOptions.fieldFilters ?? []).filter(
    (filter) => filter.field !== field.apiField,
  );
  const { data, isPending, isError } = useQuery({
    ...queries.dataTableInsight.facets({
      resource: facets.resource,
      field: field.apiField,
      limit: FACET_LIMIT,
      filter: {
        query: facets.queryOptions.query || undefined,
        fieldFilters: otherFilters,
        filterGroups: facets.queryOptions.filterGroups ?? [],
      },
      options: facets.options,
    }),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
  });

  const buckets = data?.dataTableFacets.buckets ?? [];
  const ids = field.filterType === "record" ? buckets.flatMap((bucket) => bucket.value ?? []) : [];
  const { names } = useRecordNames(field.filterRecord, ids);

  const labelOf = (value: string | null | undefined): string => {
    if (value === null || value === undefined) return t("Empty");
    if (field.filterType === "record") return names.get(value) ?? value;
    const option = field.filterOptions?.find((entry) => String(entry.value) === value);
    return option ? t(option.label) : value;
  };

  const needle = search.trim().toLowerCase();
  const shown = needle
    ? buckets.filter((bucket) => labelOf(bucket.value).toLowerCase().includes(needle))
    : buckets;
  const activeSet = new Set(active);

  const toggle = (value: string) => {
    const next = activeSet.has(value)
      ? active.filter((entry) => entry !== value)
      : [...active, value];
    facets.apply(field, next);
  };

  return (
    <div className="flex flex-col">
      <div className="border-border border-b p-2">
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("Search {0}", field.label)}
          className="h-7"
          autoFocus
        />
      </div>
      <ScrollArea viewportClassName="max-h-72" maskVariant="popover">
        <div className="flex flex-col p-1" role="listbox" aria-multiselectable>
          {isPending ? (
            Array.from({ length: 5 }, (_, index) => (
              <div key={index} className="flex items-center gap-2 px-2 py-1.5">
                <Skeleton className="size-4 rounded-sm" />
                <Skeleton className="h-3 flex-1" />
              </div>
            ))
          ) : isError ? (
            <p className="text-muted-foreground px-2 py-3 text-sm">
              {t("The values could not be counted.")}
            </p>
          ) : shown.length === 0 ? (
            <p className="text-muted-foreground px-2 py-3 text-sm">{t("No values match.")}</p>
          ) : (
            shown.map((bucket) => {
              const value = bucket.value;
              const selectable = value !== null && value !== undefined;
              const checked = selectable && activeSet.has(value);
              return (
                <label
                  key={value ?? "__empty__"}
                  role="option"
                  aria-selected={checked}
                  aria-disabled={!selectable || undefined}
                  className={cn(
                    "hover:bg-muted flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm",
                    selectable ? "cursor-pointer" : "text-muted-foreground cursor-default",
                  )}
                >
                  <Checkbox
                    checked={checked}
                    disabled={!selectable}
                    onCheckedChange={() => selectable && toggle(value)}
                  />
                  <span className="min-w-0 flex-1 truncate">{labelOf(value)}</span>
                  <span className="text-muted-foreground font-mono text-xs tabular-nums">
                    {bucket.count.toLocaleString()}
                  </span>
                </label>
              );
            })
          )}
        </div>
      </ScrollArea>
      <div className="border-border flex items-center justify-between border-t px-2 py-1.5">
        <span className="text-muted-foreground text-xs tabular-nums">
          {data
            ? t(
                "{0, plural, one {# row} other {# rows}}",
                data.dataTableFacets.total,
              )
            : null}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          disabled={active.length === 0}
          onClick={() => facets.apply(field, [])}
        >
          {t("Clear")}
        </Button>
      </div>
    </div>
  );
}
