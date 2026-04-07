import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { billingQueueStatusChoices } from "@/lib/choices";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { cn, formatCurrency } from "@/lib/utils";
import type { BillingQueueItem, BillingQueueStatus } from "@/types/billing-queue";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ExternalLinkIcon, PauseIcon, UserPlusIcon, XIcon } from "lucide-react";

const STATUS_COLORS: Record<BillingQueueStatus, string> = Object.fromEntries(
  billingQueueStatusChoices.map((c) => [c.value, c.color]),
) as Record<BillingQueueStatus, string>;

export function BillingQueueItemCard({
  item,
  isSelected,
  onClick,
  onAssignBiller,
  onHold,
  onCancel,
}: {
  item: BillingQueueItem;
  isSelected: boolean;
  onClick: () => void;
  onAssignBiller: () => void;
  onHold: () => void;
  onCancel: () => void;
}) {
  const proNumber = item.shipment?.proNumber || item.shipmentId.slice(0, 12);
  const customerName = item.shipment?.customer?.name;
  const totalCharges = item.shipment?.totalChargeAmount;
  const age = formatDistanceToNowStrict(fromUnixTime(item.createdAt), { addSuffix: true });
  const isTerminal = item.status === "Approved" || item.status === "Canceled";

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <button
          type="button"
          onClick={onClick}
          className={cn(
            "flex w-full items-stretch gap-2 rounded-md border p-2.5 text-left transition-colors",
            "hover:bg-accent/50",
            isSelected ? "border-border bg-muted" : "border-border",
          )}
        >
          <div
            className="w-[3px] shrink-0 rounded-full"
            style={{ backgroundColor: STATUS_COLORS[item.status] }}
          />
          <div className="flex flex-1 min-w-0 flex-col gap-0.5">
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-semibold truncate">{proNumber}</span>
              {totalCharges != null && (
                <span className="text-xs font-medium tabular-nums shrink-0">
                  {formatCurrency(Number(totalCharges))}
                </span>
              )}
            </div>
            <span className="text-xs text-muted-foreground truncate">
              {customerName || "No customer"}
            </span>
            <Tooltip>
              <TooltipTrigger
                render={
                  <div className="flex items-center gap-1 w-fit">
                    <span className="text-[11px] text-muted-foreground/70">{age}</span>
                  </div>
                }
              ></TooltipTrigger>
              <TooltipContent side="left" sideOffset={10}>
                {generateDateTimeStringFromUnixTimestamp(item.createdAt)}
              </TooltipContent>
            </Tooltip>
          </div>
        </button>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onAssignBiller} disabled={isTerminal}>
          <UserPlusIcon className="size-3.5" />
          Assign Biller
        </ContextMenuItem>
        <ContextMenuItem onClick={onHold} disabled={isTerminal || item.status === "OnHold"}>
          <PauseIcon className="size-3.5" />
          Hold
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          onClick={() =>
            window.open(`/shipment-management/shipments?item=${item.shipmentId}`, "_blank")
          }
        >
          <ExternalLinkIcon className="size-3.5" />
          View Shipment
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onClick={onCancel} disabled={isTerminal} className="text-destructive">
          <XIcon className="size-3.5" />
          Cancel
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
