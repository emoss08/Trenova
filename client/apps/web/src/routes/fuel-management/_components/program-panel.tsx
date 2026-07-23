import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createFuelSurchargeProgram,
  updateFuelSurchargeProgram,
  type FuelSurchargeProgramDetail,
} from "@/lib/graphql/fuel-surcharge";
import { queries } from "@/lib/queries";
import {
  fuelSurchargeProgramSchema,
  type FuelSurchargeProgramFormValues,
} from "@/types/fuel-surcharge";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { ProgramForm } from "./program-form";

const DEFAULT_VALUES: Partial<FuelSurchargeProgramFormValues> = {
  name: "",
  code: "",
  description: "",
  status: "Active",
  fuelIndexId: "",
  accessorialChargeId: "",
  method: "PerMileStep",
  pegPrice: 1.2,
  increment: 0.05,
  incrementRate: 0.01,
  milesPerGallon: null,
  percentBasis: "Linehaul",
  stepRounding: "Up",
  rateRounding: "HalfUp",
  ratePrecision: 4,
  minAmount: null,
  maxAmount: null,
  dateBasis: "PickupDate",
  priceEffectiveDay: 3,
  missingPriceFallback: "UseLatestAvailable",
  effectiveStartDate: null,
  effectiveEndDate: null,
  shipmentTypeIds: [],
  serviceTypeIds: [],
  tractorTypeIds: [],
  trailerTypeIds: [],
  tableRows: [],
};

function detailToFormValues(
  detail: FuelSurchargeProgramDetail,
): Partial<FuelSurchargeProgramFormValues> {
  return {
    name: detail.name,
    code: detail.code,
    description: detail.description ?? "",
    status: detail.status,
    fuelIndexId: detail.fuelIndexId,
    accessorialChargeId: detail.accessorialChargeId,
    method: detail.method,
    pegPrice: detail.pegPrice != null ? Number(detail.pegPrice) : null,
    increment: detail.increment != null ? Number(detail.increment) : null,
    incrementRate: detail.incrementRate != null ? Number(detail.incrementRate) : null,
    milesPerGallon: detail.milesPerGallon != null ? Number(detail.milesPerGallon) : null,
    percentBasis: detail.percentBasis,
    stepRounding: detail.stepRounding,
    rateRounding: detail.rateRounding,
    ratePrecision: detail.ratePrecision,
    minAmount: detail.minAmount != null ? Number(detail.minAmount) : null,
    maxAmount: detail.maxAmount != null ? Number(detail.maxAmount) : null,
    dateBasis: detail.dateBasis,
    priceEffectiveDay: detail.priceEffectiveDay,
    missingPriceFallback: detail.missingPriceFallback,
    effectiveStartDate: detail.effectiveStartDate ?? null,
    effectiveEndDate: detail.effectiveEndDate ?? null,
    shipmentTypeIds: detail.shipmentTypeIds ?? [],
    serviceTypeIds: detail.serviceTypeIds ?? [],
    tractorTypeIds: detail.tractorTypeIds ?? [],
    trailerTypeIds: detail.trailerTypeIds ?? [],
    tableRows: (detail.tableRows ?? []).map((row, index) => ({
      id: row.id,
      priceMin: row.priceMin != null ? Number(row.priceMin) : null,
      priceMax: row.priceMax != null ? Number(row.priceMax) : null,
      value: Number(row.value),
      sortOrder: index,
    })),
  };
}

type ProgramPanelProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  programId: string | null;
};

export function ProgramPanel({ open, onOpenChange, programId }: ProgramPanelProps) {
  const queryClient = useQueryClient();
  const isEdit = !!programId;

  const form = useForm<FuelSurchargeProgramFormValues>({
    resolver: zodResolver(fuelSurchargeProgramSchema) as Resolver<FuelSurchargeProgramFormValues>,
    defaultValues: DEFAULT_VALUES as FuelSurchargeProgramFormValues,
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { data: detail, isLoading: isDetailLoading } = useQuery({
    ...queries.fuelSurcharge.programDetail(programId ?? ""),
    enabled: open && isEdit,
  });

  useEffect(() => {
    if (open && detail) {
      reset(
        { ...DEFAULT_VALUES, ...detailToFormValues(detail) } as FuelSurchargeProgramFormValues,
        { keepDefaultValues: true },
      );
    }
    if (open && !isEdit) {
      reset(DEFAULT_VALUES as FuelSurchargeProgramFormValues);
    }
  }, [open, detail, isEdit, reset]);

  const invalidate = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queries.fuelSurcharge.currentRates().queryKey });
    void queryClient.invalidateQueries({ queryKey: ["fuel-surcharge-program-list"] });
    if (programId) {
      void queryClient.invalidateQueries({
        queryKey: queries.fuelSurcharge.programDetail(programId).queryKey,
      });
    }
  }, [queryClient, programId]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: FuelSurchargeProgramFormValues) =>
      isEdit && programId
        ? updateFuelSurchargeProgram(programId, values)
        : createFuelSurchargeProgram(values),
    onSuccess: () => {
      toast.success(isEdit ? "Fuel surcharge program updated" : "Fuel surcharge program created");
      invalidate();
      reset();
      onOpenChange(false);
    },
    setFormError: setError,
    resourceName: "Fuel Surcharge Program",
  });

  const onSubmit = useCallback(
    async (values: FuelSurchargeProgramFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? (detail?.name ?? "Fuel Surcharge Program") : "New Fuel Surcharge Program"}
      description={
        isEdit
          ? undefined
          : "Formula-first programs render their matrix live; table programs support one-step band generation"
      }
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="fuel-program-form" isLoading={isSubmitting}>
            {isEdit ? "Save Changes" : "Create Program"}
          </Button>
        </>
      }
    >
      {isEdit && isDetailLoading ? (
        <ComponentLoader message="Loading Fuel Surcharge Program..." />
      ) : (
        <FormProvider {...form}>
          <Form id="fuel-program-form" onSubmit={handleSubmit(onSubmit)}>
            <ProgramForm />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
