import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { FormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import {
  billingExceptionDispositionChoices,
  billingQueueTransferModeChoices,
  enforcementLevelChoices,
  invoiceDraftCreationModeChoices,
  invoicePostingModeChoices,
  lateChargeAssessmentModeChoices,
  paymentTermChoices,
  rateVarianceAutoResolutionModeChoices,
  readyToBillAssignmentModeChoices,
  transferScheduleChoices,
  unratedShipmentDispositionChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { BillingControl } from "@/types/billing-control";
import { billingControlSchema } from "@/types/billing-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";

export default function BillingControlForm() {
  const t = useT();

  const { data } = useSuspenseQuery({
    ...queries.billingControl.get(),
  });

  const form = useForm<BillingControl>({
    resolver: zodResolver(billingControlSchema) as Resolver<BillingControl>,
    defaultValues: data,
  });

  const { handleSubmit, reset } = form;

  const { mutateAsync } = useOptimisticMutation<
    BillingControl,
    BillingControl,
    unknown,
    BillingControl
  >({
    queryKey: queries.billingControl.get._def,
    mutationFn: async (values: BillingControl) => apiService.billingControlService.update(values),
    resourceName: "Billing Control",
    resetForm: reset,
    form,
    invalidateQueries: [queries.billingControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: BillingControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <InvoiceDefaultsCard />
          <AutomationCard />
          <ExceptionPolicyCard />
          <RatingPolicyCard />
          <LateChargesCard />
          <FormSaveDock saveButtonContent={t("Save changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function InvoiceDefaultsCard() {
  const t = useT();

  const { control } = useFormContext<BillingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Invoice defaults")}</CardTitle>
        <CardDescription>
          {t(
            "Set the organization-level invoice defaults used when customer-specific billing profile settings are not present.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="showDueDateOnInvoice"
              label={t("Show due date on invoice")}
              description={t("Displays the payment due date on customer-facing invoices.")}
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showBalanceDueOnInvoice"
              label={t("Show balance due on invoice")}
              description={t("Displays the outstanding balance due on customer-facing invoices.")}
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              name="defaultPaymentTerm"
              label={t("Default payment term")}
              description={t(
                "Fallback payment term used when a customer billing profile does not define one.",
              )}
              options={paymentTermChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="defaultInvoiceTerms"
              label={t("Default invoice terms")}
              placeholder={t("Payment, billing, and remittance terms")}
              description={t(
                "Default invoice terms text applied when customer-specific terms are not present.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="defaultInvoiceFooter"
              label={t("Default invoice footer")}
              placeholder={t("Footer content displayed on invoices")}
              description={t(
                "Default footer text shown on invoices when no customer-specific footer is configured.",
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AutomationCard() {
  const t = useT();

  const { control } = useFormContext<BillingControl>();
  const transferMode = useWatch({ control, name: "billingQueueTransferMode" });
  const draftCreationMode = useWatch({ control, name: "invoiceDraftCreationMode" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Automation policy")}</CardTitle>
        <CardDescription>
          {t(
            "Control how shipments move into billing, when invoice drafts are created, and whether posted invoices remain manual-review only or may auto-post when no blocking issues exist.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="readyToBillAssignmentMode"
              label={t("Ready-to-bill assignment mode")}
              description={t(
                "Controls whether eligible shipments are marked ready to bill automatically or only by user action.",
              )}
              options={readyToBillAssignmentModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="billingQueueTransferMode"
              label={t("Billing queue transfer mode")}
              description={t(
                "Controls whether ready-to-bill shipments enter the billing queue automatically or only by user action.",
              )}
              options={billingQueueTransferModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {transferMode === "AutomaticWhenReady" && (
            <>
              <FormControl className="max-w-[420px]">
                <SelectField
                  control={control}
                  name="billingQueueTransferSchedule"
                  label={t("Billing queue transfer schedule")}
                  description={t(
                    "Defines how frequently the automatic billing queue transfer job runs.",
                  )}
                  options={transferScheduleChoices}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="max-w-[420px]">
                <NumberField
                  control={control}
                  name="billingQueueTransferBatchSize"
                  label={t("Billing queue transfer batch size")}
                  description={t(
                    "Maximum number of ready items processed in a single automatic transfer batch.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
            </>
          )}
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="invoiceDraftCreationMode"
              label={t("Invoice draft creation mode")}
              description={t(
                "Controls whether invoice drafts are created only by users or automatically when items are transferred.",
              )}
              options={invoiceDraftCreationModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="invoicePostingMode"
              label={t("Invoice posting mode")}
              description={t(
                "Controls whether invoice posting always requires manual review or may auto-post when no blocking issues remain.",
              )}
              options={invoicePostingModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {draftCreationMode === "AutomaticWhenTransferred" && (
            <>
              <FormControl className="max-w-[420px]">
                <NumberField
                  control={control}
                  name="autoInvoiceBatchSize"
                  label={t("Auto invoice batch size")}
                  description={t(
                    "Maximum number of invoice drafts created in a single automatic batch.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="notifyOnAutoInvoiceCreation"
                  label={t("Notify on auto invoice creation")}
                  description={t(
                    "Sends notifications when invoice drafts are created automatically.",
                  )}
                  position="left"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ExceptionPolicyCard() {
  const t = useT();

  const { control } = useFormContext<BillingControl>();
  const shipmentRequirementEnforcement = useWatch({
    control,
    name: "shipmentBillingRequirementEnforcement",
  });
  const rateValidationEnforcement = useWatch({
    control,
    name: "rateValidationEnforcement",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Exception policy")}</CardTitle>
        <CardDescription>
          {t(
            "Define how shipment billing requirement failures and rate-variance validations affect billing progression, review routing, and blocking behavior.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="shipmentBillingRequirementEnforcement"
              label={t("Shipment billing requirement enforcement")}
              description={t(
                "Defines how missing shipment billing requirements affect readiness and billing progression.",
              )}
              options={enforcementLevelChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="rateValidationEnforcement"
              label={t("Rate validation enforcement")}
              description={t(
                "Defines how rate-variance validation results affect invoice workflow progression.",
              )}
              options={enforcementLevelChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {(shipmentRequirementEnforcement === "RequireReview" ||
            rateValidationEnforcement === "RequireReview") && (
            <FormControl className="max-w-[420px]">
              <SelectField
                control={control}
                name="billingExceptionDisposition"
                label={t("Billing exception disposition")}
                description={t(
                  "Determines whether review-required billing exceptions stay with billing or are returned to operations.",
                )}
                options={billingExceptionDispositionChoices}
                rules={{ required: true }}
              />
            </FormControl>
          )}
          <FormControl>
            <SwitchField
              control={control}
              name="notifyOnBillingExceptions"
              label={t("Notify on billing exceptions")}
              description={t("Sends notifications when billing exceptions are recorded.")}
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="rateVarianceTolerancePercent"
              label={t("Rate variance tolerance percent")}
              description={t(
                "Tolerance percentage used when evaluating whether a rate variance can bypass review.",
              )}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="rateVarianceAutoResolutionMode"
              label={t("Rate variance auto resolution mode")}
              description={t(
                "Controls whether review is skipped for rate variances that are within the configured tolerance.",
              )}
              options={rateVarianceAutoResolutionModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

/**
 * Whether the nightly late-charge run previews or raises debit memos. The
 * rate and grace period stay on each customer's billing profile; this card
 * only decides what the run is allowed to do with them.
 */
export function LateChargesCard() {
  const t = useT();

  const { control } = useFormContext<BillingControl>();
  const mode = useWatch({ control, name: "lateChargeAssessmentMode" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Late charges")}</CardTitle>
        <CardDescription>
          {t(
            "Overdue invoices are charged once per thirty-day period at the customer's late charge rate, after their grace period, as one debit memo per customer per run.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="lateChargeAssessmentMode"
              label={t("Assessment mode")}
              description={
                mode === "Preview"
                  ? t(
                      "The nightly run computes what it would raise and writes nothing; use the Late Charges page to raise memos by hand.",
                    )
                  : mode === "Automatic"
                    ? t(
                        "The nightly run raises the debit memos, posting them when invoice posting is automatic.",
                      )
                    : t(
                        "No late charges are assessed; the Late Charges page can still preview them.",
                      )
              }
              options={lateChargeAssessmentModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="lateChargeMinimumAmount"
              label={t("Minimum late charge")}
              aria-label={t("Minimum late charge")}
              description={t(
                "A customer whose late charges for a run add up to less than this is skipped.",
              )}
              decimalScale={2}
              fixedDecimalScale
              min={0}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function RatingPolicyCard() {
  const t = useT();

  const { control } = useFormContext<BillingControl>();

  const unratedShipmentDisposition = useWatch({
    control,
    name: "unratedShipmentDisposition",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Rating policy")}</CardTitle>
        <CardDescription>
          {t(
            "Decide what happens when no rate agreement covers a shipment's lane, and how manual rate overrides are governed.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="unratedShipmentDisposition"
              label={t("Unrated shipment disposition")}
              description={t(
                "What happens to a shipment no rate agreement covers. Falling back to a formula template is exactly how rating worked before agreements existed.",
              )}
              options={unratedShipmentDispositionChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {unratedShipmentDisposition === "FallbackFormulaTemplate" && (
            <FormControl className="max-w-[420px]">
              <FormulaTemplateAutocompleteField
                control={control}
                name="fallbackFormulaTemplateId"
                label={t("Fallback formula template")}
                placeholder={t("Select formula template")}
                clearable
                description={t(
                  "Used when an unrated shipment carries no formula template of its own. Leave empty to require one on the shipment.",
                )}
              />
            </FormControl>
          )}
          <FormControl>
            <SwitchField
              control={control}
              name="requireRateOverrideReason"
              label={t("Require rate override reason")}
              description={t(
                "A manual rate override must say why, so the audit trail explains the departure from the contract.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enforceMarginFloor"
              label={t("Enforce margin floor")}
              description={t(
                "Blocks rates that fall below an agreement's margin floor instead of only flagging them.",
              )}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
