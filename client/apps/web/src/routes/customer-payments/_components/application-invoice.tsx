import { LinkedRecordLink } from "@/components/linked-record-link";
import type { CustomerPaymentDetail } from "@/lib/graphql/customer-payment";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { useT } from "@trenova/shared/i18n/use-t";

type ApplicationInvoiceProps = {
  invoice: NonNullable<CustomerPaymentDetail["applications"]>[number]["invoice"];
};

/**
 * The invoice a payment was applied to: its number, linking to it, and who it bills. The server
 * leaves the invoice out when the person may not read invoices, which reads as withheld, and
 * when it cannot find it, which says so; never the invoice's id.
 */
export function ApplicationInvoice({ invoice }: ApplicationInvoiceProps) {
  const t = useT();
  const canReadInvoices = useCanReadKind("invoice");

  if (!invoice) {
    return canReadInvoices ? (
      <span className="text-xs text-muted-foreground">{t("Invoice not found")}</span>
    ) : (
      <WithheldValue kind="invoice" />
    );
  }

  return (
    <div className="flex flex-col">
      <LinkedRecordLink
        kind="invoice"
        id={invoice.id}
        className="font-mono text-xs font-medium hover:underline"
      >
        {invoice.number}
      </LinkedRecordLink>
      <span className="text-xs text-muted-foreground">{invoice.billToName}</span>
    </div>
  );
}
