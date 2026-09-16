import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import {
  CATEGORY_FILTERS,
  DEFAULT_FILTERS,
  SEVERITY_FILTERS,
  STATUS_FILTERS,
  hasActiveFilters,
  setStatusFilter,
  toggleFilter,
  type InsightFilterState,
} from "./insight-filters";
import {
  CATEGORY_DESCRIPTIONS,
  CATEGORY_LABELS,
  STATUS_DESCRIPTIONS,
  STATUS_LABELS,
} from "./insight-labels";

/**
 * The filters, as toggles rather than a form.
 *
 * Every control applies immediately and writes to the URL, so the view a person
 * is looking at is always the view they can send to someone else. There is no
 * apply button because there is nothing to batch: each filter is one field.
 */
export function InsightFilterBar({
  filters,
  total,
  onChange,
}: {
  filters: InsightFilterState;
  total: number;
  onChange: (next: InsightFilterState) => void;
}) {
  const t = useT();

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <SegmentedControl
          value={filters.status}
          onValueChange={(value) =>
            onChange(setStatusFilter(filters, value as InsightFilterState["status"]))
          }
          items={STATUS_FILTERS.map((status) => ({
            value: status,
            label: t(STATUS_LABELS[status]),
            caption: t(STATUS_DESCRIPTIONS[status]),
          }))}
        />

        <span className="text-muted-foreground text-xs tabular-nums">
          {total === 1 ? t("{0} finding", String(total)) : t("{0} findings", String(total))}
        </span>

        {hasActiveFilters(filters) && (
          <Button
            variant="ghost"
            size="xs"
            onClick={() => onChange(DEFAULT_FILTERS)}
            className="text-muted-foreground"
          >
            <XIcon className="size-3" />
            {t("Clear filters")}
          </Button>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-2xs text-muted-foreground mr-1">{t("Area")}</span>
        {CATEGORY_FILTERS.map((category) => (
          <FilterChip
            key={category}
            label={t(CATEGORY_LABELS[category])}
            tooltip={t(CATEGORY_DESCRIPTIONS[category])}
            selected={filters.categories.includes(category)}
            onToggle={() => onChange(toggleFilter(filters, "categories", category))}
          />
        ))}

        <span className="text-2xs text-muted-foreground mr-1 ml-3">{t("Urgency")}</span>
        {SEVERITY_FILTERS.map((severity) => (
          <FilterChip
            key={severity}
            label={severity}
            selected={filters.severities.includes(severity)}
            onToggle={() => onChange(toggleFilter(filters, "severities", severity))}
          />
        ))}
      </div>
    </div>
  );
}

function FilterChip({
  label,
  tooltip,
  selected,
  onToggle,
}: {
  label: string;
  tooltip?: string;
  selected: boolean;
  onToggle: () => void;
}) {
  const chip = (
    <Badge
      // aria-pressed rather than a checkbox role: this is a toggle button that
      // narrows a list, and a screen reader should hear whether it is on.
      render={
        <button type="button" aria-pressed={selected} onClick={onToggle}>
          {label}
        </button>
      }
      variant={selected ? "default" : "outline"}
      className={cn("cursor-pointer transition-colors", !selected && "hover:bg-muted")}
    />
  );

  if (!tooltip) {
    return chip;
  }

  return (
    <Tooltip>
      <TooltipTrigger render={chip} />
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}
