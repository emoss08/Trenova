import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { sendInvoiceEdi, type InvoiceArContext } from "@/lib/graphql/invoice";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { invoiceVoidBlocker, type Invoice } from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BanIcon, MoreHorizontalIcon, RadioTowerIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { InvoiceVoidDialog } from "./invoice-void-dialog";

/**
 * Why an EDI send cannot go right now, in the customer's terms. The plan's own
 * blockers come first; a missing plan means the AR context has not loaded.
 */
export function invoiceEdiSendBlocker(
  invoice: Pick<Invoice, "status">,
  arContext: InvoiceArContext | null | undefined,
): string | null {
  if (!arContext) return "Checking the customer's EDI configuration.";
  const plan = arContext.ediSendPlan;
  if (!plan.enabled) return plan.blockers[0] ?? "EDI invoicing is not enabled for this customer.";
  if (plan.blockers.length > 0) return plan.blockers[0];
  if (invoice.status !== "Posted") return "Only a posted invoice can be sent by EDI.";
  if (plan.status === "Queued" || plan.status === "Sending") {
    return "An EDI send for this invoice is already in progress.";
  }
  return null;
}

/**
 * The invoice-level actions that are neither posting nor sharing: voiding and
 * pushing its EDI 210. Crediting or rebilling an invoice goes through Adjust
 * Invoice, and standalone memos are raised from the customer, so neither is
 * offered here. Each action is gated by the permission the server checks, and
 * each says why it is unavailable rather than vanishing.
 */
export function InvoiceActionsMenu({
  invoice,
  arContext,
}: {
  invoice: Invoice;
  arContext: InvoiceArContext | null | undefined;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [voidOpen, setVoidOpen] = useState(false);

  const { allowed: canVoid } = usePermission(Resource.Invoice, Operation.Cancel);
  const { allowed: canSend } = usePermission(Resource.Invoice, Operation.Update);

  const voidBlocker = invoiceVoidBlocker(invoice, {
    paymentApplications: arContext?.paymentApplications.length ?? 0,
    creditApplications: arContext?.creditApplications.length ?? 0,
  });
  const ediBlocker = invoiceEdiSendBlocker(invoice, arContext);
  const ediSent = arContext?.ediSendPlan.status === "Sent";

  const sendEdi = useApiMutation({
    resourceName: "invoice EDI",
    mutationFn: () => sendInvoiceEdi(invoice.id, ediSent),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("EDI 210 for {0} queued", invoice.number));
    },
  });

  if (!canVoid && !canSend) {
    return null;
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              aria-label={t("Invoice actions")}
            >
              <MoreHorizontalIcon className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" className="w-72">
          {canSend ? (
            <>
              <DropdownMenuItem
                title={ediSent ? t("Resend EDI") : t("Send EDI")}
                description={ediBlocker ? t(ediBlocker) : undefined}
                descriptionClassProps="whitespace-normal"
                startContent={<RadioTowerIcon className="size-3.5" />}
                disabled={Boolean(ediBlocker) || sendEdi.isPending}
                onClick={() => sendEdi.mutate()}
              />
            </>
          ) : null}
          {canVoid ? (
            <>
              {canSend ? <DropdownMenuSeparator /> : null}
              <DropdownMenuItem
                title={t("Void invoice")}
                description={voidBlocker ? t(voidBlocker) : undefined}
                descriptionClassProps="whitespace-normal"
                startContent={<BanIcon className="size-3.5" />}
                color="danger"
                disabled={Boolean(voidBlocker)}
                onClick={() => setVoidOpen(true)}
              />
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>

      <InvoiceVoidDialog invoice={invoice} open={voidOpen} onOpenChange={setVoidOpen} />
    </>
  );
}
