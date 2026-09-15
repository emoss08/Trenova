import { useT } from "@trenova/shared/i18n/use-t";
import {
  PlainInvoiceStatusBadge,
  PlainSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { invoicePanelPath } from "@/lib/invoice-links";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import { useQuery } from "@tanstack/react-query";
import { ExternalLinkIcon, SplitIcon } from "lucide-react";
import { Link } from "react-router";

/**
 * True when the invoice bills someone other than the customer whose freight it
 * is: a third-party bill-to, or one payer's slice of a split shipment. Only
 * then is there anything to fetch.
 */
export function invoiceBillsOnBehalf(
  invoice: Pick<Invoice, "customerId" | "shipperCustomerId" | "isSplitBill">,
) {
  if (invoice.isSplitBill) return true;
  return Boolean(invoice.shipperCustomerId && invoice.shipperCustomerId !== invoice.customerId);
}

/**
 * Whose shipment the payer is being billed for, and the sibling invoices that
 * bill the rest of it. The bill-to block already says who pays; an AR clerk at
 * the payer needs the shipper's name to recognise the invoice at all.
 */
export function InvoiceArContextSection({ invoice }: { invoice: Invoice }) {
  if (!invoiceBillsOnBehalf(invoice)) return null;
  return <ArContextCard invoice={invoice} />;
}

function ArContextCard({ invoice }: { invoice: Invoice }) {
  const t = useT();
  const { data, isLoading } = useQuery(queries.invoice.arContext(invoice.id));

  const shipper = data?.shipperCustomer ?? invoice.shipperCustomer ?? null;
  const related = (data?.relatedInvoices ?? []).filter((row) => row.id !== invoice.id);

  return (
    <div className="bg-card rounded-lg border p-3" data-testid="invoice-ar-context">
      <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
        {t("On behalf of")}
      </p>
      {isLoading && !shipper ? (
        <Skeleton className="mt-1.5 h-4 w-40" />
      ) : shipper ? (
        <div className="mt-1.5">
          <p className="text-sm font-medium">{shipper.name}</p>
          {shipper.code ? (
            <p className="text-2xs text-muted-foreground mt-0.5">{shipper.code}</p>
          ) : null}
        </div>
      ) : (
        <p className="text-muted-foreground mt-1.5 text-xs">{t("Shipper not recorded")}</p>
      )}
      <p className="text-muted-foreground mt-1 text-xs">
        {invoice.isSplitBill
          ? t(
              "This invoice bills {0}'s share of a shipment split between payers.",
              invoice.billToName,
            )
          : t(
              "This invoice bills {0} for freight ordered by another customer.",
              invoice.billToName,
            )}
      </p>

      {related.length > 0 ? (
        <div className="mt-3 border-t pt-2">
          <p className="text-2xs text-muted-foreground mb-1.5 flex items-center gap-1 font-medium tracking-wide uppercase">
            <SplitIcon className="size-3" />
            {t("Related invoices")}
          </p>
          <ul className="space-y-1">
            {related.map((row) => (
              <li key={row.id} className="flex items-center justify-between gap-2 text-xs">
                <span className="flex min-w-0 items-center gap-1.5">
                  <Link
                    to={invoicePanelPath(row.id)}
                    className="inline-flex items-center gap-1 font-medium hover:underline"
                  >
                    {row.number}
                    <ExternalLinkIcon className="size-2.5" />
                  </Link>
                  <span className="text-muted-foreground truncate">{row.billToName}</span>
                </span>
                <span className="flex shrink-0 items-center gap-1.5">
                  <PlainInvoiceStatusBadge status={row.status} />
                  <PlainSettlementStatusBadge status={row.settlementStatus} />
                  <span className="tabular-nums">
                    {formatCurrency(Number(row.totalAmount ?? 0), row.currencyCode)}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
