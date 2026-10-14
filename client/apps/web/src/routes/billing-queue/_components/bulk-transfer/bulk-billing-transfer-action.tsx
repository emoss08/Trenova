import { usePermission } from "@/hooks/use-permission";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Send01Icon } from "@trenova/shared/components/icons";
import { lazyComponent } from "@/lib/lazy-component";
import { BulkBillingTransferDialogSkeleton } from "./bulk-billing-transfer-dialog-skeleton";

// Read when the pointer or focus reaches the button, so by the click the dialog is usually here.
const transferDialog = lazyComponent(
  () => import("./bulk-billing-transfer-dialog"),
  (module) => module.BulkBillingTransferDialog,
);
const BulkBillingTransferDialog = transferDialog.Component;

const preloadTransferDialog = () => void transferDialog.preload().catch(() => undefined);

/**
 * The header entry point. Transferring moves a shipment and can mark it Ready to
 * Invoice, which the server authorizes as a shipment update, so the button only
 * shows for people who hold that permission.
 *
 * Which run the dialog is watching lives in the URL rather than in here, so the
 * toast that announces a finished transfer can link straight back to its report.
 */
export function BulkBillingTransferAction({
  open,
  onOpenChange,
  runId,
  onRunIdChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  runId: string | null;
  onRunIdChange: (runId: string | null) => void;
}) {
  const t = useT();
  const { allowed } = usePermission(Resource.Shipment, Operation.Update);

  if (!allowed) return null;

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        onPointerEnter={preloadTransferDialog}
        onFocus={preloadTransferDialog}
        onClick={() => onOpenChange(true)}
      >
        <Send01Icon className="size-3.5" />
        {t("Transfer to billing")}
      </Button>
      {open ? (
        <LazyComponent
          fallback={
            <BulkBillingTransferDialogSkeleton
              open={open}
              onOpenChange={onOpenChange}
              runId={runId}
            />
          }
        >
          <BulkBillingTransferDialog
            open={open}
            onOpenChange={onOpenChange}
            runId={runId}
            onRunIdChange={onRunIdChange}
          />
        </LazyComponent>
      ) : null}
    </>
  );
}
