import { useApiMutation } from "@/hooks/use-api-mutation";
import { useQueryClient } from "@tanstack/react-query";
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
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { api } from "@trenova/shared/lib/api";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import type { API_ENDPOINTS } from "@trenova/shared/types/server";
import { useEffect, useId } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { FormChangesFooter, type ChangeFields } from "./form-changes";

type FormCreateModalProps<TFieldValues extends FieldValues, TSubmitValues, TResponse> = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The dialog's whole title, such as "New agreement". */
  title: string;
  description?: string;
  /** What is being created, for error messages ("factoring agreement"). */
  resourceName: string;
  form: UseFormReturn<TFieldValues, any, TSubmitValues>;
  formComponent: React.ReactNode;
  size?: React.ComponentProps<typeof DialogContent>["size"];
  notice?: React.ReactNode;
  /** Labels for the change review in the footer. */
  changeFields?: ChangeFields;
  /** A REST endpoint to POST the values to, when there is no `mutationFn`. */
  url?: API_ENDPOINTS;
  mutationFn?: (values: TSubmitValues) => Promise<TResponse>;
  /** Query keys whose reads the new record changes. */
  invalidates?: readonly (readonly unknown[])[];
  onCreated?: (data: TResponse, values: TSubmitValues) => void | Promise<void>;
  submitText?: string;
};

/**
 * The dialog a record is created in: a title, the form, and a footer that counts the changes and
 * holds Cancel and the save. The save holds until the server answers; a field error lands on its
 * field; Cmd/Ctrl+Enter saves. Only the form scrolls, so the title and the save stay in view.
 */
export function FormCreateModal<
  TFieldValues extends FieldValues,
  TSubmitValues = TFieldValues,
  TResponse = unknown,
>({
  open,
  onOpenChange,
  title,
  description,
  resourceName,
  form,
  formComponent,
  size,
  notice,
  changeFields = {},
  url,
  mutationFn,
  invalidates = [],
  onCreated,
  submitText,
}: FormCreateModalProps<TFieldValues, TSubmitValues, TResponse>) {
  const t = useT();
  const formId = useId();
  const queryClient = useQueryClient();
  const {
    formState: { isSubmitting },
    handleSubmit,
  } = form;

  const { mutateAsync } = useApiMutation<TResponse, TSubmitValues, unknown, TFieldValues>({
    form,
    resourceName,
    mutationFn: async (values) => {
      if (mutationFn) return mutationFn(values);
      if (!url) throw new Error("FormCreateModal needs a url or a mutationFn");
      return api.post<TResponse>(url, values);
    },
    onSuccess: async (data, values) => {
      await Promise.all(
        invalidates.map((queryKey) => queryClient.invalidateQueries({ queryKey: [...queryKey] })),
      );
      onOpenChange(false);
      await onCreated?.(data, values);
    },
  });

  const submit = handleSubmit(async (values) => {
    await mutateAsync(values);
  });

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void submit();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open, isSubmitting, submit]);

  return (
    <Dialog open={open} onOpenChange={(next) => !isSubmitting && onOpenChange(next)}>
      <DialogContent
        size={size}
        className="max-h-[calc(100dvh-4rem)] grid-rows-[auto_minmax(0,1fr)_auto] gap-0"
      >
        <DialogHeader className="pb-4">
          <DialogTitle>{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        <FormProvider {...form}>
          <ScrollArea className="-mx-4 min-h-0">
            <div className="flex flex-col gap-4 px-4 pb-4">
              {notice}
              <Form id={formId} onSubmit={submit} className="flex flex-col gap-6">
                {formComponent}
              </Form>
            </div>
          </ScrollArea>
          <DialogFooter className="items-center sm:justify-between">
            <FormChangesFooter form={form} fields={changeFields} create />
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={isSubmitting}
                onClick={() => onOpenChange(false)}
              >
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                form={formId}
                isLoading={isSubmitting}
                title={formatShortcut("Enter")}
              >
                {submitText ?? t("Create")}
              </Button>
            </div>
          </DialogFooter>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
