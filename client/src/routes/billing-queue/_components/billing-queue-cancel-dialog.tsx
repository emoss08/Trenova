import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
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
      toast.success("Billing queue item canceled");
      handleClose();
    },
    onError: () => {
      toast.error("Failed to cancel item");
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
          <DialogTitle>Cancel Billing Queue Item</DialogTitle>
          <DialogDescription>
            This item will be removed from the billing queue.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          placeholder="Reason for cancellation..."
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={3}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Close
          </Button>
          <Button
            variant="destructive"
            onClick={() => mutate()}
            disabled={!reason.trim() || isPending}
            isLoading={isPending}
            loadingText="Canceling..."
          >
            Cancel Item
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
