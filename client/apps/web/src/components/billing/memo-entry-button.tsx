import { MemoDialog } from "@/components/billing/memo-dialog";
import { usePermission } from "@/hooks/use-permission";
import { FileMinus02Icon, FilePlus02Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";

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

  const Icon = billType === "CreditMemo" ? FileMinus02Icon : FilePlus02Icon;

  return (
    <>
      <Button type="button" size="sm" variant="ghostInvert" onClick={() => setOpen(true)}>
        <Icon className="size-3.5" />
        {billType === "CreditMemo" ? t("Credit memo") : t("Debit memo")}
      </Button>
      {open ? (
        <MemoDialog open onOpenChange={setOpen} billType={billType} customerId={customerId} />
      ) : null}
    </>
  );
}
