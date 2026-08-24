import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const cancelReasonSchema = z.object({
  reason: z.string().min(1, { error: "A cancellation reason is required" }),
});

type CancelReasonValues = z.infer<typeof cancelReasonSchema>;

/**
 * The reason prompt for pulling a move back from an external carrier. The domain refuses
 * a cancellation without a recorded reason, so the dialog does too.
 */
export function CarrierAssignmentCancelDialog({
  open,
  onOpenChange,
  carrierName,
  isSubmitting,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  carrierName?: string | null;
  isSubmitting: boolean;
  onConfirm: (reason: string) => void;
}) {
  const form = useForm<CancelReasonValues>({
    resolver: zodResolver(cancelReasonSchema),
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ reason: "" });
  }, [onOpenChange, reset]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle>Cancel Carrier Assignment</DialogTitle>
          <DialogDescription>
            {carrierName
              ? `Pull this move back from ${carrierName}. The move returns to uncovered and the cancellation reason is recorded.`
              : "Pull this move back from the carrier. The move returns to uncovered and the cancellation reason is recorded."}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => onConfirm(values.reason))(event);
          }}
        >
          <div className="pb-4">
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              rules={{ required: true }}
              placeholder="e.g., Carrier failed to dispatch a truck"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Keep assignment
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={isSubmitting}
              loadingText="Canceling..."
            >
              Cancel assignment
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
