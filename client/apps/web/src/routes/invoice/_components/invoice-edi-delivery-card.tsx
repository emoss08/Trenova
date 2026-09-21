import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { sendInvoiceEdi, type InvoiceArContext } from "@/lib/graphql/invoice";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { useQueryClient } from "@tanstack/react-query";
import { InvoiceEdiSendStatusBadge } from "@trenova/shared/components/status-badge";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { Invoice } from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AlertTriangleIcon, RadioTowerIcon } from "lucide-react";
import { toast } from "sonner";
import { invoiceEdiSendBlocker } from "./invoice-actions-menu";

/**
 * The EDI half of delivery: whether the customer takes an 210, which partner
 * gets it, what stands in the way, and where the last send got to. Sits next
 * to the email card so an operator sees both channels at once.
 */
export function InvoiceEdiDeliveryCard({
  invoice,
  arContext,
  isLoading,
}: {
  invoice: Invoice;
  arContext: InvoiceArContext | null | undefined;
  isLoading: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canSend } = usePermission(Resource.Invoice, Operation.Update);
  const plan = arContext?.ediSendPlan;
  const blocker = invoiceEdiSendBlocker(invoice, arContext);
  const sent = plan?.status === "Sent";

  const send = useApiMutation({
    resourceName: "invoice EDI",
    mutationFn: () => sendInvoiceEdi(invoice.id, sent),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("EDI 210 for {0} queued", invoice.number));
    },
  });

  return (
    <div className="border-border rounded-md border p-4" data-testid="invoice-edi-delivery">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">{t("EDI delivery")}</h3>
            {plan ? <InvoiceEdiSendStatusBadge status={plan.status} /> : null}
          </div>
          {isLoading && !plan ? (
            <Skeleton className="mt-2 h-4 w-48" />
          ) : !plan?.enabled ? (
            <p className="text-muted-foreground mt-1 text-sm">
              {t("EDI invoicing is not enabled for this customer")}
            </p>
          ) : (
            <p className="text-muted-foreground mt-1 text-sm">
              {plan.partnerName
                ? t("Sent to {0} by {1}", plan.partnerName, plan.communicationMethod || t("EDI"))
                : t("No EDI partner is linked to this customer")}
              {plan.sentAt ? ` · ${t("Last sent {0}", formatUnixDateTime(plan.sentAt))}` : ""}
            </p>
          )}
        </div>
        {canSend && plan?.enabled ? (
          <Button
            size="sm"
            variant="outline"
            disabled={Boolean(blocker) || send.isPending}
            onClick={() => send.mutate()}
          >
            <RadioTowerIcon className="size-3.5" />
            {sent ? t("Resend EDI") : t("Send EDI")}
          </Button>
        ) : null}
      </div>

      {plan?.enabled && plan.blockers.length > 0 ? (
        <ul className="mt-3 space-y-1">
          {plan.blockers.map((item) => (
            <li key={item}>
              <Alert variant="warning" size="sm">
                <AlertTriangleIcon />
                <AlertDescription>{item}</AlertDescription>
              </Alert>
            </li>
          ))}
        </ul>
      ) : null}

      {plan?.lastError ? (
        <Alert variant="destructive" size="sm" className="mt-3">
          <AlertTriangleIcon />
          <AlertDescription>{plan.lastError}</AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
