"use no memo";
import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import type { TableDensity } from "@/types/table-configuration";
import type { Table } from "@trenova/shared/types/data-table";
import {
  ChevronRightIcon,
  PaintbrushIcon,
  Rows2Icon,
  Rows4Icon,
  SlidersHorizontalIcon,
} from "lucide-react";
import { useState } from "react";

type DataTableDisplayMenuProps = {
  table: Table<RowData>;
  density?: TableDensity;
  onDensityChange?: (density: TableDensity) => void;
  formatRuleCount?: number;
  onEditFormatRules?: () => void;
};

const DENSITY_OPTIONS: { value: TableDensity; label: string; icon: typeof Rows2Icon }[] = [
  { value: "comfortable", label: "Comfortable", icon: Rows2Icon },
  { value: "compact", label: "Compact", icon: Rows4Icon },
];

export default function DataTableDisplayMenu({
  table,
  density = "comfortable",
  onDensityChange,
  formatRuleCount = 0,
  onEditFormatRules,
}: DataTableDisplayMenuProps) {
  const [open, setOpen] = useState(false);

  const columns = table
    .getAllColumns()
    .filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide());
  const hiddenCount = columns.filter((column) => !column.getIsVisible()).length;

  if (!onDensityChange && !onEditFormatRules && columns.length === 0) {
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm">
            <SlidersHorizontalIcon className="size-3.5" />
            <span className="hidden lg:inline">Display</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="dark w-72 gap-0 p-0">
        <div className="flex flex-col gap-3 p-3">
          {onDensityChange && (
            <div className="flex flex-col gap-1.5">
              <span className="text-muted-foreground text-xs font-medium">Density</span>
              <div
                className="bg-muted grid grid-cols-2 gap-1 rounded-lg p-1"
                role="radiogroup"
                aria-label="Row density"
              >
                {DENSITY_OPTIONS.map((option) => {
                  const selected = density === option.value;
                  return (
                    <button
                      key={option.value}
                      type="button"
                      role="radio"
                      aria-checked={selected}
                      onClick={() => onDensityChange(option.value)}
                      className={cn(
                        "flex h-6.5 cursor-pointer items-center justify-center gap-1.5 rounded-md text-xs transition-colors",
                        selected
                          ? "bg-background text-foreground shadow-sm"
                          : "text-muted-foreground hover:text-foreground",
                      )}
                    >
                      <option.icon className="size-3.5" />
                      {option.label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
          {columns.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground text-xs font-medium">Columns</span>
                {hiddenCount > 0 && (
                  <button
                    type="button"
                    className="text-muted-foreground hover:text-foreground cursor-pointer text-xs transition-colors"
                    onClick={() => table.toggleAllColumnsVisible(true)}
                  >
                    Show all
                  </button>
                )}
              </div>
              <div className="flex max-h-44 flex-wrap content-start gap-1 overflow-y-auto">
                {columns.map((column) => {
                  const label = column.columnDef.meta?.label || column.id;
                  const isVisible = column.getIsVisible();
                  return (
                    <button
                      key={column.id}
                      type="button"
                      aria-pressed={isVisible}
                      onClick={() => column.toggleVisibility(!isVisible)}
                      className={cn(
                        "h-6 cursor-pointer rounded-md border px-2 text-xs transition-colors",
                        isVisible
                          ? "border-border bg-muted text-foreground hover:bg-muted/70"
                          : "border-border/70 text-muted-foreground hover:border-border hover:text-foreground border-dashed",
                      )}
                    >
                      {label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>
        {onEditFormatRules && (
          <Button
            variant="ghost"
            size="sm"
            className="border-border w-full justify-start rounded-t-none border-t font-normal"
            onClick={() => {
              setOpen(false);
              onEditFormatRules();
            }}
          >
            <PaintbrushIcon className="text-muted-foreground size-3.5" />
            Conditional formatting
            <span className="text-muted-foreground ml-auto flex items-center gap-1">
              {formatRuleCount > 0 && (
                <span className="bg-muted flex size-5 items-center justify-center rounded-md font-mono text-xs">
                  {formatRuleCount}
                </span>
              )}
              <ChevronRightIcon className="size-3.5" />
            </span>
          </Button>
        )}
      </PopoverContent>
    </Popover>
  );
}
