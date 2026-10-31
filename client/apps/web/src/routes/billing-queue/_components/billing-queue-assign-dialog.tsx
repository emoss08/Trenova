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
import {
  billingQueueAssignSchema,
  type BillingQueueAssignInput,
} from "@trenova/shared/types/billing-queue";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import type { StartBulkRun } from "./use-bulk-run";

/** One queue item to assign, with the label a failure is reported under. */
export type BillingQueueTarget = {
  id: string;
  label: string;
};

type BillingQueueAssignDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targets: readonly BillingQueueTarget[];
  /** Starts the bulk run that assigns several items; one item is assigned directly. */
  run?: StartBulkRun;
  onDone?: () => void;
};

export function BillingQueueAssignDialog({
  open,
  onOpenChange,
  targets,
  run,
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

  const single = targets.length === 1 || !run;
  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: BillingQueueAssignInput) => {
      if (single) {
        await Promise.all(
          targets.map((target) => apiService.billingQueueService.assign(target.id, values)),
        );
        return;
      }
      await run(
        "Assign",
        targets.map((target) => target.id),
        { billerId: values.billerId },
      );
    },
    resourceName: "BillingQueueItem",
    form,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      onDone?.();
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
