import { useT } from "@trenova/shared/i18n/use-t";
import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import type { TableDensity } from "@/types/table-configuration";
import { columnHeaderLabel } from "@/lib/data-table";
import type { Table } from "@trenova/shared/types/data-table";
import {
  Brush01Icon,
  ChevronSelectorHorizontalIcon,
  ChevronRightIcon,
  FlipBackwardIcon,
  Rows02Icon,
  Rows03Icon,
  Sliders01Icon,
} from "@trenova/shared/components/icons";
import { useState, type ReactNode } from "react";

type DataTableDisplayMenuProps = {
  table: Table<RowData>;
  density?: TableDensity;
  onDensityChange?: (density: TableDensity) => void;
  formatRuleCount?: number;
  onEditFormatRules?: () => void;
  /** Whether this person has arranged the table their own way, so there is a layout to forget. */
  hasSavedLayout?: boolean;
  onResetLayout?: () => void;
  /** Fits every column to its widest content on the page. */
  onFitColumns?: () => void;
  /** Whether rows changed since the person's last visit are marked. */
  highlightChanges?: boolean;
  onHighlightChangesChange?: (on: boolean) => void;
  /** Whether the totals row shows; offered only on a table that has totals. */
  showTotals?: boolean;
  onShowTotalsChange?: (on: boolean) => void;
  virtualized?: boolean;
  onVirtualizedChange?: (on: boolean) => void;
};

const DENSITY_OPTIONS: { value: TableDensity; label: string; icon: typeof Rows02Icon }[] = [
  { value: "comfortable", label: "Comfortable", icon: Rows02Icon },
  { value: "compact", label: "Compact", icon: Rows03Icon },
];

/** Past this many columns the list gets a search box. */
const COLUMN_SEARCH_THRESHOLD = 8;

function Section({ title, aside, children }: { title: string; aside?: ReactNode; children: ReactNode }) {
  return (
    <section className="border-border flex flex-col gap-2 border-t px-3 py-3 first:border-t-0">
      <div className="flex h-5 items-center justify-between gap-2">
        <h3 className="text-muted-foreground text-xs font-medium">{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  );
}

function SwitchRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (on: boolean) => void;
}) {
  return (
    <label className="hover:bg-muted/60 -mx-1.5 flex cursor-pointer items-center gap-3 rounded-md px-1.5 py-1.5">
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="text-sm">{label}</span>
        {description ? (
          <span className="text-muted-foreground text-xs">{description}</span>
        ) : null}
      </span>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </label>
  );
}

function ActionRow({
  icon: Icon,
  label,
  trailing,
  disabled,
  onClick,
}: {
  icon: typeof Rows02Icon;
  label: string;
  trailing?: ReactNode;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      variant="ghost"
      size="sm"
      disabled={disabled}
      className="-mx-1.5 h-8 w-[calc(100%+0.75rem)] justify-start px-1.5 font-normal"
      onClick={onClick}
    >
      <Icon className="text-muted-foreground size-3.5" />
      <span className="flex-1 text-left">{label}</span>
      {trailing}
    </Button>
  );
}

