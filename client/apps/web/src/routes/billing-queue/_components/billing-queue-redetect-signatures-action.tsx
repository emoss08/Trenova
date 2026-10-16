import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { apiService } from "@/services/api";
import { ScanSearchIcon } from "@trenova/shared/components/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PODSignatureRedetectResult } from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";

export function podSignatureRedetectSummary(
  t: TranslateFn,
  result: PODSignatureRedetectResult,
): string {
  if (result.queued === 0) {
    return t("No queued POD is waiting for its signature to be read");
  }

  return t(
    "{0, plural, one {Reading # POD again} other {Reading # PODs again}}. Each billing item updates when its POD is read.",
    result.queued,
  );
}

export function BillingQueueRedetectSignaturesAction() {
  const t = useT();
  const [open, setOpen] = useState(false);
  const { allowed: canUpdateQueue } = usePermission(Resource.BillingQueue, Operation.Update);
  const { allowed: canUpdateDocuments } = usePermission(Resource.Document, Operation.Update);

  const { mutate, isPending } = useApiMutation<PODSignatureRedetectResult, void>({
    resourceName: "Billing queue",
    mutationFn: () => apiService.billingQueueService.redetectPODSignatures(),
    onSuccess: (result) => {
      setOpen(false);
      toast.success(podSignatureRedetectSummary(t, result));
    },
  });

  if (!canUpdateQueue || !canUpdateDocuments) return null;

  return (
    <>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <ScanSearchIcon className="size-3.5" />
        {t("Read POD signatures")}
      </Button>
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Read POD signatures again?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Document intelligence reads every queued proof of delivery whose signature nobody has read yet and marks it signed or unsigned. PODs a person already marked, and PODs uploaded without machine reading, are left as they are.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              size="default"
              onClick={() => mutate()}
              disabled={isPending}
              isLoading={isPending}
            >
              {t("Read signatures")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
