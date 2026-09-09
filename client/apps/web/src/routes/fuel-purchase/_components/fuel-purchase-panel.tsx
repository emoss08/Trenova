import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import {
  createFuelPurchase,
  FUEL_PURCHASE_LIST_KEY,
  updateFuelPurchase,
  type FuelPurchaseRow,
} from "@/lib/graphql/fuel-purchase";
import type { FuelPurchaseInput } from "@trenova/graphql/generated/graphql";
import { blankToNull } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  createFuelPurchaseFormSchema,
  type FuelPurchaseFormValues,
} from "@trenova/shared/types/fuel-purchase";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileTextIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { FuelPurchaseForm } from "./fuel-purchase-form";

const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));

const CLOCK_GRACE_SECONDS = 60;

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

export function buildFuelPurchaseDefaults(
  row: FuelPurchaseRow | null | undefined,
  now: number,
): FuelPurchaseFormValues {
  if (!row) {
    return {
      tractorId: "",
      workerId: null,
      jurisdictionId: "",
      purchasedAt: now,
      vendor: null,
      vendorCity: null,
      fuelType: "Diesel",
      quantity: "",
      quantityUnit: "Gallon",
      unitPrice: null,
      totalAmount: "",
      currencyCode: "USD",
      odometer: null,
      fuelCardId: null,
      cardLastFour: null,
      transactionReference: null,
      taxPaid: true,
      notes: null,
    };
  }
  return {
    tractorId: row.tractorId,
    workerId: row.workerId ?? null,
    jurisdictionId: row.jurisdictionId,
    purchasedAt: row.purchasedAt,
    vendor: row.vendor ?? null,
    vendorCity: row.vendorCity ?? null,
    fuelType: row.fuelType,
    quantity: row.quantity,
    quantityUnit: row.quantityUnit,
    unitPrice: row.unitPrice ?? null,
    totalAmount: row.totalAmount,
    currencyCode: row.currencyCode,
    odometer: row.odometer ?? null,
    fuelCardId: row.fuelCardId ?? null,
    cardLastFour: row.cardLastFour ?? null,
    transactionReference: row.transactionReference ?? null,
    taxPaid: row.taxPaid,
    notes: row.notes ?? null,
  };
}

export function toFuelPurchaseInput(values: FuelPurchaseFormValues): FuelPurchaseInput {
  return {
    tractorId: values.tractorId,
    workerId: blankToNull(values.workerId),
    jurisdictionId: values.jurisdictionId,
    purchasedAt: values.purchasedAt,
    vendor: blankToNull(values.vendor),
    vendorCity: blankToNull(values.vendorCity),
    fuelType: values.fuelType,
    quantity: values.quantity.trim(),
    quantityUnit: values.quantityUnit,
    unitPrice: blankToNull(values.unitPrice),
    totalAmount: values.totalAmount.trim(),
    currencyCode: values.currencyCode.trim().toUpperCase(),
    odometer: values.odometer ?? null,
    fuelCardId: blankToNull(values.fuelCardId),
    cardLastFour: blankToNull(values.cardLastFour),
    transactionReference: blankToNull(values.transactionReference),
    taxPaid: values.taxPaid,
    notes: blankToNull(values.notes),
  };
}

function useFuelPurchaseResolver(): Resolver<FuelPurchaseFormValues> {
  return useMemo<Resolver<FuelPurchaseFormValues>>(
    () => (values, context, options) =>
      (
        zodResolver(
          createFuelPurchaseFormSchema(nowSeconds() + CLOCK_GRACE_SECONDS),
        ) as Resolver<FuelPurchaseFormValues>
      )(values, context, options),
    [],
  );
}

export function FuelPurchasePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FuelPurchaseRow>) {
  if (mode === "edit" && row) {
    return <FuelPurchaseEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <FuelPurchaseCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function FuelPurchaseCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const resolver = useFuelPurchaseResolver();
  const form = useForm<FuelPurchaseFormValues>({
    resolver,
    defaultValues: buildFuelPurchaseDefaults(null, nowSeconds()),
  });

  return (
    <FormCreatePanel<FuelPurchaseFormValues, FuelPurchaseRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Fuel Purchase"
      description="Record fuel bought for a tractor. The purchase is a tax record: its gallons and the jurisdiction they were bought in feed the quarterly IFTA return."
      queryKey={FUEL_PURCHASE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<FuelPurchaseForm isEdit={false} />}
      mutationFn={async (values) => {
        await createFuelPurchase(toFuelPurchaseInput(values));
        return values;
      }}
    />
  );
}

function FuelPurchaseEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: FuelPurchaseRow;
}) {
  const resolver = useFuelPurchaseResolver();
  const formRow = {
    ...row,
    ...buildFuelPurchaseDefaults(row, nowSeconds()),
  } as unknown as FuelPurchaseRow & Record<string, unknown>;
  const form = useForm<FuelPurchaseFormValues>({
    resolver,
    defaultValues: buildFuelPurchaseDefaults(row, nowSeconds()),
  });

  const tabs = useMemo(
    () => [
      {
        value: "documents",
        label: "Documents",
        icon: FileTextIcon,
        content: DocumentsTab,
        contentProps: {
          resourceType: "fuel_purchase",
          resourceId: row.id,
        },
      },
    ],
    [row.id],
  );

  return (
    <TabbedFormEditPanel<FuelPurchaseFormValues, FuelPurchaseRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Fuel Purchase"
      titleComponent={(record) => (
        <span>
          {record.tractor?.code ? `${record.tractor.code} · ` : ""}
          {record.jurisdiction.code} · {record.gallons} gal
        </span>
      )}
      queryKey={FUEL_PURCHASE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={
        <FuelPurchaseForm
          isEdit
          imported={row.source === "CardImport"}
          importBatchId={row.importBatchId}
        />
      }
      tabs={tabs}
      mutationFn={async (values) => {
        await updateFuelPurchase(row.id, row.version, toFuelPurchaseInput(values));
        return values;
      }}
    />
  );
}
