import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createPtoPolicy,
  PTO_POLICY_LIST_KEY,
  updatePtoPolicy,
  type PTOPolicyRow,
} from "@/lib/graphql/pto-policy";
import type { PtoPolicyInput } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { ptoPolicyFormSchema, type PTOPolicyFormValues } from "@trenova/shared/types/pto-policy";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { PTOPolicyForm } from "./pto-policy-form";

export function buildPtoPolicyDefaults(row?: PTOPolicyRow | null): PTOPolicyFormValues {
  if (!row) {
    return {
      name: "",
      code: "",
      description: null,
      status: "Active",
      isDefault: false,
      yearBasis: "CalendarYear",
      countWeekends: true,
      waitingPeriodDays: 0,
      requiresApproval: true,
      enforceBalance: true,
      allowNegative: false,
      negativeFloorDays: null,
      rules: [
        {
          ptoType: "Vacation",
          accrualMethod: "Monthly",
          accrualAmountDays: "0.83",
          maxBalanceDays: null,
          carryoverCapDays: null,
          carryoverExpiryDays: 0,
          tiers: [],
          onTermination: "Forfeit",
        },
      ],
    };
  }
  return {
    name: row.name,
    code: row.code,
    description: row.description ?? null,
    status: row.status,
    isDefault: row.isDefault,
    yearBasis: row.yearBasis,
    countWeekends: row.countWeekends,
    waitingPeriodDays: row.waitingPeriodDays,
    requiresApproval: row.requiresApproval,
    enforceBalance: row.enforceBalance,
    allowNegative: row.allowNegative,
    negativeFloorDays:
      row.allowNegative && Number(row.negativeFloorDays) !== 0 ? row.negativeFloorDays : null,
    rules: row.rules.map((rule) => ({
      ptoType: rule.ptoType,
      accrualMethod: rule.accrualMethod,
      accrualAmountDays: rule.accrualAmountDays,
      maxBalanceDays: rule.maxBalanceDays ?? null,
      carryoverCapDays: rule.carryoverCapDays ?? null,
      carryoverExpiryDays: rule.carryoverExpiryDays,
      tiers: rule.tiers.map((tier) => ({
        minMonths: tier.minMonths,
        accrualAmountDays: tier.accrualAmountDays,
        maxBalanceDays: tier.maxBalanceDays ?? null,
      })),
      onTermination: rule.onTermination,
    })),
  };
}

export function toPtoPolicyInput(values: PTOPolicyFormValues, version?: number): PtoPolicyInput {
  return {
    name: values.name,
    code: values.code.toUpperCase(),
    description: values.description ?? undefined,
    status: values.status,
    isDefault: values.isDefault,
    yearBasis: values.yearBasis,
    countWeekends: values.countWeekends,
    waitingPeriodDays: values.waitingPeriodDays,
    requiresApproval: values.requiresApproval,
    enforceBalance: values.enforceBalance,
    allowNegative: values.allowNegative,
    negativeFloorDays: values.allowNegative ? (values.negativeFloorDays ?? undefined) : "0",
    rules: values.rules.map((rule) => ({
      ptoType: rule.ptoType,
      accrualMethod: rule.accrualMethod,
      accrualAmountDays: rule.accrualMethod === "None" ? "0" : rule.accrualAmountDays,
      maxBalanceDays: rule.maxBalanceDays ?? undefined,
      carryoverCapDays: rule.carryoverCapDays ?? undefined,
      carryoverExpiryDays: rule.carryoverExpiryDays,
      tiers:
        rule.accrualMethod === "None"
          ? []
          : rule.tiers.map((tier) => ({
              minMonths: tier.minMonths,
              accrualAmountDays: tier.accrualAmountDays,
              maxBalanceDays: tier.maxBalanceDays ?? undefined,
            })),
      onTermination: rule.onTermination,
    })),
    version,
  };
}

export function PTOPolicyPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<PTOPolicyRow>) {
  if (mode === "edit" && row) {
    return <PTOPolicyEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <PTOPolicyCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function PTOPolicyCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<PTOPolicyFormValues>({
    resolver: zodResolver(ptoPolicyFormSchema) as Resolver<PTOPolicyFormValues>,
    defaultValues: buildPtoPolicyDefaults(null),
  });

  return (
    <FormCreatePanel<PTOPolicyFormValues, PTOPolicyRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("PTO Policy")}
      description={t("Define how each type of paid time off accrues, caps, and carries over for the workers you assign to it.")}
      queryKey={PTO_POLICY_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<PTOPolicyForm isEdit={false} />}
      mutationFn={async (values) => {
        await createPtoPolicy(toPtoPolicyInput(values));
        return values;
      }}
    />
  );
}

function PTOPolicyEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: PTOPolicyRow;
}) {
  const t = useT();

  const formRow = { ...row, ...buildPtoPolicyDefaults(row) } as unknown as PTOPolicyRow &
    Record<string, unknown>;
  const form = useForm<PTOPolicyFormValues>({
    resolver: zodResolver(ptoPolicyFormSchema) as Resolver<PTOPolicyFormValues>,
    defaultValues: buildPtoPolicyDefaults(row),
  });

  return (
    <FormEditPanel<PTOPolicyFormValues, PTOPolicyRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("PTO Policy")}
      fieldKey="code"
      queryKey={PTO_POLICY_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<PTOPolicyForm isEdit openAssignmentCount={row.openAssignmentCount} />}
      mutationFn={async (values) => {
        await updatePtoPolicy(row.id, toPtoPolicyInput(values, row.version));
        return values;
      }}
    />
  );
}
