import { DateField } from "@/components/fields/date-field/date-field";
import { handleMutationError, useApiMutation } from "@/hooks/use-api-mutation";
import {
  activateFactoringAgreement,
  endFactoringAgreement,
  type FactoringAgreement,
} from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

export function ActivateAgreementDialog({
  agreement,
  onOpenChange,
}: {
  agreement: FactoringAgreement | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!agreement) throw new Error("No agreement selected");
      return activateFactoringAgreement({ id: agreement.id, version: agreement.version });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "factoring agreement" }),
  });

  return (
    <AlertDialog open={agreement !== null} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Activate this agreement?")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Invoices posted to this factor from its start date use these terms, and every open invoice assigned to the factor is checked again.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction isLoading={isPending} onClick={() => mutate()}>
            {t("Activate")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

const END_FORM_ID = "end-factoring-agreement-form";

const endSchema = z.object({ endAt: z.number().nullable() });
type EndValues = z.infer<typeof endSchema>;

export function EndAgreementDialog({
  agreement,
  onOpenChange,
}: {
  agreement: FactoringAgreement | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<EndValues>({
    resolver: zodResolver(endSchema) as Resolver<EndValues>,
    defaultValues: { endAt: null },
  });
  const { reset } = form;

  useEffect(() => {
    if (agreement) reset({ endAt: null });
  }, [agreement, reset]);

  const mutation = useApiMutation<FactoringAgreement, EndValues, unknown, EndValues>({
    form,
    resourceName: "factoring agreement",
    mutationFn: (values) => {
      if (!agreement) throw new Error("No agreement selected");
      return endFactoringAgreement({
        id: agreement.id,
        version: agreement.version,
        endAt: values.endAt,
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={agreement !== null} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("End this agreement?")}</DialogTitle>
          <DialogDescription>
            {t(
              "Invoices posted after it ends are not ready to factor until another agreement is active. Invoices already posted keep the terms they were posted under.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form id={END_FORM_ID} onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
            <FormGroup cols={1}>
              <FormControl>
                <DateField
                  control={form.control}
                  name="endAt"
                  label={t("Ends")}
                  placeholder={t("Now")}
                  description={t("Leave blank to end it now.")}
                  clearable
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="submit"
            form={END_FORM_ID}
            variant="destructive"
            isLoading={mutation.isPending}
          >
            {t("End agreement")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
