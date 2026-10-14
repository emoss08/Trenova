import { DocumentTypeMultiSelectField } from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  FACTORING_ACCOUNTING_TREATMENT_DESCRIPTIONS,
  FACTORING_ACCOUNTING_TREATMENT_LABELS,
  FACTORING_ACCOUNTING_TREATMENT_VALUES,
  FACTORING_RECOURSE_DESCRIPTIONS,
  FACTORING_RECOURSE_LABELS,
  FACTORING_RECOURSE_VALUES,
} from "@/lib/factoring";
import {
  createFactoringAgreement,
  updateFactoringAgreement,
  type FactoringAgreement,
} from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import {
  FormProvider,
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type Resolver,
} from "react-hook-form";
import { toast } from "sonner";
import {
  agreementFormToInput,
  agreementToFormValues,
  emptyAgreementValues,
  factoringAgreementFormSchema,
  feeTierStarts,
  MAX_AGREEMENT_NOTES,
  MAX_CHARGEBACK_WINDOW_DAYS,
  MAX_FEE_TIERS,
  MAX_FUNDING_DAYS,
  type FactoringAgreementFormValues,
} from "./factoring-agreement-schema";

const FORM_ID = "factoring-agreement-form";

/**
 * Drafts a new agreement with a factor, or changes one that has not ended.
 * The terms are entered the way a factoring contract states them — percent
 * of the invoice, days outstanding — and sent as basis points and day bands.
 */
