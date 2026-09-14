import { useT } from "@trenova/shared/i18n/use-t";
import { BillingRecordCard } from "@/components/billing/billing-record-card";
import {
  PlainInvoiceScopeBadge,
  PlainInvoiceStatusBadge,
  PlainSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@trenova/shared/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { InvoiceTableRowFieldsFragment } from "@trenova/graphql/generated/graphql";
import { invoiceBillingPeriod, invoiceBillsSingleShipment } from "@/lib/invoice-scope";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ExternalLinkIcon, FileTextIcon, PackageIcon, SendIcon } from "lucide-react";

export function InvoiceItemCard({
  invoice,
  isSelected,
  onClick,
  onPost,
}: {
  invoice: InvoiceTableRowFieldsFragment;
  isSelected: boolean;
  onClick: () => void;
  onPost: () => void;
}) {
  const t = useT();

  const age = formatDistanceToNowStrict(fromUnixTime(invoice.createdAt), { addSuffix: true });
  const customerName = invoice.customer?.name ?? invoice.billToName;
  const totalAmount = Number(invoice.totalAmount ?? 0);
  const billsSingleShipment = invoiceBillsSingleShipment(invoice.scope);
  const billingPeriod = invoiceBillingPeriod(invoice);
  const scopeDetail = billingPeriod
    ? t(
        "{0} · {1, plural, one {# shipment} other {# shipments}}",
        billingPeriod,
        invoice.shipmentCount,
      )
    : !billsSingleShipment
      ? t("{0, plural, one {# shipment} other {# shipments}}", invoice.shipmentCount)
      : null;

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <BillingRecordCard
          title={invoice.number}
          auxiliary={
            <span className="flex items-center gap-1.5">
              <PlainInvoiceScopeBadge scope={invoice.scope} />
              <span className="text-muted-foreground font-mono text-[10px]">
                {invoice.billType}
              </span>
            </span>
          }
          amount={formatCurrency(totalAmount, invoice.currencyCode)}
          subtitle={customerName || "Unknown bill-to"}
          meta={
            <div className="flex flex-col gap-1">
              {scopeDetail ? (
                <span className="text-muted-foreground truncate text-[11px] tabular-nums">
                  {scopeDetail}
                </span>
              ) : null}
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5">
                  <PlainInvoiceStatusBadge status={invoice.status} />
                  <PlainSettlementStatusBadge status={invoice.settlementStatus} />
                </div>
                <Tooltip>
                  <TooltipTrigger
                    render={<span className="text-muted-foreground/70 text-[11px]">{age}</span>}
                  />
                  <TooltipContent side="left" sideOffset={10}>
                    {generateDateTimeStringFromUnixTimestamp(invoice.createdAt)}
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
        {billsSingleShipment && invoice.shipmentId ? (
          <ContextMenuItem
            onClick={() => window.open(shipmentPanelPath(invoice.shipmentId), "_blank")}
          >
            <ExternalLinkIcon className="size-3.5" />
            {t("View Shipment")}
          </ContextMenuItem>
        ) : null}
        {invoice.scope === "Order" && invoice.orderId ? (
          <ContextMenuItem
            onClick={() =>
              window.open(
                `/shipment-management/orders?panelType=edit&panelEntityId=${invoice.orderId}`,
                "_blank",
              )
            }
          >
            <PackageIcon className="size-3.5" />
            {t("View Order")}
          </ContextMenuItem>
        ) : null}
        {billsSingleShipment ? (
          <ContextMenuItem
            onClick={() =>
              window.open(`/billing/queue?item=${invoice.billingQueueItemId}`, "_blank")
            }
          >
            <FileTextIcon className="size-3.5" />
            {t("View Billing Queue Item")}
          </ContextMenuItem>
        ) : null}
        <ContextMenuItem onClick={onPost} disabled={invoice.status === "Posted"}>
          <SendIcon className="size-3.5" />
          {t("Post Invoice")}
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
