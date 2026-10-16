import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { BULK_CONCURRENCY, notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  billingQueueAssignSchema,
  type BillingQueueAssignInput,
} from "@trenova/shared/types/billing-queue";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";

/** One queue item to assign, with the label a failure is reported under. */
export type BillingQueueTarget = {
  id: string;
  label: string;
};

type BillingQueueAssignDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targets: readonly BillingQueueTarget[];
  onDone?: () => void;
};

export function BillingQueueAssignDialog({
  open,
  onOpenChange,
  targets,
  onDone,
}: BillingQueueAssignDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();

  const form = useForm<BillingQueueAssignInput>({
    resolver: zodResolver(billingQueueAssignSchema),
    defaultValues: {
      billerId: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const single = targets.length === 1;
  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: BillingQueueAssignInput) => {
      if (single) {
        await apiService.billingQueueService.assign(targets[0].id, values);
        return null;
      }
      return settleAll(
        targets,
        (target) => apiService.billingQueueService.assign(target.id, values),
        { concurrency: BULK_CONCURRENCY, label: (target) => target.label },
      );
    },
    resourceName: "BillingQueueItem",
    form,
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      if (!result) {
        return;
      }
      notifyBulkOutcome(result, {
        succeeded: (count) => t("{0, plural, one {# item} other {# items}} assigned", count),
        partial: (succeeded, failed) =>
          t("{0} assigned; {1} could not be assigned", succeeded, failed),
        allFailed: (failed) =>
          t("{0, plural, one {# item} other {# items}} could not be assigned", failed),
      });
      if (result.failed.length === 0) {
        onDone?.();
      }
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ billerId: "" });
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: BillingQueueAssignInput) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Assign biller")}</DialogTitle>
          <DialogDescription>
            {single
              ? t("Assign a biller to review this billing queue item.")
              : t(
                  "Assign a biller to review {0, plural, one {# billing queue item} other {# billing queue items}}.",
                  targets.length,
                )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <UserAutocompleteField
                control={control}
                name="billerId"
                label={t("Biller")}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Assigning...")}>
              {t("Assign")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
