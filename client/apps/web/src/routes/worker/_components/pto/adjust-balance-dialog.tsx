import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ptoTypeChoices } from "@/lib/choices";
import { adjustWorkerPtoBalance, type WorkerPTOBalanceView } from "@/lib/graphql/pto-policy";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  adjustPtoBalanceFormSchema,
  type AdjustPTOBalanceFormValues,
} from "@trenova/shared/types/pto-policy";
import { useCallback, useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type AdjustBalanceDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  balances: WorkerPTOBalanceView[];
  onAdjusted?: () => void;
};

export function AdjustBalanceDialog({
  open,
  onOpenChange,
  workerId,
  balances,
  onAdjusted,
}: AdjustBalanceDialogProps) {
  const t = useT();

  const tracked = useMemo(() => balances.filter((balance) => balance.tracked), [balances]);
  const form = useForm<AdjustPTOBalanceFormValues>({
    resolver: zodResolver(adjustPtoBalanceFormSchema) as Resolver<AdjustPTOBalanceFormValues>,
    defaultValues: {
      ptoType: tracked[0]?.ptoType ?? "Vacation",
      amountDays: "",
      effectiveAt: getTodayDate(),
      note: "",
    },
  });
  const { control, handleSubmit, reset } = form;
  const [ptoType, amountDays] = useWatch({ control, name: ["ptoType", "amountDays"] });

  useEffect(() => {
    if (open) {
      reset({
        ptoType: tracked[0]?.ptoType ?? "Vacation",
        amountDays: "",
        effectiveAt: getTodayDate(),
        note: "",
      });
    }
  }, [open, reset, tracked]);

  const current = balances.find((balance) => balance.ptoType === ptoType);
  const preview = useMemo(() => {
    const amount = Number(amountDays);
    if (!current || !Number.isFinite(amount) || amount === 0) return null;
    return (Number(current.balanceDays) + amount).toFixed(2);
  }, [amountDays, current]);

  const { mutateAsync, isPending } = useApiMutation<
    Awaited<ReturnType<typeof adjustWorkerPtoBalance>>,
    AdjustPTOBalanceFormValues,
    unknown,
    AdjustPTOBalanceFormValues
  >({
    form,
    resourceName: "PTO balance",
    mutationFn: (values) =>
      adjustWorkerPtoBalance({
        workerId,
        ptoType: values.ptoType,
        amountDays: values.amountDays,
        effectiveAt: values.effectiveAt,
        note: values.note,
      }),
    onSuccess: (entry) => {
      toast.success(t("Balance adjusted"), {
        description: `${entry.ptoType} balance is now ${entry.balanceAfterDays} days.`,
      });
      onAdjusted?.();
      onOpenChange(false);
    },
  });

  const onSubmit = useCallback(
    async (values: AdjustPTOBalanceFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Adjust PTO Balance")}</DialogTitle>
          <DialogDescription>
            {t("Post a manual correction to the ledger. Positive amounts add days, negative amounts remove them. The note is kept on the ledger and in the audit log.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              void handleSubmit(onSubmit)(e);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <SelectField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="ptoType"
                  label={t("PTO type")}
                  placeholder={t("Pick a PTO type")}
                  options={ptoTypeChoices}
                  rules={{ required: true }}
                  description={t("The balance this correction is posted to.")}
                />
              </FormControl>
              <FormControl>
                <InputField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="amountDays"
                  label={t("Amount (days)")}
                  placeholder={t("e.g. 2 or -1.5")}
                  rules={{ required: true }}
                  description={
                    current
                      ? `Current ${current.balanceDays}${preview ? ` → ${preview}` : ""}`
                      : "Days to add, or remove with a minus sign."
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="effectiveAt"
                  label={t("Effective date")}
                  placeholder={t("Today")}
                  rules={{ required: true }}
                  description={t("The date the ledger shows the correction taking effect.")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<AdjustPTOBalanceFormValues>
                  control={control}
                  name="note"
                  label={t("Reason")}
                  placeholder={t("e.g. Credited 2 days for holiday worked")}
                  rules={{ required: true }}
                  maxLength={255}
                  description={t("Why the balance is being corrected; it stays on the ledger entry.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="button"
                onClick={() => void handleSubmit(onSubmit)()}
                isLoading={isPending}
                loadingText={t("Posting...")}
                // disabled={tracked.length === 0}
              >
                {t("Post Adjustment")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
