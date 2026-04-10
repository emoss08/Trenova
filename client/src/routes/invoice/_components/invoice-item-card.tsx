import { BillingRecordCard } from "@/components/billing/billing-record-card";
import { PlainInvoiceStatusBadge } from "@/components/status-badge";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { cn, formatCurrency } from "@/lib/utils";
import type { Invoice } from "@/types/invoice";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ExternalLinkIcon, FileTextIcon, SendIcon } from "lucide-react";

const SETTLEMENT_STYLES: Record<string, string> = {
  Paid: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
  PartiallyPaid: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  Unpaid: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
};

export function InvoiceItemCard({
  invoice,
  isSelected,
  onClick,
  onPost,
}: {
  invoice: Invoice;
  isSelected: boolean;
  onClick: () => void;
  onPost: () => void;
}) {
  const age = formatDistanceToNowStrict(fromUnixTime(invoice.createdAt), { addSuffix: true });
  const customerName = invoice.customer?.name ?? invoice.billToName;
  const totalAmount = Number(invoice.totalAmount ?? 0);

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <BillingRecordCard
          title={invoice.number}
          auxiliary={
            <span className="font-mono text-[10px] text-muted-foreground">{invoice.billType}</span>
          }
          amount={formatCurrency(totalAmount, invoice.currencyCode)}
          subtitle={customerName || "Unknown bill-to"}
          meta={
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5">
                <PlainInvoiceStatusBadge status={invoice.status} />
                <span
                  className={cn(
                    "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
                    SETTLEMENT_STYLES[invoice.settlementStatus] ?? SETTLEMENT_STYLES.Unpaid,
                  )}
                >
                  {invoice.settlementStatus === "PartiallyPaid"
                    ? "Partial"
                    : invoice.settlementStatus}
                </span>
              </div>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="text-[11px] text-muted-foreground/70">{age}</span>
                  }
                />
                <TooltipContent side="left" sideOffset={10}>
                  {generateDateTimeStringFromUnixTimestamp(invoice.createdAt)}
                </TooltipContent>
              </Tooltip>
            </div>
          }
          isSelected={isSelected}
          onClick={onClick}
        />
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem
          onClick={() =>
            window.open(`/shipment-management/shipments?item=${invoice.shipmentId}`, "_blank")
          }
        >
          <ExternalLinkIcon className="size-3.5" />
          View Shipment
        </ContextMenuItem>
        <ContextMenuItem
          onClick={() => window.open(`/billing/queue?item=${invoice.billingQueueItemId}`, "_blank")}
        >
          <FileTextIcon className="size-3.5" />
          View Billing Queue Item
        </ContextMenuItem>
        <ContextMenuItem onClick={onPost} disabled={invoice.status === "Posted"}>
          <SendIcon className="size-3.5" />
          Post Invoice
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
