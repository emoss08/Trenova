import { BillingRecordCard } from "@/components/billing/billing-record-card";
import { PlainBillingQueueStatusBadge } from "@/components/status-badge";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { formatCurrency } from "@/lib/utils";
import type { BillingQueueItem } from "@/types/billing-queue";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ExternalLinkIcon, PauseIcon, UserPlusIcon, XIcon } from "lucide-react";

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
        <BillingRecordCard
          title={proNumber}
          auxiliary={
            item.number ? (
              <span className="font-mono text-[10px] text-muted-foreground">{item.number}</span>
            ) : null
          }
          amount={totalCharges != null ? formatCurrency(Number(totalCharges)) : undefined}
          subtitle={customerName || "No customer"}
          meta={
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5">
                <PlainBillingQueueStatusBadge status={item.status} />
                {item.isAdjustmentOrigin ? (
                  <span className="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-950 dark:text-amber-300">
                    Rebill
                  </span>
                ) : null}
              </div>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="text-[11px] text-muted-foreground/70">{age}</span>
                  }
                />
                <TooltipContent side="left" sideOffset={10}>
                  {generateDateTimeStringFromUnixTimestamp(item.createdAt)}
                </TooltipContent>
              </Tooltip>
            </div>
          }
          isSelected={isSelected}
          onClick={onClick}
        />
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
