import { useT } from "@trenova/shared/i18n/use-t";
import { useDataTableRowActions, useDataTableRowState } from "@/contexts/data-table-row-context";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { cn } from "@trenova/shared/lib/utils";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ChevronDownIcon, DotsHorizontalIcon } from "@trenova/shared/components/icons";
import { Fragment } from "react";

type ActionsCellProps = {
  row: Row<Shipment>;
  onToggleExpanded: () => void;
};

export function ActionsCell({ row, onToggleExpanded }: ActionsCellProps) {
  const t = useT();
  const actions = useDataTableRowActions<Shipment>();
  const { isExpanded: expanded } = useDataTableRowState();
  const visible = actions.filter((action) => !action.hidden?.(row));
  const standard = visible.filter((action) => action.variant !== "destructive");
  const destructive = visible.filter((action) => action.variant === "destructive");

  const renderItem = (action: RowAction<Shipment>) => {
    const Icon = action.icon;
    const disabled = action.disabled?.(row) ?? false;
    return (
      <DropdownMenuItem
        key={action.id}
        title={t(action.label)}
        color={action.variant === "destructive" ? "danger" : undefined}
        disabled={disabled}
        startContent={Icon ? <Icon className="size-3.5" /> : undefined}
        endContent={
          action.shortcut ? (
            <DropdownMenuShortcut>{action.shortcut}</DropdownMenuShortcut>
          ) : undefined
        }
        onClick={(event) => {
          event.stopPropagation();
          if (!disabled) void action.onClick(row);
        }}
      />
    );
  };

  return (
    <div
      className="flex items-center justify-end gap-0.5"
      onClick={(event) => event.stopPropagation()}
    >
      <Button
        variant="ghost"
        size="icon-xs"
        aria-label={expanded ? t("Collapse row") : t("Expand row")}
        aria-expanded={expanded}
        onClick={onToggleExpanded}
      >
        <ChevronDownIcon
          className={cn("size-3.5 transition-transform duration-200", expanded && "rotate-180")}
        />
      </Button>
      {visible.length > 0 ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={t("Row actions")}
                className="opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 data-popup-open:opacity-100"
              >
                <DotsHorizontalIcon className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" sideOffset={4} className="min-w-52">
            {standard.map(renderItem)}
            {destructive.length > 0 && standard.length > 0 ? <DropdownMenuSeparator /> : null}
            {destructive.map((action) => (
              <Fragment key={action.id}>{renderItem(action)}</Fragment>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null}
    </div>
  );
}
