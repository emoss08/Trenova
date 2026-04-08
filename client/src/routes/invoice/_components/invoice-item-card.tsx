import { BillingRecordCard } from "@/components/billing/billing-record-card";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { formatCurrency } from "@/lib/utils";
import type { Invoice, InvoiceStatus } from "@/types/invoice";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ExternalLinkIcon, FileTextIcon, SendIcon } from "lucide-react";

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  Draft: "#64748b",
  Posted: "#16a34a",
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
          accentColor={STATUS_COLORS[invoice.status]}
          title={invoice.number}
          auxiliary={
            <span className="font-mono text-[10px] text-muted-foreground">{invoice.billType}</span>
          }
          amount={formatCurrency(totalAmount, invoice.currencyCode)}
          subtitle={customerName || "Unknown bill-to"}
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
                {generateDateTimeStringFromUnixTimestamp(invoice.createdAt)}
              </TooltipContent>
            </Tooltip>
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
