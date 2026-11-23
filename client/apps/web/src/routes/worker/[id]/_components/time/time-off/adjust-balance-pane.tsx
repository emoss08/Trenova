import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
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
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  adjustPtoBalanceFormSchema,
  type AdjustPTOBalanceFormValues,
} from "@trenova/shared/types/pto-policy";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";

export type AdjustBalancePaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  balances: readonly WorkerPTOBalanceView[];
  onAdjusted: () => void;
};

export type AdjustmentPreview = { text: string; warn: boolean };

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
    return { text: t("{0} is {1} days today.", name, current.toFixed(2)), warn: false };
  }
  const next = current + change;
  const cap = balance.maxBalanceDays ? Number(balance.maxBalanceDays) : null;
  if (cap !== null && next > cap) {
    return {
      text: t(
        "{0} goes from {1} to {2} days, over the {3} day cap.",
        name,
        current.toFixed(2),
        next.toFixed(2),
        cap.toFixed(2),
      ),
      warn: true,
    };
  }
  if (next < 0) {
    return {
      text: t(
        "{0} goes from {1} to {2} days, below zero.",
        name,
        current.toFixed(2),
        next.toFixed(2),
      ),
      warn: true,
    };
  }
  return {
    text: t("{0} goes from {1} to {2} days.", name, current.toFixed(2), next.toFixed(2)),
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
  const { control, handleSubmit, reset } = form;
  const [ptoType, amountDays] = useWatch({ control, name: ["ptoType", "amountDays"] });

  useEffect(() => {
    if (open) reset(emptyValues(balances));
  }, [open, reset, balances]);

  const preview = adjustmentPreview(
    t,
    balances.find((balance) => balance.ptoType === ptoType),
    amountDays ?? "",
  );

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
    onSuccess: () => {
      onAdjusted();
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t("Adjust a balance")}
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
                <InputField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="amountDays"
                  label={t("Adjustment")}
                  placeholder={t("+1.0 or -0.5")}
                  required
                  className="font-mono"
                />
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
              {preview ? (
                <FormControl cols="full">
                  <p
                    aria-live="polite"
                    className={cn(
                      "text-2xs tabular-nums",
                      preview.warn ? "text-warning-foreground" : "text-muted-foreground",
                    )}
                  >
                    {preview.text}
                  </p>
                </FormControl>
              ) : null}
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
