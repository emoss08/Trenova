import { TextareaField } from "@/components/fields/textarea-field";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useId, type ReactNode } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

export const MAX_REASON_LENGTH = 500;

function reasonSchema(maxLength: number, optional: boolean) {
  return z.object({
    reason: z
      .string()
      .trim()
      .min(optional ? 0 : 1, { error: () => translate("Say why") })
      .max(maxLength, {
        error: () => translate("Keep the reason under {0} characters", maxLength),
      }),
  });
}
type ReasonValues = z.infer<ReturnType<typeof reasonSchema>>;

type ReasonDialogProps = {
  open: boolean;
  title: string;
  description: string;
  placeholder: string;
  confirmLabel: string;
  destructive?: boolean;
  /** The longest reason the server keeps; 500 characters unless the record allows another. */
  maxLength?: number;
  /** When true the person may confirm without writing anything. */
  optional?: boolean;
  /** The field's label; "Reason" unless the words are a note rather than a reason. */
  label?: string;
  /** Help under the field, such as who will read the reason. */
  fieldDescription?: ReactNode;
  /** What the field holds when the dialog opens, such as the record's current note. */
  defaultReason?: string;
  /** The words on the button that backs out; "Back" unless the step needs its own words. */
  cancelLabel?: string;
  pending: boolean;
  onConfirm: (reason: string) => void;
  onOpenChange: (open: boolean) => void;
};

/**
 * Asks why before a step that needs a reason: a remittance line ignored, a
 * manual journal rejected or cancelled. With `optional` it asks for a note
 * the person may leave empty, such as what they tell an agent when declining
 * its plan.
 *
 * The form lives inside the dialog's content, which is mounted only while the
 * dialog is open, so every opening starts from `defaultReason` and nothing that
 * refetches while it is open can replace what the person is typing.
 */
export function ReasonDialog({ open, onOpenChange, ...props }: ReasonDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <ReasonForm {...props} onCancel={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

function ReasonForm({
  title,
  description,
  placeholder,
  confirmLabel,
  destructive = false,
  maxLength = MAX_REASON_LENGTH,
  optional = false,
  label,
  fieldDescription,
  defaultReason = "",
  cancelLabel,
  pending,
  onConfirm,
  onCancel,
}: Omit<ReasonDialogProps, "open" | "onOpenChange"> & { onCancel: () => void }) {
  const t = useT();
  const formId = useId();
  const form = useForm<ReasonValues>({
    resolver: zodResolver(reasonSchema(maxLength, optional)) as Resolver<ReasonValues>,
    defaultValues: { reason: defaultReason },
  });

  return (
    <>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>{description}</DialogDescription>
      </DialogHeader>
      <FormProvider {...form}>
        <Form
          id={formId}
          onSubmit={(event) => {
            event.stopPropagation();
            void form.handleSubmit((values) => onConfirm(values.reason.trim()))(event);
          }}
        >
          <FormGroup cols={1}>
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={label ?? t("Reason")}
                description={fieldDescription}
                placeholder={placeholder}
                rows={3}
                required={!optional}
              />
            </FormControl>
          </FormGroup>
        </Form>
      </FormProvider>
      <DialogFooter>
        <Button type="button" size="sm" variant="outline" onClick={onCancel}>
          {cancelLabel ?? t("Back")}
        </Button>
        <Button
          type="submit"
          size="sm"
          form={formId}
          variant={destructive ? "destructive" : "default"}
          isLoading={pending}
        >
          {confirmLabel}
        </Button>
      </DialogFooter>
    </>
  );
}
