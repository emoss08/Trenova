import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  IFTA_TAX_RATE_LIST_KEY,
  upsertIftaTaxRates,
  type IftaTaxRateRow,
} from "@/lib/graphql/ifta-tax-rate";
import type { IftaTaxRateInput } from "@trenova/graphql/generated/graphql";
import { blankToNull } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { IftaQuarter } from "@trenova/shared/types/fuel-ifta-enums";
import {
  iftaTaxRateFormSchema,
  type IftaTaxRateFormValues,
} from "@trenova/shared/types/ifta-tax-rate";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { IftaTaxRateForm } from "./ifta-tax-rate-form";

export type IftaPeriod = { year: number; quarter: number };

export function mostRecentCompletedQuarter(now = new Date()): IftaPeriod {
  const quarter = Math.floor(now.getMonth() / 3) + 1;
  return quarter === 1
    ? { year: now.getFullYear() - 1, quarter: 4 }
    : { year: now.getFullYear(), quarter: quarter - 1 };
}

export function buildIftaTaxRateDefaults(
  row: IftaTaxRateRow | null | undefined,
  period: IftaPeriod,
): IftaTaxRateFormValues {
  if (!row) {
    return {
      jurisdictionId: "",
      year: period.year,
      quarter: String(period.quarter) as IftaQuarter,
      fuelType: "Diesel",
      ratePerGallon: "",
      surchargeRatePerGallon: null,
      sourceNote: null,
      sourceUrl: null,
    };
  }
  return {
    jurisdictionId: row.jurisdictionId,
    year: row.year,
    quarter: String(row.quarter) as IftaQuarter,
    fuelType: row.fuelType,
    ratePerGallon: row.ratePerGallon,
    surchargeRatePerGallon: row.surchargeRatePerGallon ?? null,
    sourceNote: row.sourceNote ?? null,
    sourceUrl: row.sourceUrl ?? null,
  };
}

export function toIftaTaxRateInput(values: IftaTaxRateFormValues): IftaTaxRateInput {
  return {
    jurisdictionId: values.jurisdictionId,
    year: values.year,
    quarter: Number(values.quarter),
    fuelType: values.fuelType,
    ratePerGallon: values.ratePerGallon.trim(),
    surchargeRatePerGallon: blankToNull(values.surchargeRatePerGallon) ?? "0",
    sourceNote: blankToNull(values.sourceNote),
    sourceUrl: blankToNull(values.sourceUrl),
  };
}

export function IftaTaxRatePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<IftaTaxRateRow>) {
  if (mode === "edit" && row) {
    return <IftaTaxRateEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <IftaTaxRateCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function IftaTaxRateCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<IftaTaxRateFormValues>({
    resolver: zodResolver(iftaTaxRateFormSchema) as Resolver<IftaTaxRateFormValues>,
    defaultValues: buildIftaTaxRateDefaults(null, mostRecentCompletedQuarter()),
  });

  return (
    <FormCreatePanel<IftaTaxRateFormValues, IftaTaxRateRow>
      open={open}
      onOpenChange={onOpenChange}
      title="IFTA Tax Rate"
      description="Publish one jurisdiction's rate for a quarter and fuel type. Rates are global, so this is what every organization's return will owe."
      queryKey={IFTA_TAX_RATE_LIST_KEY}
      form={form}
      size="md"
      formComponent={<IftaTaxRateForm isEdit={false} />}
      mutationFn={async (values) => {
        await upsertIftaTaxRates([toIftaTaxRateInput(values)]);
        return values;
      }}
    />
  );
}

function IftaTaxRateEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: IftaTaxRateRow;
}) {
  const period = { year: row.year, quarter: row.quarter };
  const formRow = {
    ...row,
    ...buildIftaTaxRateDefaults(row, period),
  } as unknown as IftaTaxRateRow & Record<string, unknown>;
  const form = useForm<IftaTaxRateFormValues>({
    resolver: zodResolver(iftaTaxRateFormSchema) as Resolver<IftaTaxRateFormValues>,
    defaultValues: buildIftaTaxRateDefaults(row, period),
  });

  return (
    <FormEditPanel<IftaTaxRateFormValues, IftaTaxRateRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="IFTA Tax Rate"
      titleComponent={(record) => (
        <span>
          {record.jurisdiction.code} · {record.fuelType} · Q{record.quarter} {record.year}
        </span>
      )}
      queryKey={IFTA_TAX_RATE_LIST_KEY}
      form={form}
      size="md"
      formComponent={<IftaTaxRateForm isEdit />}
      mutationFn={async (values) => {
        await upsertIftaTaxRates([toIftaTaxRateInput(values)]);
        return values;
      }}
    />
  );
}
