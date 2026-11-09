import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { draftFactoringNoa, type FactoringNoa } from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

const MAX_NOA_NOTES = 2000;

const draftSchema = z.object({
  customerId: z.string().min(1, { error: () => translate("Customer is required") }),
  notes: z
    .string()
    .max(MAX_NOA_NOTES, { error: () => translate("Notes must be at most 2,000 characters") }),
});
type DraftValues = z.infer<typeof draftSchema>;

const DRAFT_FORM_ID = "draft-factoring-noa-form";

export function DraftNoticeDialog({
  factoringCompanyId,
  customerId = null,
  open,
  onOpenChange,
  onDrafted,
}: {
  factoringCompanyId: string;
  /** The customer the notice is for, when the dialog opens from that customer's row. */
  customerId?: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDrafted: (noa: FactoringNoa) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<DraftValues>({
    resolver: zodResolver(draftSchema) as Resolver<DraftValues>,
    defaultValues: { customerId: "", notes: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (open) reset({ customerId: customerId ?? "", notes: "" });
  }, [open, customerId, reset]);

  const mutation = useApiMutation<FactoringNoa, DraftValues, unknown, DraftValues>({
    form,
    resourceName: "notice of assignment",
    mutationFn: (values) =>
      draftFactoringNoa({
        customerId: values.customerId,
        factoringCompanyId,
        notes: values.notes.trim() === "" ? null : values.notes.trim(),
      }),
    onSuccess: (noa) => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
      onDrafted(noa);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("New notice")}</DialogTitle>
          <DialogDescription>
            {t(
              "Drafts the letter telling a customer to pay this factor. Review it, then send it for the customer to sign.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={DRAFT_FORM_ID}
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <CustomerAutocompleteField
                  control={form.control}
                  name="customerId"
                  label={t("Customer")}
                  placeholder={t("Select a customer")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <TextareaField control={form.control} name="notes" label={t("Notes")} rows={2} />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={DRAFT_FORM_ID} isLoading={mutation.isPending}>
            {t("Save draft")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
