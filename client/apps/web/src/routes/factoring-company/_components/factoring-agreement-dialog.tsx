import {
  DocumentTypeMultiSelectField,
  GLAccountAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField, DateField } from "@/components/fields/date-field/date-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { TimeOfDayField } from "@/components/fields/time-of-day-field";
import { FormCreateModal } from "@/components/form-create-modal";
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
  useFormContext,
  type Control,
  type Resolver,
} from "react-hook-form";
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

function useAgreementForm() {
  return useForm<FactoringAgreementFormValues>({
    resolver: zodResolver(factoringAgreementFormSchema) as Resolver<FactoringAgreementFormValues>,
    defaultValues: emptyAgreementValues(0),
  });
}

/**
 * Drafts a new agreement with a factor, or changes one that has not ended. A new one opens in the
 * form-create dialog; a change opens in its own dialog over the same fields.
 */
export function FactoringAgreementDialog({
  factoringCompanyId,
  agreement,
  template = null,
  open,
  onOpenChange,
}: {
  factoringCompanyId: string;
  agreement: FactoringAgreement | null;
  /** For a new agreement: the one whose terms it starts from, dated from today. */
  template?: FactoringAgreement | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return agreement ? (
    <FactoringAgreementEditDialog
      factoringCompanyId={factoringCompanyId}
      agreement={agreement}
      open={open}
      onOpenChange={onOpenChange}
    />
  ) : (
    <FactoringAgreementCreateDialog
      factoringCompanyId={factoringCompanyId}
      template={template}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

function FactoringAgreementCreateDialog({
  factoringCompanyId,
  template,
  open,
  onOpenChange,
}: {
  factoringCompanyId: string;
  template: FactoringAgreement | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useAgreementForm();
  const { reset } = form;

  useEffect(() => {
    if (!open) return;
    const today = dateToUnixTimestamp(new Date());
    reset(
      template
        ? { ...agreementToFormValues(template), effectiveFrom: today, effectiveTo: null }
        : emptyAgreementValues(today),
    );
  }, [open, template, reset]);

  return (
    <FormCreateModal<FactoringAgreementFormValues, FactoringAgreementFormValues, FactoringAgreement>
      open={open}
      onOpenChange={onOpenChange}
      title={template ? t("New version from these terms") : t("New agreement")}
      description={t(
        "The terms you agreed with this factor. A factor has one active agreement at a time, and each invoice uses the terms in force when it posted.",
      )}
      resourceName="factoring agreement"
      size="xl"
      form={form}
      formComponent={<FactoringAgreementFields control={form.control} />}
      mutationFn={(values) =>
        createFactoringAgreement(agreementFormToInput(factoringCompanyId, values))
      }
      onCreated={() => invalidateFactoringQueries(queryClient)}
      submitText={t("Save draft")}
    />
  );
}

function FactoringAgreementEditDialog({
  factoringCompanyId,
  agreement,
  open,
  onOpenChange,
}: {
  factoringCompanyId: string;
  agreement: FactoringAgreement;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useAgreementForm();
  const { reset } = form;

  useEffect(() => {
    if (open) reset(agreementToFormValues(agreement));
  }, [open, agreement, reset]);

  const mutation = useApiMutation<
    FactoringAgreement,
    FactoringAgreementFormValues,
    unknown,
    FactoringAgreementFormValues
  >({
    form,
    resourceName: "factoring agreement",
    mutationFn: (values) =>
      updateFactoringAgreement({
        id: agreement.id,
        version: agreement.version,
        agreement: agreementFormToInput(factoringCompanyId, values),
      }),
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("Edit agreement")}</DialogTitle>
          <DialogDescription>
            {t(
              "The terms you agreed with this factor. A factor has one active agreement at a time, and each invoice uses the terms in force when it posted.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={FORM_ID}
            onSubmit={form.handleSubmit((values) => mutation.mutateAsync(values))}
            className="flex flex-col gap-6"
          >
            <FactoringAgreementFields control={form.control} />
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={FORM_ID} isLoading={mutation.isPending}>
            {t("Save changes")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** The agreement's terms, entered the way a factoring contract states them. */
function FactoringAgreementFields() {
  const t = useT();
  const { control } = useFormContext<FactoringAgreementFormValues>();
  const treatment = useWatch({ control, name: "accountingTreatment" });
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

  return (
    <>
      <FormSection title={t("Term")}>
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="effectiveFrom"
              label={t("Starts")}
              description={t("The first day invoices are funded on these terms.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="effectiveTo"
              label={t("Ends")}
              placeholder={t("Open-ended")}
              description={t("Leave blank while the contract runs until either side ends it.")}
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
              description={t("Who carries the loss when a customer never pays.")}
              options={recourseOptions}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="accountingTreatment"
              label={t("Accounting treatment")}
              description={t("How funding is booked: a sale of the invoice, or a loan against it.")}
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

      <FormSection
        title={t("Accounts")}
        description={t(
          "Where funding, reserve releases and chargebacks are booked. Nothing is booked for this factor until the accounts its treatment needs are set.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <GLAccountAutocompleteField
              control={control}
              name="feeExpenseAccountId"
              label={t("Factoring fee account")}
              placeholder={t("Select Factoring fee account")}
              description={t("Where the factor's fees are expensed.")}
              clearable
            />
          </FormControl>
          {treatment === "SecuredBorrowing" ? (
            <FormControl>
              <GLAccountAutocompleteField
                control={control}
                name="liabilityAccountId"
                label={t("Owed to factor")}
                placeholder={t("Select owed to factor account")}
                description={t(
                  "The liability for what the factor advanced, until the customer pays.",
                )}
                clearable
              />
            </FormControl>
          ) : (
            <FormControl>
              <GLAccountAutocompleteField
                control={control}
                name="dueFromFactorAccountId"
                label={t("Due from factor")}
                placeholder={t("Select Due from factor account")}
                description={t("The reserve the factor holds until the customer pays it.")}
                clearable
              />
            </FormControl>
          )}
          <FormControl cols="full">
            <GLAccountAutocompleteField
              control={control}
              name="cashAccountId"
              label={t("Cash account")}
              placeholder={t("Select Cash Account")}
              description={t(
                "Where the factor's payments land. Empty uses the organization's cash account.",
              )}
              clearable
            />
          </FormControl>
        </FormGroup>
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
            <TimeOfDayField
              control={control}
              name="submissionTime"
              label={t("Daily schedule time")}
              description={t(
                "When the day's schedule of ready invoices is proposed, in your organization's time zone.",
              )}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <DocumentTypeMultiSelectField
              control={control}
              name="documentTypes"
              label={t("Documents the factor needs")}
              placeholder={t("Select Document Types")}
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
              description={t("For your team only. Never sent to the factor or printed.")}
              rows={3}
              maxLength={MAX_AGREEMENT_NOTES}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
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
      <div className="grid grid-cols-[5rem_1fr_1fr_1fr_2rem] gap-2">
        {[
          { label: t("From day"), description: t("Where the band starts") },
          { label: t("Until day"), description: t("Where it ends; blank on the last") },
          { label: t("Fee rate"), description: t("Percent of the invoice") },
          { label: t("Flat fee"), description: t("Added to every invoice") },
        ].map((column) => (
          <span key={column.label} className="flex min-w-0 flex-col gap-0.5">
            <span className="text-muted-foreground text-xs font-medium">{column.label}</span>
            <span className="text-foreground-subtle text-2xs">{column.description}</span>
          </span>
        ))}
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
