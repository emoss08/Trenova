import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@trenova/shared/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useNow } from "@trenova/shared/hooks/use-now";
import {
  formatCompactAge,
  generateDateTimeStringFromUnixTimestamp,
} from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { BillingQueueRow } from "@/lib/graphql/billing-queue";
import {
  LinkExternal01Icon,
  PauseIcon,
  UserPlus01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { billingQueueItemLabel } from "@/lib/billing-queue-label";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { holdReasonLabel } from "@/lib/billing-queue-checks";
import { recordPath } from "@/config/record-links";

type RowNote = {
  text: string;
  tone: "warning" | "muted" | "nova" | "concern";
};

/**
 * The line a row leads with: what the item waits on while it waits, what Nova made
 * of it once it is ready, and where it stands after that.
 */
export function rowNote(item: BillingQueueRow, t: TranslateFn): RowNote {
  const headline = item.headline;
  const more = headline && headline.open > 1 ? headline.open - 1 : 0;
  const withMore = (text: string) => (more > 0 ? t("{0} · {1} more", text, more) : text);

  switch (item.lane) {
    case "NeedsYou":
      if (headline) return { text: withMore(headline.summary), tone: "warning" };
      if (!item.assignedBillerId) return { text: t("Nobody is assigned"), tone: "warning" };
      return { text: t("Waiting for review"), tone: "warning" };
    case "Waiting":
      return {
        text: headline ? withMore(headline.summary) : t("Waiting on someone else"),
        tone: "muted",
      };
    case "Ready":
      if (item.agentVerdict === "Concern" && item.agentSummary) {
        return { text: item.agentSummary, tone: "concern" };
      }
      if (item.agentVerdict === "Agrees" && item.agentSummary) {
        return { text: item.agentSummary, tone: "nova" };
      }
      return { text: t("All five checks clear"), tone: "muted" };
    case "Checking":
      return { text: t("Running the checks"), tone: "muted" };
    case "Held": {
      const reason = holdReasonLabel(item.holdReasonCode, t);
      return {
        text: reason ? t("On hold · {0}", reason) : t("On hold"),
        tone: "muted",
      };
    }
    case "Approved":
      return { text: t("Approved · invoice not posted"), tone: "muted" };
    case "Closed":
      return { text: t("Closed"), tone: "muted" };
  }
}

export function BillingQueueItemCard({
  item,
  isSelected,
  onClick,
  onAssignBiller,
  onHold,
  onCancel,
  checked,
  onCheckedChange,
}: {
  item: BillingQueueRow;
  isSelected: boolean;
  onClick: () => void;
  onAssignBiller: () => void;
  onHold: () => void;
  onCancel: () => void;
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const canReadShipment = useCanReadKind("shipment");

  const proNumber = item.shipment?.proNumber || billingQueueItemLabel(item);
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
  // One queue item per payer: the row shows this payer's share, and says so,
  // or two rows for one shipment both read as billing the full freight.
  const isPartial =
    allocated != null && shipmentTotal != null && Math.abs(allocated - shipmentTotal) >= 0.005;
  const amount = isPartial ? allocated : (shipmentTotal ?? allocated);
  const isTerminal = item.status === "Approved" || item.status === "Canceled";
  const note = rowNote(item, t);

  const showMeter = item.lane === "Ready" && !item.agentVerdict;

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <div
          className={cn(
            "mx-1.5 flex items-start gap-2 rounded-md p-2 text-sm transition-colors",
            isSelected ? "bg-surface-selected" : "hover:bg-surface-hover",
          )}
        >
          {onCheckedChange ? (
            <Checkbox
              className="mt-0.5 shrink-0"
              checked={checked === true}
              onCheckedChange={(value) => onCheckedChange(value === true)}
              aria-label={t("Select {0}", proNumber)}
            />
          ) : null}
          <button
            type="button"
            onClick={onClick}
            aria-current={isSelected ? "true" : undefined}
            className="ui-focus-ring flex min-w-0 flex-1 flex-col gap-0.5 rounded-sm text-left"
          >
            <span className="flex w-full min-w-0 items-baseline gap-2">
              <span className="text-foreground shrink-0 font-mono font-medium">{proNumber}</span>
              <span className="text-foreground-muted min-w-0 flex-1 truncate">
                {payerName || t("No customer")}
              </span>
              {amount != null ? (
                <span className="text-foreground shrink-0 font-mono">{formatCurrency(amount)}</span>
              ) : null}
            </span>
            {onBehalfOf ? (
              <span className="text-foreground-muted truncate text-xs">
                {t("On behalf of {0}", onBehalfOf)}
              </span>
            ) : null}
            <span className="flex w-full min-w-0 items-center gap-1.5 text-xs">
              {showMeter ? (
                <span
                  role="img"
                  aria-label={t("All five checks clear")}
                  className="flex shrink-0 gap-0.5"
                >
                  {[0, 1, 2, 3, 4].map((index) => (
                    <span key={index} className="bg-success size-1.5 rounded-xs" />
                  ))}
                </span>
              ) : null}
              {note.tone === "nova" || note.tone === "concern" ? (
                <NovaBadge size="xs" className="size-2.5 shrink-0" />
              ) : null}
              <span
                className={cn(
                  "min-w-0 flex-1 truncate",
                  note.tone === "warning" || note.tone === "concern"
                    ? "text-warning-foreground"
                    : "text-foreground-muted",
                )}
              >
                {note.text}
              </span>
              {item.isAdjustmentOrigin ? (
                <span className="bg-warning-subtle text-warning-foreground rounded-xs shrink-0 px-1 text-2xs font-medium">
                  {t("Rebill")}
                </span>
              ) : null}
              {isPartial ? (
                <span className="bg-accent-indigo-subtle text-accent-indigo-on-subtle rounded-xs shrink-0 px-1 text-2xs font-medium">
                  {t("Split")}
                </span>
              ) : null}
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="text-foreground-subtle shrink-0">
                      {formatCompactAge(now - item.createdAt)}
                    </span>
                  }
                />
                <TooltipContent side="left" sideOffset={10}>
                  {t("Queued {0}", generateDateTimeStringFromUnixTimestamp(item.createdAt))}
                </TooltipContent>
              </Tooltip>
            </span>
          </button>
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onAssignBiller} disabled={isTerminal}>
          <UserPlus01Icon className="size-3.5" />
          {t("Assign biller")}
        </ContextMenuItem>
        <ContextMenuItem onClick={onHold} disabled={isTerminal || item.status === "OnHold"}>
          <PauseIcon className="size-3.5" />
          {t("Hold")}
        </ContextMenuItem>
        {item.shipmentId && canReadShipment ? (
          <>
            <ContextMenuSeparator />
            <ContextMenuItem
              onClick={() =>
                window.open(
                  recordPath("shipment", item.shipmentId ?? ""),
                  "_blank",
                  "noopener,noreferrer",
                )
              }
            >
              <LinkExternal01Icon className="size-3.5" />
              {t("View shipment")}
            </ContextMenuItem>
          </>
        ) : null}
        <ContextMenuSeparator />
        <ContextMenuItem onClick={onCancel} disabled={isTerminal} className="text-destructive">
          <XCloseIcon className="size-3.5" />
          {t("Cancel")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