export function FactoringAgreementDialog({
  factoringCompanyId,
  agreement,
  open,
  onOpenChange,
}: {
  factoringCompanyId: string;
  agreement: FactoringAgreement | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<FactoringAgreementFormValues>({
    resolver: zodResolver(factoringAgreementFormSchema) as Resolver<FactoringAgreementFormValues>,
    defaultValues: emptyAgreementValues(0),
  });
  const { reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(
      agreement
        ? agreementToFormValues(agreement)
        : emptyAgreementValues(dateToUnixTimestamp(new Date())),
    );
  }, [open, agreement, reset]);

  const mutation = useApiMutation<
    FactoringAgreement,
    FactoringAgreementFormValues,
    unknown,
    FactoringAgreementFormValues
  >({
    form,
    resourceName: "factoring agreement",
    mutationFn: (values) => {
      const input = agreementFormToInput(factoringCompanyId, values);
      return agreement
        ? updateFactoringAgreement({
            id: agreement.id,
            version: agreement.version,
            agreement: input,
          })
        : createFactoringAgreement(input);
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      toast.success(agreement ? t("Agreement updated") : t("Agreement drafted"), {
        description: agreement
          ? undefined
          : t("Activate it when the contract is signed so invoices start using its terms."),
      });
      onOpenChange(false);
    },
  });

  const recourseOptions = FACTORING_RECOURSE_VALUES.map((value) => ({
    value,
    label: FACTORING_RECOURSE_LABELS[value],
    description: FACTORING_RECOURSE_DESCRIPTIONS[value],
  }));
  const treatmentOptions = FACTORING_ACCOUNTING_TREATMENT_VALUES.map((value) => ({
    value,
    label: FACTORING_ACCOUNTING_TREATMENT_LABELS[value],
    description: FACTORING_ACCOUNTING_TREATMENT_DESCRIPTIONS[value],
  }));
  const { control } = form;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{agreement ? t("Edit agreement") : t("New agreement")}</DialogTitle>
          <DialogDescription>
            {t(
              "The terms you agreed with this factor. A factor has one active agreement at a time, and each invoice uses the terms in force when it posted.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div>
          <FormProvider {...form}>
            <Form
              id={FORM_ID}
              onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
              className="flex flex-col gap-6"
            >
              <FormSection title={t("Term")}>
                <FormGroup cols={2}>
                  <FormControl>
                    <DateField
                      control={control}
                      name="effectiveFrom"
                      label={t("Starts")}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <DateField
                      control={control}
                      name="effectiveTo"
                      label={t("Ends")}
                      placeholder={t("Open-ended")}
                      description={t(
                        "Leave blank while the contract runs until either side ends it.",
                      )}
                      clearable
                    />
                  </FormControl>
                </FormGroup>
              </FormSection>

              <FormSection title={t("Advance and reserve")}>
                <FormGroup cols={2}>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="advanceRatePercent"
                      label={t("Advance rate")}
                      description={t("Share of the invoice paid when the factor funds it.")}
                      suffix="%"
                      decimalScale={2}
                      min={0}
                      max={100}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="reserveRatePercent"
                      label={t("Reserve rate")}
                      description={t("Share held back until the customer pays the factor.")}
                      suffix="%"
                      decimalScale={2}
                      min={0}
                      max={100}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <MoneyField
                      control={control}
                      name="minimumFeeMinor"
                      label={t("Minimum fee")}
                      description={t("The least the factor charges on one invoice.")}
                    />
                  </FormControl>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="chargebackWindowDays"
                      label={t("Chargeback window")}
                      description={t(
                        "Days after the due date before a recourse factor can charge an unpaid invoice back.",
                      )}
                      sideText={t("days")}
                      decimalScale={0}
                      min={0}
                      max={MAX_CHARGEBACK_WINDOW_DAYS}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField
                      control={control}
                      name="recourse"
                      label={t("Recourse")}
                      options={recourseOptions}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField
                      control={control}
                      name="accountingTreatment"
                      label={t("Accounting treatment")}
                      options={treatmentOptions}
                      rules={{ required: true }}
                    />
                  </FormControl>
                </FormGroup>
              </FormSection>

              <FormSection
                title={t("Fees")}
                description={t(
                  "The fee by how long the invoice stays open, counted from funding. Each band starts where the one before it ends; leave the last one open-ended.",
                )}
              >
                <FeeTiersEditor control={control} />
              </FormSection>

              <FormSection title={t("Submission")}>
                <FormGroup cols={2}>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="fundingDays"
                      label={t("Funding days")}
                      description={t("Business days from submission to funding.")}
                      decimalScale={0}
                      min={0}
                      max={MAX_FUNDING_DAYS}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      control={control}
                      name="submissionTime"
                      label={t("Daily schedule time")}
                      description={t(
                        "When the day's schedule of ready invoices is proposed, in your organization's time zone.",
                      )}
                      type="time"
                      step={300}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <DocumentTypeMultiSelectField
                      control={control}
                      name="documentTypes"
                      label={t("Documents the factor needs")}
                      description={t(
                        "Documents every invoice must carry for this factor, beyond what billing already requires. An invoice missing one is not ready.",
                      )}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <TextareaField
                      control={control}
                      name="notes"
                      label={t("Notes")}
                      rows={3}
                      maxLength={MAX_AGREEMENT_NOTES}
                    />
                  </FormControl>
                </FormGroup>
              </FormSection>
            </Form>
          </FormProvider>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={FORM_ID} isLoading={mutation.isPending}>
            {agreement ? t("Save changes") : t("Save draft")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function FeeTiersEditor({ control }: { control: Control<FactoringAgreementFormValues> }) {
  const t = useT();
  const { fields, append, remove } = useFieldArray({ control, name: "feeTiers" });
  const tiers = useWatch({ control, name: "feeTiers" });
  const starts = feeTierStarts(tiers ?? []);

  const addTier = () => {
    append({ daysTo: null, ratePercent: 0, flatMinor: 0 });
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="text-muted-foreground grid grid-cols-[5rem_1fr_1fr_1fr_2rem] gap-2 text-xs font-medium">
        <span>{t("From day")}</span>
        <span>{t("Until day")}</span>
        <span>{t("Fee rate")}</span>
        <span>{t("Flat fee")}</span>
        <span />
      </div>
      {fields.map((field, index) => (
        <div
          key={field.id}
          className="grid grid-cols-[5rem_1fr_1fr_1fr_2rem] items-start gap-2"
          data-testid="fee-tier-row"
        >
          <span className="pt-1.5 text-sm tabular-nums">{starts[index] ?? 0}</span>
          <NumberField
            control={control}
            name={`feeTiers.${index}.daysTo`}
            aria-label={t("Until day")}
            placeholder={index === fields.length - 1 ? t("Open-ended") : undefined}
            decimalScale={0}
            min={0}
          />
          <NumberField
            control={control}
            name={`feeTiers.${index}.ratePercent`}
            aria-label={t("Fee rate")}
            suffix="%"
            decimalScale={2}
            min={0}
            max={100}
          />
          <MoneyField
            control={control}
            name={`feeTiers.${index}.flatMinor`}
            aria-label={t("Flat fee")}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("Remove tier")}
            disabled={fields.length === 1}
            onClick={() => remove(index)}
          >
            <Trash01Icon />
          </Button>
        </div>
      ))}
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addTier}
          disabled={fields.length >= MAX_FEE_TIERS}
        >
          <PlusIcon />
          {t("Add tier")}
        </Button>
      </div>
    </div>
  );
}
