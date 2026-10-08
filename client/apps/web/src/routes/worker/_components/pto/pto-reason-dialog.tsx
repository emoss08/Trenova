import { useT } from "@trenova/shared/i18n/use-t";
import { translate } from "@trenova/shared/i18n/runtime";
import { TextareaField, type TextareaPreset } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { notifyBulkOutcome } from "@/lib/bulk-outcome";
import {
  bulkWorkerPTOAction,
  cancelWorkerPTO,
  rejectWorkerPTO,
} from "@/lib/graphql/worker-mutations";
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
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import {
  ptoCancelRequestSchema,
  ptoReasonRequestSchema,
  type PTOBulkActionPayload,
  type PTOReasonRequest,
} from "@trenova/shared/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { bulkPayloadToOutcome } from "./pto-actions";
import { usePTOInvalidation } from "./use-pto-invalidation";

export type PTOReasonDialogMode = "reject" | "cancel";

const REJECTION_PRESETS: TextareaPreset[] = [
  {
    id: "worker-request",
    label: "Worker request",
    description: "PTO rejected at worker's request",
  },
  {
    id: "business-request",
    label: "Business request",
    description: "PTO rejected at business's request",
  },
  {
    id: "coverage",
    label: "Coverage",
    description: "No coverage available for the requested dates",
  },
  {
    id: "other",
    label: "Other",
    description: "Other reason",
  },
];

const CANCELLATION_PRESETS: TextareaPreset[] = [
  {
    id: "worker-request",
    label: "Worker request",
    description: "Cancelled at worker's request",
  },
  {
    id: "schedule-change",
    label: "Schedule change",
    description: "Cancelled due to a schedule change",
  },
  {
    id: "duplicate",
    label: "Duplicate",
    description: "Duplicate of another request",
  },
];

type ModeCopy = {
  title: string;
  description: (count: number) => string;
  confirm: string;
  loading: string;
  presets: TextareaPreset[];
  reasonDescription: string;
};

const MODE_COPY: Record<PTOReasonDialogMode, ModeCopy> = {
  reject: {
    get title() {
      return translate("Reject PTO");
    },
    description: (count) =>
      count === 1
        ? translate("Reject this PTO request and let the worker know why.")
        : translate(
            "Reject {0, plural, one {# PTO request} other {# PTO requests}} and let the workers know why.",
            count,
          ),
    get confirm() {
      return translate("Confirm Rejection");
    },
    get loading() {
      return translate("Rejecting PTO...");
    },
    presets: REJECTION_PRESETS,
    get reasonDescription() {
      return translate("The worker sees this reason in Dash and by SMS.");
    },
  },
  cancel: {
    get title() {
      return translate("Cancel PTO");
    },
    description: (count) =>
      count === 1
        ? translate(
            "Withdraw this PTO request. Approved time off is released back to the schedule.",
          )
        : translate(
            "Withdraw {0, plural, one {# PTO request} other {# PTO requests}}. Approved time off is released back to the schedule.",
            count,
          ),
    get confirm() {
      return translate("Confirm Cancellation");
    },
    get loading() {
      return translate("Cancelling PTO...");
    },
    presets: CANCELLATION_PRESETS,
    get reasonDescription() {
      return translate("Optional. The worker sees this reason in Dash and by SMS.");
    },
  },
};

export type PTOReasonDialogProps = TableSheetProps & {
  ptoIds: string[];
  mode: PTOReasonDialogMode;
  skipped?: number;
  onCompleted?: () => void;
};

export function PTOReasonDialog({
  open,
  onOpenChange,
  ptoIds,
  mode,
  skipped = 0,
  onCompleted,
}: PTOReasonDialogProps) {
  const t = useT();

  const invalidate = usePTOInvalidation();
  const copy = MODE_COPY[mode];
  const schema = mode === "cancel" ? ptoCancelRequestSchema : ptoReasonRequestSchema;

  const form = useForm<PTOReasonRequest>({
    resolver: zodResolver(schema) as Resolver<PTOReasonRequest>,
    defaultValues: { ptoIds, reason: "" },
  });

  const {
    control,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    reset({ ptoIds, reason: "" });
  }, [ptoIds, reset]);

  const { mutateAsync } = useApiMutation<
    PTOBulkActionPayload | null,
    PTOReasonRequest,
    unknown,
    PTOReasonRequest
  >({
    form,
    resourceName: "PTO",
    mutationFn: async (values) => {
      const reason = values.reason.trim();
      if (values.ptoIds.length === 1) {
        const [id] = values.ptoIds;
        if (mode === "reject") {
          await rejectWorkerPTO(id, reason);
        } else {
          await cancelWorkerPTO(id, reason);
        }
        return null;
      }
      return bulkWorkerPTOAction({
        ptoIds: values.ptoIds,
        action: mode === "reject" ? "Reject" : "Cancel",
        reason: reason.length > 0 ? reason : null,
      });
    },
    onSuccess: (payload) => {
      if (payload) {
        notifyBulkOutcome(
          bulkPayloadToOutcome(payload),
          mode === "reject"
            ? {
                succeeded: (count) =>
                  t("Rejected {0, plural, one {# PTO request} other {# PTO requests}}", count),
                partial: (succeeded, failed) =>
                  t(
                    "Rejected {0, plural, one {# PTO request} other {# PTO requests}}; {1} failed",
                    succeeded,
                    failed,
                  ),
                allFailed: (failed) =>
                  t(
                    "All {0, plural, one {# selected PTO request} other {# selected PTO requests}} failed",
                    failed,
                  ),
                skipped,
              }
            : {
                succeeded: (count) =>
                  t("Cancelled {0, plural, one {# PTO request} other {# PTO requests}}", count),
                partial: (succeeded, failed) =>
                  t(
                    "Cancelled {0, plural, one {# PTO request} other {# PTO requests}}; {1} failed",
                    succeeded,
                    failed,
                  ),
                allFailed: (failed) =>
                  t(
                    "All {0, plural, one {# selected PTO request} other {# selected PTO requests}} failed",
                    failed,
                  ),
                skipped,
              },
        );
      } else {
        toast.success(mode === "reject" ? t("PTO rejected") : t("PTO cancelled"), {
          description:
            skipped > 0
              ? t(
                  "The worker has been notified. {0, plural, one {# ineligible skipped.} other {# ineligible skipped.}}",
                  skipped,
                )
              : t("The worker has been notified."),
        });
      }
      void invalidate();
      reset({ ptoIds: [], reason: "" });
      onOpenChange(false);
      onCompleted?.();
    },
  });

  const onSubmit = useCallback(
    async (values: PTOReasonRequest) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>
            {copy.description(ptoIds.length)}
            {skipped > 0
              ? ` ${t("{0} selected request{1} not eligible and will be skipped.", skipped, skipped === 1 ? " is" : t("s are"))}`
              : ""}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              void handleSubmit(onSubmit)(e);
            }}
          >
            <FormGroup cols={1}>
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  rules={{ required: mode === "reject" }}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("e.g. No coverage for those dates")}
                  description={copy.reasonDescription}
                  presets={copy.presets}
                  maxLength={255}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Back")}
              </Button>
              <Button
                type="button"
                onClick={() => void handleSubmit(onSubmit)()}
                variant="destructive"
                isLoading={isSubmitting}
                loadingText={copy.loading}
              >
                {copy.confirm}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
