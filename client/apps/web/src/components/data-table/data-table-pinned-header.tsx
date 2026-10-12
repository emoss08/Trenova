import { ChevronDownIcon, Pin01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

type DataTablePinnedHeaderProps = {
  count: number;
  collapsed: boolean;
  colSpan: number;
  onToggle: () => void;
  onUnpinAll: () => void;
};

/**
 * Heads the rows a person keeps at the top of the table. It sticks under the
 * column header like a group's header, and its label sticks to the left edge.
 */
export function DataTablePinnedHeader({
  count,
  collapsed,
  colSpan,
  onToggle,
  onUnpinAll,
}: DataTablePinnedHeaderProps) {
  const t = useT();

  return (
    <TableRow
      data-pinned-header
      data-collapsed={collapsed || undefined}
      className="hover:bg-transparent"
    >
      <TableCell
        colSpan={colSpan}
        className="bg-canvas border-border sticky top-(--row-head-h) z-15 h-9 cursor-pointer border-b p-0"
        onClick={onToggle}
      >
        <div className="sticky left-0 flex w-(--dt-viewport-w,100%) items-center gap-2 px-3">
          <button
            type="button"
            aria-expanded={!collapsed}
            aria-label={collapsed ? t("Show pinned rows") : t("Hide pinned rows")}
            onClick={(event) => {
              event.stopPropagation();
              onToggle();
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
          <Pin01Icon aria-hidden className="text-muted-foreground size-3.5 -rotate-45" />
          <span className="text-foreground text-sm font-medium">{t("Pinned")}</span>
          <span className="text-muted-foreground font-mono text-xs tabular-nums">
            {count.toLocaleString()}
          </span>
          <span className="flex-1" />
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="text-muted-foreground"
            onClick={(event) => {
              event.stopPropagation();
              onUnpinAll();
            }}
          >
            {t("Unpin all")}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
