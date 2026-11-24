import { SelectField } from "@/components/fields/select-field";
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
import type { SelectOption } from "@trenova/shared/types/fields";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, type ReactNode } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { z } from "zod";

export const MAX_REASON_LENGTH = 500;

/** A fixed set the person picks why from, before the words that go with it. */
export type ReasonChoice = {
  label: string;
  placeholder: string;
  options: SelectOption[];
  /** The picks that leave the words required even when the dialog is optional. */
  needsWords?: string[];
};

function reasonSchema(maxLength: number, optional: boolean, choice?: ReasonChoice) {
  return z
    .object({
      choice: z.string(),
      reason: z
        .string()
        .trim()
        .min(optional ? 0 : 1, { error: () => translate("Say why") })
        .max(maxLength, {
          error: () => translate("Keep the reason under {0} characters", maxLength),
        }),
    })
    .superRefine((values, ctx) => {
      if (!choice) return;
      if (values.choice === "") {
        ctx.addIssue({ code: "custom", path: ["choice"], message: translate("Pick one") });
        return;
      }
      if (choice.needsWords?.includes(values.choice) && values.reason.trim() === "") {
        ctx.addIssue({ code: "custom", path: ["reason"], message: translate("Say why") });
      }
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
  /** A fixed set to pick why from, above the words; the pick is passed to onConfirm. */
  choice?: ReasonChoice;
  pending: boolean;
  /** Called with the words, and with the pick when the dialog offers a choice. */
  onConfirm: (reason: string, choice?: string) => void;
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
  choice,
  pending,
  onConfirm,
  onCancel,
}: Omit<ReasonDialogProps, "open" | "onOpenChange"> & { onCancel: () => void }) {
  const t = useT();
  const formId = useId();
  const form = useForm<ReasonValues>({
    resolver: zodResolver(reasonSchema(maxLength, optional, choice)) as Resolver<ReasonValues>,
    defaultValues: { choice: "", reason: defaultReason },
  });
  const picked = useWatch({ control: form.control, name: "choice" });
  const wordsRequired = !optional || Boolean(choice?.needsWords?.includes(picked));

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
            void form.handleSubmit((values) =>
              choice
                ? onConfirm(values.reason.trim(), values.choice)
                : onConfirm(values.reason.trim()),
            )(event);
          }}
        >
          <FormGroup cols={1}>
            {choice && (
              <FormControl>
                <SelectField<ReasonValues>
                  control={form.control}
                  name="choice"
                  label={choice.label}
                  placeholder={choice.placeholder}
                  options={choice.options}
                  required
                />
              </FormControl>
            )}
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={label ?? t("Reason")}
                description={fieldDescription}
                placeholder={placeholder}
                rows={3}
                required={wordsRequired}
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
