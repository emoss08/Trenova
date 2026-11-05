import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { billingQueueService } from "@/services/billing-queue";
import type { BillingQueueTarget } from "./billing-queue-assign-dialog";
import { useCallback, useState } from "react";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";
import type { StartBulkRun } from "./use-bulk-run";

export function BillingQueueCancelDialog({
  open,
  onOpenChange,
  targets,
  run,
  onDone,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targets: readonly BillingQueueTarget[];
  /** Starts the bulk run that cancels several items; one item is canceled directly. */
  run?: StartBulkRun;
  onDone?: () => void;
}) {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [reason, setReason] = useState("");

  const single = targets.length === 1 || !run;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setReason("");
  }, [onOpenChange]);

  const { mutate, isPending } = useApiMutation({
    mutationFn: async () => {
      const cancelReason = reason.trim();
      if (single) {
        await Promise.all(
          targets.map((target) =>
            billingQueueService.updateStatus(target.id, {
              status: "Canceled",
              cancelReason,
            }),
          ),
        );
        return;
      }
      await run(
        "Cancel",
        targets.map((target) => target.id),
        { reason: cancelReason },
      );
    },
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      onDone?.();
      handleClose();
    },
    onSettled: invalidate,
  });

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {targets.length === 1
              ? t("Cancel billing queue item")
              : t("Cancel {0} billing queue items", targets.length)}
          </DialogTitle>
          <DialogDescription>
            {targets.length === 1
              ? t("This item will be removed from the billing queue.")
              : t("These items will be removed from the billing queue.")}
          </DialogDescription>
        </DialogHeader>
        <Textarea
          placeholder={t("Reason for cancellation...")}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={3}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Close")}
          </Button>
          <Button
            variant="destructive"
            onClick={() => mutate()}
            disabled={!reason.trim() || isPending}
            isLoading={isPending}
            loadingText={t("Canceling...")}
          >
            {targets.length === 1 ? t("Cancel item") : t("Cancel items")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
