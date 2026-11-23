import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { FieldNote } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { adjustWorkerPtoBalance, type WorkerPTOBalanceView } from "@/lib/graphql/pto-policy";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  adjustPtoBalanceFormSchema,
  type AdjustPTOBalanceFormValues,
} from "@trenova/shared/types/pto-policy";
import { FormProvider, useForm, useWatch, type Control, type Resolver } from "react-hook-form";

export type AdjustBalancePaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  balances: readonly WorkerPTOBalanceView[];
  /** Refreshes the figures the adjustment moved; the save holds until it settles. */
  onAdjusted: () => Promise<unknown>;
};

export type AdjustmentPreview = { text: string; warn: boolean };

/** A balance in days as the reader's language writes it, to two places: "1.50", "1,200.00". */
function days(value: number): string {
  return formatNumber(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/** What an adjustment does to the balance, in one line, before it is posted. */
export function adjustmentPreview(
  t: TranslateFn,
  balance: WorkerPTOBalanceView | undefined,
  amount: string,
): AdjustmentPreview | null {
  if (!balance) return null;
  const name = t(ptoTypeLabel(balance.ptoType));
  const current = Number(balance.balanceDays);
  const change = Number(amount);
  if (!amount.trim() || !Number.isFinite(change) || change === 0) {
    return { text: t("{0} is {1} days today.", name, days(current)), warn: false };
  }
  const next = current + change;
  const cap = balance.maxBalanceDays ? Number(balance.maxBalanceDays) : null;
  if (cap !== null && next > cap) {
    return {
      text: t(
        "{0} goes from {1} to {2} days, over the {3} day cap.",
        name,
        days(current),
        days(next),
        days(cap),
      ),
      warn: true,
    };
  }
  if (next < 0) {
    return {
      text: t("{0} goes from {1} to {2} days, below zero.", name, days(current), days(next)),
      warn: true,
    };
  }
  return {
    text: t("{0} goes from {1} to {2} days.", name, days(current), days(next)),
    warn: false,
  };
}

function emptyValues(balances: readonly WorkerPTOBalanceView[]): AdjustPTOBalanceFormValues {
  return {
    ptoType: (balances.find((balance) => balance.tracked)?.ptoType ??
      "Vacation") as AdjustPTOBalanceFormValues["ptoType"],
    amountDays: "",
    effectiveAt: getTodayDate(),
    note: "",
  };
}

/** A correction written to the ledger with the reader's name and reason. */
export function AdjustBalancePane({
  open,
  onOpenChange,
  workerId,
  balances,
  onAdjusted,
}: AdjustBalancePaneProps) {
  const t = useT();
  const tracked = balances.filter((balance) => balance.tracked);
  const options = tracked.map((balance) => ({
    value: balance.ptoType,
    label: t(ptoTypeLabel(balance.ptoType)),
  }));
  const form = useForm<AdjustPTOBalanceFormValues>({
    resolver: zodResolver(adjustPtoBalanceFormSchema) as Resolver<AdjustPTOBalanceFormValues>,
    defaultValues: emptyValues(balances),
  });
  const { control, handleSubmit } = form;

  const { mutateAsync, isPending } = useApiMutation<
    Awaited<ReturnType<typeof adjustWorkerPtoBalance>>,
    AdjustPTOBalanceFormValues,
    unknown,
    AdjustPTOBalanceFormValues
  >({
    form,
    resourceName: "PTO balance",
    scope: { id: `worker-pto-adjust:${workerId}` },
    mutationFn: (values) =>
      adjustWorkerPtoBalance({
        workerId,
        ptoType: values.ptoType,
        amountDays: values.amountDays,
        effectiveAt: values.effectiveAt,
        note: values.note,
      }),
    onSuccess: async () => {
      await onAdjusted();
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t("Adjust balance")}
      description={t(
        "A correction written to the ledger with your name and reason. Positive days add, negative days remove.",
      )}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(event);
          }}
        >
          <WorkPaneBody>
            <FormGroup cols={2}>
              <FormControl cols="full">
                <SelectField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="ptoType"
                  label={t("Balance")}
                  placeholder={t("Pick a balance")}
                  options={options}
                  required
                />
              </FormControl>
              <FormControl>
                <AdjustmentField control={control} balances={balances} />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="effectiveAt"
                  label={t("Effective")}
                  placeholder={t("Today")}
                  required
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="note"
                  label={t("Reason")}
                  placeholder={t("e.g. Worked the Thanksgiving holiday")}
                  required
                  maxLength={255}
                  description={t("Stays on the ledger entry and in the audit log.")}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {t("Adjust balance")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}

/** The adjustment, with what it does to the balance under it as it is typed. */
function AdjustmentField({
  control,
  balances,
}: {
  control: Control<AdjustPTOBalanceFormValues>;
  balances: readonly WorkerPTOBalanceView[];
}) {
  const t = useT();
  const [ptoType, amountDays] = useWatch({ control, name: ["ptoType", "amountDays"] });
  const preview = adjustmentPreview(
    t,
    balances.find((balance) => balance.ptoType === ptoType),
    amountDays ?? "",
  );
  return (
    <InputField<AdjustPTOBalanceFormValues>
      control={control}
      name="amountDays"
      label={t("Adjustment")}
      placeholder={t("+1.0 or -0.5")}
      required
      inputClassProps="font-mono"
      description={
        preview ? (
          <FieldNote live tone={preview.warn ? "warning" : "muted"}>
            {preview.text}
          </FieldNote>
        ) : undefined
      }
    />
  );
}
