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
import { useEffect, useId } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

export const MAX_REMITTANCE_REASON = 500;

const reasonSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why") })
    .max(MAX_REMITTANCE_REASON, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
});
type ReasonValues = z.infer<typeof reasonSchema>;

type ReasonDialogProps = {
  open: boolean;
  title: string;
  description: string;
  placeholder: string;
  confirmLabel: string;
  destructive?: boolean;
  pending: boolean;
  onConfirm: (reason: string) => void;
  onOpenChange: (open: boolean) => void;
};

/** Asks why before a line is ignored or a remittance discarded. */
export function ReasonDialog({
  open,
  title,
  description,
  placeholder,
  confirmLabel,
  destructive = false,
  pending,
  onConfirm,
  onOpenChange,
}: ReasonDialogProps) {
  const t = useT();
  const formId = useId();
  const form = useForm<ReasonValues>({
    resolver: zodResolver(reasonSchema) as Resolver<ReasonValues>,
    defaultValues: { reason: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
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
                  label={t("Reason")}
                  placeholder={placeholder}
                  rows={3}
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Back")}
          </Button>
          <Button
            type="submit"
            form={formId}
            variant={destructive ? "destructive" : "default"}
            isLoading={pending}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
