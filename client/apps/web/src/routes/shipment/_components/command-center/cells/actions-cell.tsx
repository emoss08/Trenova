import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { RowAction } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { Row } from "@tanstack/react-table";
import { MoreHorizontalIcon } from "lucide-react";

export function ActionsCell({
  row,
  actions,
}: {
  row: Row<Shipment>;
  actions: RowAction<Shipment>[];
}) {
  const visibleActions = actions.filter((a) => !a.hidden?.(row));
  if (visibleActions.length === 0) return null;

  return (
    <div onClick={(e) => e.stopPropagation()} onMouseEnter={(e) => e.stopPropagation()}>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="icon-xs" aria-label="Row actions">
              <MoreHorizontalIcon className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" sideOffset={4} className="min-w-48">
          {visibleActions.map((action) => {
            const Icon = action.icon;
            const disabled = action.disabled?.(row) ?? false;
            return (
              <DropdownMenuItem
                key={action.id}
                title={action.label}
                color={action.variant === "destructive" ? "danger" : undefined}
                disabled={disabled}
                startContent={Icon ? <Icon className="size-3.5" /> : undefined}
                onClick={(e) => {
                  e.stopPropagation();
                  if (!disabled) action.onClick(row);
                }}
              />
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
