import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  cancelFactoringSubmission,
  retryFactoringSubmission,
  type FactoringSubmission,
} from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCw01Icon } from "@trenova/shared/components/icons";
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
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { z } from "zod";

const MAX_CANCEL_REASON = 500;
const CANCEL_FORM_ID = "cancel-factoring-submission-form";

/**
 * Sends a failed schedule again. The server reads the factor's delivery
 * settings as they are now, so a fixed address or host is picked up.
 */
export function RetryFactoringSubmissionButton({
  submission,
}: {
  submission: FactoringSubmission;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const retry = useMutation({
    mutationFn: () => retryFactoringSubmission({ id: submission.id, version: submission.version }),
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
    },
    onError: (error) => {
      invalidateFactoringQueries(queryClient);
      const described = describeError(error);
      toast.error(t("The schedule was not retried"), {
        description: described.detail ?? described.message,
      });
    },
  });

  return (
    <Button
      type="button"
      size="sm"
      isLoading={retry.isPending}
      loadingText={t("Retrying...")}
      onClick={() => retry.mutate()}
    >
      <RefreshCw01Icon className="size-3.5" />
      {t("Retry")}
    </Button>
  );
}

const cancelSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why the schedule is cancelled") })
    .max(MAX_CANCEL_REASON, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
});
type CancelValues = z.infer<typeof cancelSchema>;

/**
 * Cancels a failed schedule. Its invoices are released back to the queue,
 * where they can go out on another schedule; nothing is sent to the factor.
 */
export function CancelFactoringSubmissionDialog({
  submission,
  onOpenChange,
}: {
  submission: FactoringSubmission | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<CancelValues>({
    resolver: zodResolver(cancelSchema) as Resolver<CancelValues>,
    defaultValues: { reason: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (submission) reset({ reason: "" });
  }, [submission, reset]);

  const mutation = useApiMutation<FactoringSubmission, CancelValues, unknown, CancelValues>({
    form,
    resourceName: "factoring schedule",
    mutationFn: (values) => {
      if (!submission) throw new Error("No schedule selected");
      return cancelFactoringSubmission({
        id: submission.id,
        version: submission.version,
        reason: values.reason.trim(),
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={submission !== null} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Cancel schedule")}</DialogTitle>
          <DialogDescription>
            {submission
              ? t(
                  "Schedule {0} was never delivered. Cancelling it puts its {1, plural, one {# invoice} other {# invoices}} back in the queue to send on another schedule.",
                  submission.number,
                  submission.invoiceCount,
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={CANCEL_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why this schedule will not be sent")}
                  rows={3}
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Keep schedule")}
          </Button>
          <Button
            type="submit"
            form={CANCEL_FORM_ID}
            variant="destructive"
            isLoading={mutation.isPending}
          >
            {t("Cancel schedule")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
