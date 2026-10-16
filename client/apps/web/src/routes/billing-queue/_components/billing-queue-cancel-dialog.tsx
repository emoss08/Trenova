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
import { apiService } from "@/services/api";
import { BULK_CONCURRENCY, notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import type { BillingQueueTarget } from "./billing-queue-assign-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";

export function BillingQueueCancelDialog({
  open,
  onOpenChange,
  targets,
  onDone,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targets: readonly BillingQueueTarget[];
  onDone?: () => void;
}) {
  const t = useT();

  const [reason, setReason] = useState("");
  const queryClient = useQueryClient();

  const single = targets.length === 1;
  const { mutate, isPending } = useMutation({
    mutationFn: () =>
      settleAll(
        targets,
        (target) =>
          apiService.billingQueueService.updateStatus(target.id, {
            status: "Canceled",
            cancelReason: reason.trim(),
          }),
        { concurrency: BULK_CONCURRENCY, label: (target) => target.label },
      ),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      notifyBulkOutcome(result, {
        succeeded: (count) =>
          single
            ? t("Billing queue item canceled")
            : t("{0, plural, one {# item} other {# items}} canceled", count),
        partial: (succeeded, failed) =>
          t("{0} canceled; {1} could not be canceled", succeeded, failed),
        allFailed: (failed) =>
          single
            ? t("Failed to cancel item")
            : t("{0, plural, one {# item} other {# items}} could not be canceled", failed),
      });
      if (result.failed.length === 0) {
        onDone?.();
      }
      handleClose();
    },
    onError: () => {
      toast.error(t("Failed to cancel item"));
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setReason("");
  }, [onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {single
              ? t("Cancel billing queue item")
              : t("Cancel {0} billing queue items", targets.length)}
          </DialogTitle>
          <DialogDescription>
            {single
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
            {single ? t("Cancel item") : t("Cancel items")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
