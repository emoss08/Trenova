import { BillingRecordCard } from "@/components/billing/billing-record-card";
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
import { formatCurrency } from "@/lib/utils";
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
        <BillingRecordCard
          accentColor={STATUS_COLORS[item.status]}
          title={proNumber}
          auxiliary={
            item.number ? (
              <span className="font-mono text-[10px] text-muted-foreground">{item.number}</span>
            ) : null
          }
          amount={totalCharges != null ? formatCurrency(Number(totalCharges)) : undefined}
          subtitle={customerName || "No customer"}
          meta={
            <Tooltip>
              <TooltipTrigger
                render={
                  <div className="flex w-fit items-center gap-1">
                    <span className="text-[11px] text-muted-foreground/70">{age}</span>
                  </div>
                }
              />
              <TooltipContent side="left" sideOffset={10}>
                {generateDateTimeStringFromUnixTimestamp(item.createdAt)}
              </TooltipContent>
            </Tooltip>
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
