import { usePermission } from "@/hooks/use-permission";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SendIcon } from "lucide-react";
import { lazy, useState } from "react";

const BulkBillingTransferDialog = lazy(() =>
  import("./bulk-billing-transfer-dialog").then((module) => ({
    default: module.BulkBillingTransferDialog,
  })),
);

/**
 * The header entry point. Transferring moves a shipment and can mark it Ready to
 * Invoice, which the server authorizes as a shipment update, so the button only
 * shows for people who hold that permission.
 */
export function BulkBillingTransferAction() {
  const t = useT();
  const { allowed } = usePermission(Resource.Shipment, Operation.Update);
  const [open, setOpen] = useState(false);

  if (!allowed) return null;

  return (
    <>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <SendIcon className="size-3.5" />
        {t("Transfer to Billing")}
      </Button>
      {open ? (
        <LazyComponent>
          <BulkBillingTransferDialog open={open} onOpenChange={setOpen} />
        </LazyComponent>
      ) : null}
    </>
  );
}
