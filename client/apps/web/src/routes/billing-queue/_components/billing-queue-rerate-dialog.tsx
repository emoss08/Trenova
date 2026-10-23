import { useT } from "@trenova/shared/i18n/use-t";
import { FormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { TariffInputsFieldsSkeleton } from "@/components/tariff-inputs/tariff-inputs-fields-skeleton";
import { useTemplateInputs } from "@/hooks/use-template-inputs";
import { describeApiError } from "@/lib/api-error-message";
import { refineShipmentInputs, shipmentTemplateIds } from "@/lib/shipment-tariff-inputs";
import { tariffInputsResolver } from "@/lib/tariff-inputs-resolver";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { TARIFF_INPUTS_REFINEMENT } from "@trenova/shared/lib/tariff-inputs";
import { tariffFormInputValuesSchema } from "@trenova/shared/types/tariff-form";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

const rerateSchema = z.object({
  formulaTemplateId: z.string().min(1, { error: () => translate("Required") }),
  tariffInputs: tariffFormInputValuesSchema.default({}),
});

type RerateFormValues = z.infer<typeof rerateSchema>;

/**
 * Re-rates an item's shipment on new terms: the tariff that bills it, and the values that
 * tariff reads. Saving replaces the shipment's tariff and input values and rates it again, so
 * the freight lines and the tariff's own charges come back from the new bill.
 */
export function BillingQueueRerateDialog({
  open,
  onOpenChange,
  itemId,
  currentTemplateId,
  currentInputs,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemId: string;
  currentTemplateId?: string;
  currentInputs?: Record<string, unknown>;
}) {
  const t = useT();

  const queryClient = useQueryClient();

  const form = useForm<RerateFormValues>({
    resolver: tariffInputsResolver<RerateFormValues>(
      queryClient,
      shipmentTemplateIds,
      (inputs) =>
        zodResolver(
          rerateSchema.superRefine(refineShipmentInputs(inputs), TARIFF_INPUTS_REFINEMENT),
        ) as Resolver<RerateFormValues>,
    ),
    defaultValues: {
      formulaTemplateId: currentTemplateId ?? "",
      tariffInputs: currentInputs ?? {},
    },
  });

  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useMutation({
    mutationFn: (values: RerateFormValues) =>
      apiService.billingQueueService.updateCharges(itemId, {
        formulaTemplateId: values.formulaTemplateId,
        tariffInputs: values.tariffInputs,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    },
    onError: (error) => {
      toast.error(t("Failed to re-rate shipment"), { description: describeApiError(error) });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: RerateFormValues) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Re-rate shipment")}</DialogTitle>
          <DialogDescription>
            {t(
              "Choose the tariff that bills this shipment and the values it reads. Saving rates the shipment again and replaces its freight lines and the tariff's own charges.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <FormGroup cols={1} className="pb-4">
              <FormControl>
                <FormulaTemplateAutocompleteField
                  control={control}
                  name="formulaTemplateId"
                  label={t("Tariff")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <RerateInputs />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Re-rating...")}>
                {t("Re-rate")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

/** The chosen tariff's inputs, holding the shipment's values; one the tariff lacks is dropped. */
function RerateInputs() {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<RerateFormValues>();
  const templateId = useWatch({ control, name: "formulaTemplateId" });
  const declarations = useTemplateInputs(templateId);
  const declared = declarations.data;

  // A value for an input the newly chosen tariff does not declare would only be refused, so
  // once its declarations arrive the shipment's values are narrowed to the ones it reads.
  useEffect(() => {
    if (!declared) return;
    const current = getValues("tariffInputs") ?? {};
    const names = new Set(declared.map((declaration) => declaration.name));
    if (Object.keys(current).every((name) => names.has(name))) return;
    setValue(
      "tariffInputs",
      Object.fromEntries(Object.entries(current).filter(([name]) => names.has(name))),
      { shouldDirty: true },
    );
  }, [declared, getValues, setValue]);

  if (!templateId) return null;

  if (declarations.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertTriangleIcon className="size-4" />
        <AlertDescription>
          {describeApiError(declarations.error, t("The tariff's inputs could not be read."))}
        </AlertDescription>
      </Alert>
    );
  }

  if (!declared) return <TariffInputsFieldsSkeleton fields={2} />;

  return <TariffInputsFields declarations={declared} name="tariffInputs" />;
}
