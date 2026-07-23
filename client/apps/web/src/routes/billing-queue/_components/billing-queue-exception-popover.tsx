import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { updateBillingQueueStatusGraphQL } from "@/lib/graphql/billing-queue";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { exceptionReasonLabels } from "@/lib/choices";
import type { SelectOption } from "@trenova/shared/types/fields";
import {
  exceptionReasonCodeSchema,
  type BillingQueueItem,
  type ExceptionReasonCode,
} from "@trenova/shared/types/billing-queue";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

type ExceptionTargetStatus = Extract<
  BillingQueueItem["status"],
  "Exception" | "SentBackToOps"
>;

type ExceptionFormValues = {
  exceptionReasonCode: ExceptionReasonCode | "";
  exceptionNotes: string;
};

const REASON_OPTIONS: SelectOption[] = exceptionReasonCodeSchema.options.map((code) => ({
  label: exceptionReasonLabels[code],
  value: code,
}));

function buildExceptionSchema(targetStatus: ExceptionTargetStatus) {
  return z
    .object({
      exceptionReasonCode: exceptionReasonCodeSchema.or(z.literal("")),
      exceptionNotes: z.string(),
    })
    .superRefine((values, ctx) => {
      if (values.exceptionReasonCode === "") {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["exceptionReasonCode"],
          message: "Exception reason is required",
        });
      }

      const notesRequired =
        targetStatus === "Exception" || values.exceptionReasonCode === "Other";

      if (notesRequired && values.exceptionNotes.trim() === "") {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["exceptionNotes"],
          message: "Exception notes are required",
        });
      }
    });
}

type BillingQueueExceptionPopoverProps = {
  itemId: BillingQueueItem["id"];
  targetStatus: ExceptionTargetStatus;
  label: string;
  icon: React.ReactNode;
  variant?: "outline" | "destructive";
  disabled?: boolean;
  successMessage: string;
  onSuccess: () => void;
};

export function BillingQueueExceptionPopover({
  itemId,
  targetStatus,
  label,
  icon,
  variant = "outline",
  disabled,
  successMessage,
  onSuccess,
}: BillingQueueExceptionPopoverProps) {
  const [open, setOpen] = useState(false);

  const schema = useMemo(() => buildExceptionSchema(targetStatus), [targetStatus]);

  const form = useForm<ExceptionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      exceptionReasonCode: "",
      exceptionNotes: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: ExceptionFormValues) =>
      updateBillingQueueStatusGraphQL(itemId, {
        status: targetStatus,
        exceptionReasonCode: values.exceptionReasonCode || null,
        exceptionNotes: values.exceptionNotes.trim() || null,
      }),
    resourceName: "BillingQueueItem",
    setFormError: setError,
    onSuccess: () => {
      onSuccess();
      toast.success(successMessage);
    },
  });

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      setOpen(nextOpen);
      if (!nextOpen) {
        reset({ exceptionReasonCode: "", exceptionNotes: "" });
      }
    },
    [reset],
  );

  const onSubmit = useCallback(
    async (values: ExceptionFormValues) => {
      await mutateAsync(values);
      handleOpenChange(false);
    },
    [mutateAsync, handleOpenChange],
  );

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button size="sm" variant={variant} disabled={disabled}>
            {icon}
            {label}
          </Button>
        }
      />
      <PopoverContent className="w-80" align="start">
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <p className="mb-2 text-sm font-medium">{label}</p>
          <FormGroup cols={1}>
            <FormControl>
              <SelectField
                control={control}
                name="exceptionReasonCode"
                label="Reason"
                placeholder="Select reason..."
                options={REASON_OPTIONS}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={control}
                name="exceptionNotes"
                label="Notes"
                placeholder="Add notes..."
                rules={{
                  required: targetStatus === "Exception",
                }}
              />
            </FormControl>
          </FormGroup>
          <div className="mt-3 flex justify-end">
            <Button
              type="submit"
              size="sm"
              isLoading={isSubmitting}
              loadingText="Submitting..."
            >
              Submit
            </Button>
          </div>
        </Form>
      </PopoverContent>
    </Popover>
  );
}
