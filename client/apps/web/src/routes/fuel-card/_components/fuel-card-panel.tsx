import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createFuelCard,
  FUEL_CARD_LIST_KEY,
  updateFuelCard,
  type FuelCardRow,
} from "@/lib/graphql/fuel-card";
import type { FuelCardInput } from "@trenova/graphql/generated/graphql";
import { blankToNull } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { fuelCardFormSchema, type FuelCardFormValues } from "@trenova/shared/types/fuel-card";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { FuelCardForm } from "./fuel-card-form";

export function buildFuelCardDefaults(row?: FuelCardRow | null): FuelCardFormValues {
  if (!row) {
    return {
      provider: "Comdata",
      lastFour: "",
      label: "",
      externalCardId: null,
      assignedWorkerId: null,
      assignedTractorId: null,
      status: "Active",
      expiresAt: null,
      notes: null,
    };
  }
  return {
    provider: row.provider,
    lastFour: row.lastFour,
    label: row.label,
    externalCardId: row.externalCardId ?? null,
    assignedWorkerId: row.assignedWorkerId ?? null,
    assignedTractorId: row.assignedTractorId ?? null,
    status: row.status === "Cancelled" ? "Suspended" : row.status,
    expiresAt: row.expiresAt ?? null,
    notes: row.notes ?? null,
  };
}

export function toFuelCardInput(values: FuelCardFormValues): FuelCardInput {
  return {
    provider: values.provider,
    lastFour: values.lastFour.trim(),
    label: values.label.trim(),
    externalCardId: blankToNull(values.externalCardId),
    assignedWorkerId: blankToNull(values.assignedWorkerId),
    assignedTractorId: blankToNull(values.assignedTractorId),
    status: values.status,
    expiresAt: values.expiresAt ?? null,
    notes: blankToNull(values.notes),
  };
}

export function FuelCardPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<FuelCardRow>) {
  if (mode === "edit" && row) {
    return <FuelCardEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <FuelCardCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function FuelCardCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<FuelCardFormValues>({
    resolver: zodResolver(fuelCardFormSchema) as Resolver<FuelCardFormValues>,
    defaultValues: buildFuelCardDefaults(null),
  });

  return (
    <FormCreatePanel<FuelCardFormValues, FuelCardRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Fuel Card"
      description="Register a card so imported statements and hand-keyed purchases can be tied to the driver and unit that fuelled with it."
      queryKey={FUEL_CARD_LIST_KEY}
      form={form}
      size="md"
      formComponent={<FuelCardForm isEdit={false} />}
      mutationFn={async (values) => {
        await createFuelCard(toFuelCardInput(values));
        return values;
      }}
    />
  );
}

function FuelCardEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: FuelCardRow;
}) {
  const formRow = { ...row, ...buildFuelCardDefaults(row) } as unknown as FuelCardRow &
    Record<string, unknown>;
  const form = useForm<FuelCardFormValues>({
    resolver: zodResolver(fuelCardFormSchema) as Resolver<FuelCardFormValues>,
    defaultValues: buildFuelCardDefaults(row),
  });

  return (
    <FormEditPanel<FuelCardFormValues, FuelCardRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Fuel Card"
      fieldKey="label"
      queryKey={FUEL_CARD_LIST_KEY}
      form={form}
      size="md"
      formComponent={
        <FuelCardForm
          isEdit
          cancelled={row.status === "Cancelled"}
          cancelReason={row.cancelReason}
        />
      }
      mutationFn={async (values) => {
        await updateFuelCard(row.id, row.version, toFuelCardInput(values));
        return values;
      }}
    />
  );
}
