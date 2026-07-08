import { Button } from "@/components/ui/button";
import type { RowAction } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import type { Row } from "@tanstack/react-table";
import { CommentBlock } from "./comment-stack";

const QUICK_ACTION_IDS = new Set([
  "edit",
  "send-edi-load-tender",
  "transfer-ownership",
  "transfer-to-billing",
  "cancel",
  "uncancel",
]);

export function QuickActionsBlock({
  row,
  actions,
}: {
  row: Row<Shipment>;
  actions: RowAction<Shipment>[];
}) {
  const visibleActions = actions.filter(
    (action) => QUICK_ACTION_IDS.has(action.id) && !action.hidden?.(row),
  );

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1.5">
        {visibleActions.map((action) => {
          const Icon = action.icon;
          const disabled = action.disabled?.(row) ?? false;

          return (
            <Button
              key={action.id}
              type="button"
              variant={action.variant === "destructive" ? "destructive" : "outline"}
              size="xs"
              className="justify-start"
              disabled={disabled}
              onClick={(event) => {
                event.stopPropagation();
                if (!disabled) action.onClick(row);
              }}
            >
              {Icon && <Icon className="size-3" />}
              {action.label}
            </Button>
          );
        })}
      </div>
      <CommentBlock shipmentId={row.original.id} />
    </div>
  );
}

export default QuickActionsBlock;
