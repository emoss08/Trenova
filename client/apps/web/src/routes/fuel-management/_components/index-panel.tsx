import { useT } from "@trenova/shared/i18n/use-t";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { fuelTypeChoices } from "@/lib/choices";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createFuelIndex,
  updateFuelIndex,
  type FuelDashboardEntry,
} from "@/lib/graphql/fuel-surcharge";
import { queries } from "@/lib/queries";
import { fuelIndexSchema, type FuelIndex } from "@/types/fuel-surcharge";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const DEFAULT_VALUES: Partial<FuelIndex> = {
  name: "",
  code: "",
  description: "",
  source: "Custom",
  fuelType: "Diesel",
  region: "",
  eiaSeriesId: "",
  currency: "USD",
  isActive: true,
};

type IndexPanelProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  entry: FuelDashboardEntry | null;
};

export function IndexPanel({ open, onOpenChange, entry }: IndexPanelProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = !!entry;

  const form = useForm<FuelIndex>({
    resolver: zodResolver(fuelIndexSchema) as Resolver<FuelIndex>,
    defaultValues: DEFAULT_VALUES as FuelIndex,
  });

  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (!open) return;
    if (entry) {
      reset({
        ...DEFAULT_VALUES,
        name: entry.index.name,
        code: entry.index.code,
        description: entry.index.description ?? "",
        source: entry.index.source,
        fuelType: entry.index.fuelType,
        region: entry.index.region ?? "",
        eiaSeriesId: entry.index.eiaSeriesId ?? "",
        currency: entry.index.currency,
        isActive: entry.index.isActive,
      } as FuelIndex);
    } else {
      reset(DEFAULT_VALUES as FuelIndex);
    }
  }, [open, entry, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: FuelIndex) =>
      isEdit && entry ? updateFuelIndex(entry.index.id, values) : createFuelIndex(values),
    onSuccess: () => {
      toast.success(isEdit ? "Fuel index updated" : "Custom fuel index created");
      void queryClient.invalidateQueries({
        queryKey: queries.fuelSurcharge.dashboard().queryKey,
      });
      reset();
      onOpenChange(false);
    },
    form,
    resourceName: "Fuel Index",
  });

  const onSubmit = useCallback(
    async (values: FuelIndex) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? (entry?.index.name ?? "Fuel Index") : "New Custom Fuel Index"}
      description={t(
        "Custom indices take manually entered weekly prices — ideal for Canadian FCA or contract-dictated pegs",
      )}
      size="md"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form="fuel-index-form" isLoading={isSubmitting}>
            {isEdit ? t("Save Changes") : t("Create Index")}
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <Form id="fuel-index-form" onSubmit={handleSubmit(onSubmit)}>
          <FormGroup cols={2}>
            <FormControl cols="full">
              <SwitchField
                control={form.control}
                name="isActive"
                label={t("Active")}
                description={t(
                  "Inactive indices are hidden from program selection and the dashboard.",
                )}
                outlined
                position="left"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="name"
                label={t("Name")}
                placeholder={t("FCA Canadian Diesel")}
                rules={{ required: true }}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="code"
                label={t("Code")}
                placeholder="FCA_CAD"
                rules={{ required: true }}
                maxLength={50}
                description={t("Short unique identifier.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
                name="fuelType"
                label={t("Fuel Type")}
                options={fuelTypeChoices}
                description={t("The fuel product this index prices.")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="region"
                label={t("Region")}
                placeholder={t("Canada, PADD 2, Northeast...")}
                maxLength={100}
                description={t("Geographic region the price applies to.")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="currency"
                label={t("Currency")}
                placeholder={t("USD")}
                rules={{ required: true }}
                maxLength={3}
                description={t("3-letter ISO currency of the entered prices.")}
              />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                control={form.control}
                name="description"
                label={t("Description")}
                placeholder={t("Where this index comes from and how it's maintained")}
              />
            </FormControl>
          </FormGroup>
        </Form>
      </FormProvider>
    </DataTablePanelContainer>
  );
}
