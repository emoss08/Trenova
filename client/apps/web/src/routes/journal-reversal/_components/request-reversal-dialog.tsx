import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useId } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { JournalReversalForm } from "./journal-reversal-form";
import {
  reversalRequestBody,
  todayField,
  type ReversalRequestValues,
} from "./journal-reversal-request";

/**
 * Asks for a posted journal entry to be reversed, from the entry itself: the
 * day the mirror entry should post and why. The request waits for someone who
 * may approve reversals, as one made on the reversals page does.
 */
export function RequestReversalDialog({
  entryId,
  entryNumber,
  open,
  onOpenChange,
  onDone,
}: {
  entryId: string;
  entryNumber: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDone: () => void;
}) {
  const t = useT();
  const formId = useId();
  const form = useForm<ReversalRequestValues>({
    defaultValues: {
      originalJournalEntryId: entryId,
      requestedAccountingDate: todayField(),
      reasonCode: "",
      reasonText: "",
    },
  });
  const { reset, handleSubmit } = form;

  useEffect(() => {
    if (open) {
      reset({
        originalJournalEntryId: entryId,
        requestedAccountingDate: todayField(),
        reasonCode: "",
        reasonText: "",
      });
    }
  }, [open, entryId, reset]);

  const request = useApiMutation({
    mutationFn: async (values: ReversalRequestValues) =>
      apiService.journalReversalService.create(reversalRequestBody(values)),
    onSuccess: () => {
      onOpenChange(false);
      onDone();
    },
    form,
    resourceName: "Journal Reversal",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Request a reversal of {0}", entryNumber)}</DialogTitle>
          <DialogDescription>
            {t("The entry stays in the ledger as it was posted; the reversal takes it back.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={formId}
            onSubmit={(event) => {
              event.stopPropagation();
              void handleSubmit((values) => request.mutateAsync(values))(event);
            }}
          >
            <JournalReversalForm entryNumber={entryNumber} />
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Back")}
          </Button>
          <Button type="submit" form={formId} isLoading={request.isPending}>
            {t("Request reversal")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
