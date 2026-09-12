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
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { toast } from "sonner";

export function BillingQueueCancelDialog({
  open,
  onOpenChange,
  itemId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemId: string;
}) {
  const t = useT();

  const [reason, setReason] = useState("");
  const queryClient = useQueryClient();

  const { mutate, isPending } = useMutation({
    mutationFn: () =>
      apiService.billingQueueService.updateStatus(itemId, {
        status: "Canceled",
        cancelReason: reason,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      toast.success(t("Billing queue item canceled"));
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
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>{t("Cancel Billing Queue Item")}</DialogTitle>
          <DialogDescription>
            {t("This item will be removed from the billing queue.")}
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
            {t("Cancel Item")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
