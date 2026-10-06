import { useT } from "@trenova/shared/i18n/use-t";
import { FilterLinesIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { FilterItem } from "@trenova/shared/types/data-table";
import {
  activeFacetCount,
  clearFacets,
  facetSelection,
  toggleFacetValue,
  type DataTableFacet,
} from "@/lib/data-table-facets";

type DataTableFacetFilterProps = {
  facets: DataTableFacet[];
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  isLoading?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** Shows the label beside the icon; a narrow toolbar shows the icon alone. */
  labelClassName?: string;
};

/**
 * Filters a table by ticking values of a few known fields, each with how many
 * rows carry it. The choices are ordinary `in` filters, so the chips, the
 * builder and saved views see them like any other.
 */
export function DataTableFacetFilter({
  facets,
  filters,
  onFiltersChange,
  isLoading = false,
  onOpenChange,
  labelClassName,
}: DataTableFacetFilterProps) {
  const t = useT();
  const active = activeFacetCount(filters, facets);

  return (
    <Popover onOpenChange={onOpenChange}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            data-active={active > 0 || undefined}
            className="data-active:bg-brand-subtle data-active:text-brand-subtle-foreground"
          >
            <FilterLinesIcon className="size-3.5" />
            <span className={labelClassName}>{t("Filter")}</span>
            {active > 0 ? (
              <span className="bg-brand text-foreground-on-solid rounded-full px-1.5 font-mono text-2xs tabular-nums">
                {active}
              </span>
            ) : null}
          </Button>
        }
      />
      <PopoverContent align="start" className="w-72 gap-0 p-0">
        <div className="max-h-96 overflow-y-auto py-1">
          {isLoading && facets.length === 0 ? (
            <div className="flex flex-col gap-2 p-3">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-32" />
            </div>
          ) : (
            facets.map((facet) => {
              const selected = facetSelection(filters, facet.field);
              return (
                <div key={facet.key} className="flex flex-col py-1">
                  <div className="text-muted-foreground px-3 py-1 text-xs font-medium">
                    {facet.label}
                  </div>
                  {facet.values.length === 0 ? (
                    <div className="text-muted-foreground px-3 py-1 text-xs">{t("No values")}</div>
                  ) : (
                    facet.values.map((value) => {
                      const checked = selected.includes(value.value);
                      return (
                        <label
                          key={value.value}
                          className={cn(
                            "hover:bg-surface-hover flex h-7 cursor-pointer items-center gap-2 px-3 text-sm",
                            checked && "text-foreground",
                          )}
                        >
                          <Checkbox
                            checked={checked}
                            onCheckedChange={() =>
                              onFiltersChange(toggleFacetValue(filters, facet, value.value))
                            }
                          />
                          <span className="min-w-0 flex-1 truncate">{value.label}</span>
                          <span className="text-muted-foreground font-mono text-xs tabular-nums">
                            {value.count.toLocaleString()}
                          </span>
                        </label>
                      );
                    })
                  )}
                </div>
              );
            })
          )}
        </div>
        <div className="border-border flex items-center justify-between border-t px-2 py-1.5">
          <Button
            variant="ghost"
            size="xs"
            disabled={active === 0}
            onClick={() => onFiltersChange(clearFacets(filters, facets))}
          >
            {t("Clear all")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