export default function DataTableDisplayMenu({
  table,
  density = "comfortable",
  onDensityChange,
  formatRuleCount = 0,
  onEditFormatRules,
  hasSavedLayout = false,
  onResetLayout,
  onFitColumns,
  highlightChanges = true,
  onHighlightChangesChange,
  showTotals = true,
  onShowTotalsChange,
  virtualized = false,
  onVirtualizedChange,
}: DataTableDisplayMenuProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const columns = table
    .getAllColumns()
    .filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide());
  const shownCount = columns.filter((column) => column.getIsVisible()).length;
  const needle = search.trim().toLowerCase();
  const labelOf = (column: (typeof columns)[number]) => columnHeaderLabel(column);
  const listed = needle
    ? columns.filter((column) => labelOf(column).toLowerCase().includes(needle))
    : columns;

  const hasRowOptions = !!(onShowTotalsChange || onHighlightChangesChange || onVirtualizedChange);
  const hasActions = !!(onEditFormatRules || onFitColumns || onResetLayout);

  if (!onDensityChange && !hasActions && columns.length === 0) {
    return null;
  }

  const runAndClose = (action: () => void) => () => {
    setOpen(false);
    action();
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setSearch("");
      }}
    >
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm">
            <Sliders01Icon className="size-3.5" />
            <span className="hidden lg:inline">{t("Display")}</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 gap-0 p-0">
        {onDensityChange ? (
          <Section title={t("Density")}>
            <SegmentedControl
              items={DENSITY_OPTIONS.map((option) => ({
                value: option.value,
                label: t(option.label),
                icon: option.icon,
              }))}
              value={density}
              onValueChange={onDensityChange}
              fullWidth
              aria-label={t("Row density")}
            />
          </Section>
        ) : null}

        {columns.length > 0 ? (
          <Section
            title={t("Columns")}
            aside={
              <span className="flex items-center gap-2 text-xs">
                <span className="text-muted-foreground tabular-nums">
                  {t("{0} of {1} shown", shownCount, columns.length)}
                </span>
                {shownCount < columns.length ? (
                  <button
                    type="button"
                    className="text-foreground ui-focus-ring cursor-pointer rounded-sm hover:underline"
                    onClick={() => table.toggleAllColumnsVisible(true)}
                  >
                    {t("Show all")}
                  </button>
                ) : null}
              </span>
            }
          >
            {columns.length > COLUMN_SEARCH_THRESHOLD ? (
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t("Find a column")}
                className="h-7"
              />
            ) : null}
            <ScrollArea viewportClassName="max-h-56" maskVariant="popover">
              <ul className="flex flex-col">
                {listed.map((column) => {
                  const isVisible = column.getIsVisible();
                  return (
                    <li key={column.id}>
                      <label className="hover:bg-muted/60 -mx-1.5 flex h-7 cursor-pointer items-center gap-2 rounded-md px-1.5 text-sm">
                        <Checkbox
                          checked={isVisible}
                          onCheckedChange={(checked) => column.toggleVisibility(checked === true)}
                        />
                        <span className={cn("truncate", !isVisible && "text-muted-foreground")}>
                          {labelOf(column)}
                        </span>
                      </label>
                    </li>
                  );
                })}
                {listed.length === 0 ? (
                  <li className="text-muted-foreground py-2 text-xs">{t("No column matches.")}</li>
                ) : null}
              </ul>
            </ScrollArea>
          </Section>
        ) : null}

        {hasRowOptions ? (
          <Section title={t("Rows")}>
            <div className="flex flex-col">
              {onShowTotalsChange ? (
                <SwitchRow
                  label={t("Show totals")}
                  checked={showTotals}
                  onCheckedChange={onShowTotalsChange}
                />
              ) : null}
              {onHighlightChangesChange ? (
                <SwitchRow
                  label={t("Mark changes since my last visit")}
                  checked={highlightChanges}
                  onCheckedChange={onHighlightChangesChange}
                />
              ) : null}
              {onVirtualizedChange ? (
                <SwitchRow
                  label={t("Draw only rows in view")}
                  description={t("Faster on long pages; find in page sees only drawn rows.")}
                  checked={virtualized}
                  onCheckedChange={onVirtualizedChange}
                />
              ) : null}
            </div>
          </Section>
        ) : null}

        {hasActions ? (
          <Section title={t("Layout")}>
            <div className="flex flex-col">
              {onEditFormatRules ? (
                <ActionRow
                  icon={Brush01Icon}
                  label={t("Conditional formatting")}
                  onClick={runAndClose(onEditFormatRules)}
                  trailing={
                    <span className="text-muted-foreground flex items-center gap-1">
                      {formatRuleCount > 0 ? (
                        <span className="bg-muted flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 text-xs tabular-nums">
                          {formatRuleCount}
                        </span>
                      ) : null}
                      <ChevronRightIcon className="size-3.5" />
                    </span>
                  }
                />
              ) : null}
              {onFitColumns ? (
                <ActionRow
                  icon={ChevronSelectorHorizontalIcon}
                  label={t("Fit columns to content")}
                  onClick={runAndClose(onFitColumns)}
                />
              ) : null}
              {onResetLayout ? (
                <ActionRow
                  icon={FlipBackwardIcon}
                  label={t("Reset to default layout")}
                  disabled={!hasSavedLayout}
                  onClick={runAndClose(onResetLayout)}
                />
              ) : null}
            </div>
          </Section>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
