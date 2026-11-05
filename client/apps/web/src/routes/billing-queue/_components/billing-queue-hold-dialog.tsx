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
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { HOLD_REASONS, holdReasonLabel } from "@/lib/billing-queue-checks";
import { billingQueueService } from "@/services/billing-queue";
import type { BillingQueueHoldReason } from "@trenova/shared/types/billing-queue";
import { useState } from "react";
import type { BillingQueueTarget } from "./billing-queue-assign-dialog";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";
import type { StartBulkRun } from "./use-bulk-run";

/**
 * Sets items aside with the reason the queue reports them under. Several items go
 * through one bulk run, behind its undo window; one item is held directly.
 */
export function BillingQueueHoldDialog({
  targets,
  run,
  defaultNote = "",
  onOpenChange,
  onDone,
}: {
  targets: readonly BillingQueueTarget[];
  run?: StartBulkRun;
  defaultNote?: string;
  onOpenChange: (open: boolean) => void;
  onDone?: () => void;
}) {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [reason, setReason] = useState<BillingQueueHoldReason | null>(null);
  const [note, setNote] = useState(defaultNote);

  const reasonItems = HOLD_REASONS.map((value) => ({
    value,
    label: holdReasonLabel(value, t),
  }));

  const hold = useApiMutation({
    mutationFn: async (holdReasonCode: BillingQueueHoldReason) => {
      const reviewNotes = note.trim();
      if (targets.length === 1 || !run) {
        await Promise.all(
          targets.map((target) =>
            billingQueueService.updateStatus(target.id, {
              status: "OnHold",
              holdReasonCode,
              reviewNotes: reviewNotes || undefined,
            }),
          ),
        );
        return;
      }
      await run(
        "Hold",
        targets.map((target) => target.id),
        { holdReasonCode, reason: reviewNotes },
      );
    },
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      onDone?.();
      onOpenChange(false);
    },
    onSettled: invalidate,
  });

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>
            {t("{0, plural, one {Hold # item} other {Hold # items}}", targets.length)}
          </DialogTitle>
          <DialogDescription>
            {t("Held items leave the review lanes until someone releases them.")}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div className="grid gap-1.5">
            <Label>{t("Reason")}</Label>
            <Select
              value={reason}
              items={reasonItems}
              onValueChange={(value) => setReason(value as BillingQueueHoldReason)}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("Why it's on hold")} />
              </SelectTrigger>
              <SelectContent>
                {reasonItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label>{t("Note")}</Label>
            <Textarea
              placeholder={t("What it waits on (optional)")}
              value={note}
              onChange={(event) => setNote(event.target.value)}
              rows={3}
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            disabled={!reason || hold.isPending}
            isLoading={hold.isPending}
            loadingText={t("Holding...")}
            onClick={() => reason && hold.mutate(reason)}
          >
            {t("Hold")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
