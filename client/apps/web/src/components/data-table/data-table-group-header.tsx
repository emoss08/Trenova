import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTableGroup, DataTableGroupKey } from "@trenova/shared/types/data-table";

type DataTableGroupHeaderProps = {
  group: DataTableGroup;
  collapsed: boolean;
  colSpan: number;
  onToggle: (key: DataTableGroupKey) => void;
};

/**
 * A group's header row. It sticks under the column header while its rows
 * scroll beneath it, and the label sticks to the left edge so a wide table
 * never scrolls it out of view.
 */
export function DataTableGroupHeader({
  group,
  collapsed,
  colSpan,
  onToggle,
}: DataTableGroupHeaderProps) {
  const t = useT();
  const toggle = () => onToggle(group.key);

  return (
    <TableRow
      data-group-key={group.key}
      data-collapsed={collapsed || undefined}
      className="hover:bg-transparent"
    >
      <TableCell
        colSpan={colSpan}
        className="bg-canvas border-border sticky top-(--row-head-h) z-15 h-9 cursor-pointer border-b p-0"
        onClick={toggle}
      >
        <div className="sticky left-0 flex w-(--dt-viewport-w,100%) items-center gap-2 px-3">
          <button
            type="button"
            aria-expanded={!collapsed}
            aria-label={collapsed ? t("Expand {0}", group.label) : t("Collapse {0}", group.label)}
            onClick={(event) => {
              event.stopPropagation();
              toggle();
            }}
            className="ui-focus-ring text-muted-foreground grid size-5 place-items-center rounded-sm"
          >
            <ChevronDownIcon
              className={cn(
                "size-3.5 transition-transform duration-300 ease-(--ease-settle)",
                collapsed && "-rotate-90",
              )}
            />
          </button>
          {group.swatchClassName ? (
            <span aria-hidden className={cn("size-2 shrink-0 rounded-xs", group.swatchClassName)} />
          ) : null}
          <span className="text-foreground text-sm font-medium">{group.label}</span>
          {group.count != null ? (
            <span className="text-muted-foreground font-mono text-xs tabular-nums">
              {group.count.toLocaleString()}
            </span>
          ) : null}
          <span className="flex-1" />
          {group.aggregate != null ? (
            <span className="flex items-baseline gap-1">
              {group.aggregateLabel ? (
                <span className="text-muted-foreground text-xs">{group.aggregateLabel}:</span>
              ) : null}
              <span className="text-foreground font-sans text-sm font-medium tabular-nums">
                {group.aggregate}
              </span>
            </span>
          ) : null}
        </div>
      </TableCell>
    </TableRow>
  );
}
