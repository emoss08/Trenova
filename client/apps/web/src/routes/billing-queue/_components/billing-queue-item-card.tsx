import { useT } from "@trenova/shared/i18n/use-t";
import { BillingRecordCard } from "@/components/billing/billing-record-card";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@trenova/shared/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
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
  const t = useT();

  const proNumber = item.shipment?.proNumber || item.shipmentId.slice(0, 12);
  const customerName = item.shipment?.customer?.name;
  const payerName = item.billToCustomer?.name ?? customerName;
  const onBehalfOf =
    item.billToCustomerId &&
    item.shipment?.customerId &&
    item.billToCustomerId !== item.shipment.customerId
      ? customerName
      : null;
  const shipmentTotal =
    item.shipment?.totalChargeAmount != null ? Number(item.shipment.totalChargeAmount) : null;
  const allocated = item.allocatedTotalAmount != null ? Number(item.allocatedTotalAmount) : null;
  // One queue item per payer: the card shows this payer's share, and says so,
  // or two cards for one shipment both read as billing the full freight.
  const isPartial =
    allocated != null && shipmentTotal != null && Math.abs(allocated - shipmentTotal) >= 0.005;
  const totalCharges = isPartial ? allocated : shipmentTotal;
  const age = formatDistanceToNowStrict(fromUnixTime(item.createdAt), { addSuffix: true });
  const isTerminal = item.status === "Approved" || item.status === "Canceled";

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <BillingRecordCard
          title={proNumber}
          auxiliary={
            item.number ? (
              <span className="text-muted-foreground font-mono text-[10px]">{item.number}</span>
            ) : null
          }
          amount={totalCharges != null ? formatCurrency(Number(totalCharges)) : undefined}
          subtitle={payerName || "No customer"}
          meta={
            <div className="flex flex-col gap-1">
              {onBehalfOf ? (
                <span className="text-muted-foreground truncate text-[11px]">
                  {t("On behalf of {0}", onBehalfOf)}
                </span>
              ) : null}
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5">
                  <PlainBillingQueueStatusBadge status={item.status} />
                  {item.isAdjustmentOrigin ? (
                    <span className="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-950 dark:text-amber-300">
                      {t("Rebill")}
                    </span>
                  ) : null}
                  {isPartial ? (
                    <span className="inline-flex items-center rounded-full bg-indigo-50 px-2 py-0.5 text-[11px] font-medium text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300">
                      {t("Split")}
                    </span>
                  ) : null}
                </div>
                <Tooltip>
                  <TooltipTrigger
                    render={<span className="text-muted-foreground/70 text-[11px]">{age}</span>}
                  />
                  <TooltipContent side="left" sideOffset={10}>
                    {generateDateTimeStringFromUnixTimestamp(item.createdAt)}
                  </TooltipContent>
                </Tooltip>
              </div>
            </div>
          }
          isSelected={isSelected}
          onClick={onClick}
        />
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onAssignBiller} disabled={isTerminal}>
          <UserPlusIcon className="size-3.5" />
          {t("Assign Biller")}
        </ContextMenuItem>
        <ContextMenuItem onClick={onHold} disabled={isTerminal || item.status === "OnHold"}>
          <PauseIcon className="size-3.5" />
          {t("Hold")}
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          onClick={() =>
            window.open(`/shipment-management/shipments?item=${item.shipmentId}`, "_blank")
          }
        >
          <ExternalLinkIcon className="size-3.5" />
          {t("View Shipment")}
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onClick={onCancel} disabled={isTerminal} className="text-destructive">
          <XIcon className="size-3.5" />
          {t("Cancel")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
