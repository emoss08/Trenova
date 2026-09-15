import { MemoDialog } from "@/components/billing/memo-dialog";
import { usePermission } from "@/hooks/use-permission";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { FileMinusIcon, FilePlusIcon } from "lucide-react";
import { useState } from "react";

/**
 * Opens the memo dialog for a customer from wherever the customer is in
 * view: the ledger, the billing profile. Renders nothing for a viewer who
 * cannot create invoices, so the absence of the button is itself the answer.
 */
export function MemoEntryButton({
  customerId,
  billType,
}: {
  customerId: string;
  billType: "CreditMemo" | "DebitMemo";
}) {
  const t = useT();
  const { allowed } = usePermission(Resource.Invoice, Operation.Create);
  const [open, setOpen] = useState(false);

  if (!allowed) return null;

  const Icon = billType === "CreditMemo" ? FileMinusIcon : FilePlusIcon;

  return (
    <>
      <Button type="button" size="sm" variant="outline" onClick={() => setOpen(true)}>
        <Icon className="size-3.5" />
        {billType === "CreditMemo" ? t("Credit memo") : t("Debit memo")}
      </Button>
      {open ? (
        <MemoDialog open onOpenChange={setOpen} billType={billType} customerId={customerId} />
      ) : null}
    </>
  );
}
