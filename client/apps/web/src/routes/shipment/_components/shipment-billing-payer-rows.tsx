import { useT } from "@trenova/shared/i18n/use-t";
import {
  PlainBillingQueueStatusBadge,
  PlainInvoiceStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { invoicePanelPath } from "@/lib/invoice-links";
import { queries } from "@/lib/queries";
import { billingQueueItemsByShipmentQuery } from "@/routes/billing-queue/billing-queue-queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ShipmentBillingPayerReadiness } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { ExternalLinkIcon, ShieldAlertIcon } from "lucide-react";
import { Link } from "react-router";

/**
 * One line per payer on a split shipment: their share, where their queue item
 * sits, and the invoice once it exists. Each payer moves through billing on
 * their own, so a single status for the shipment would hide the one that is
 * stuck.
 */
export function ShipmentBillingPayerRows({
  shipmentId,
  payers,
}: {
  shipmentId: string;
  payers: ShipmentBillingPayerReadiness[];
}) {
  const t = useT();
  const { data: queueItems, isLoading: queueLoading } = useQuery({
    ...billingQueueItemsByShipmentQuery(shipmentId),
    enabled: Boolean(shipmentId),
  });
  const { data: invoices, isLoading: invoicesLoading } = useQuery({
    ...queries.invoice.byShipment(shipmentId),
    enabled: Boolean(shipmentId),
  });

  if (payers.length === 0) return null;

  return (
    <div className="mt-3 space-y-1" data-testid="billing-payer-rows">
      <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
        {t("Payers")}
      </p>
      {payers.map((payer) => {
        const queueItem = (queueItems?.results ?? []).find(
          (item) => item.billToCustomerId === payer.payerId,
        );
        const invoice = (invoices ?? []).find((row) => row.customerId === payer.payerId);
        const label = payer.payerCode ? `${payer.payerCode} – ${payer.payerName}` : payer.payerName;
        return (
          <div
            key={payer.payerId}
            className="bg-muted/40 flex items-center justify-between gap-3 rounded-md px-2.5 py-1.5"
            data-testid={`billing-payer-row-${payer.payerId}`}
          >
            <div className="flex min-w-0 items-center gap-1.5">
              <span className="truncate text-xs font-medium">{label}</span>
              {payer.isPrimary ? (
                <span className="bg-primary/10 text-2xs text-primary rounded px-1 py-0.5">
                  {t("Primary")}
                </span>
              ) : null}
              {payer.creditHold ? (
                <span
                  className="text-destructive text-2xs inline-flex items-center gap-0.5"
                  title={t("Credit hold: {0}", payer.creditStatus)}
                >
                  <ShieldAlertIcon className="size-3" />
                  {t("Credit hold")}
                </span>
              ) : null}
            </div>
            <div className="flex shrink-0 items-center gap-2 text-xs">
              <span className="tabular-nums">{formatCurrency(Number(payer.shareAmount ?? 0))}</span>
              {queueLoading || invoicesLoading ? (
                <Skeleton className="h-4 w-16" />
              ) : invoice ? (
                <Link
                  to={invoicePanelPath(invoice.id)}
                  className="inline-flex items-center gap-1 hover:underline"
                >
                  <PlainInvoiceStatusBadge status={invoice.status} />
                  <span className="font-mono text-[10px]">{invoice.number}</span>
                  <ExternalLinkIcon className="size-2.5" />
                </Link>
              ) : queueItem ? (
                <Link
                  to={`/billing/queue?item=${queueItem.id}&includePosted=true`}
                  className="inline-flex items-center gap-1 hover:underline"
                >
                  <PlainBillingQueueStatusBadge status={queueItem.status} />
                  <ExternalLinkIcon className="size-2.5" />
                </Link>
              ) : (
                <span className="text-muted-foreground">{t("Not transferred")}</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
