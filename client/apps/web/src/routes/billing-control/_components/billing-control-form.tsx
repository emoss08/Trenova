import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  billingExceptionDispositionChoices,
  billingQueueTransferModeChoices,
  enforcementLevelChoices,
  invoiceDraftCreationModeChoices,
  invoicePostingModeChoices,
  paymentTermChoices,
  rateVarianceAutoResolutionModeChoices,
  readyToBillAssignmentModeChoices,
  transferScheduleChoices,
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
  const { data } = useSuspenseQuery({
    ...queries.billingControl.get(),
  });

  const form = useForm<BillingControl>({
    resolver: zodResolver(billingControlSchema) as Resolver<BillingControl>,
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

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
    setFormError: setError,
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
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function InvoiceDefaultsCard() {
  const { control } = useFormContext<BillingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invoice Defaults</CardTitle>
        <CardDescription>
          Set the organization-level invoice defaults used when customer-specific billing profile
          settings are not present.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="showDueDateOnInvoice"
              label="Show Due Date On Invoice"
              description="Displays the payment due date on customer-facing invoices."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showBalanceDueOnInvoice"
              label="Show Balance Due On Invoice"
              description="Displays the outstanding balance due on customer-facing invoices."
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              name="defaultPaymentTerm"
              label="Default Payment Term"
              description="Fallback payment term used when a customer billing profile does not define one."
              options={paymentTermChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="defaultInvoiceTerms"
              label="Default Invoice Terms"
              placeholder="Payment, billing, and remittance terms"
              description="Default invoice terms text applied when customer-specific terms are not present."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="defaultInvoiceFooter"
              label="Default Invoice Footer"
              placeholder="Footer content displayed on invoices"
              description="Default footer text shown on invoices when no customer-specific footer is configured."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AutomationCard() {
  const { control } = useFormContext<BillingControl>();
  const transferMode = useWatch({ control, name: "billingQueueTransferMode" });
  const draftCreationMode = useWatch({ control, name: "invoiceDraftCreationMode" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Automation Policy</CardTitle>
        <CardDescription>
          Control how shipments move into billing, when invoice drafts are created, and whether
          posted invoices remain manual-review only or may auto-post when no blocking issues exist.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="readyToBillAssignmentMode"
              label="Ready-To-Bill Assignment Mode"
              description="Controls whether eligible shipments are marked ready to bill automatically or only by user action."
              options={readyToBillAssignmentModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="billingQueueTransferMode"
              label="Billing Queue Transfer Mode"
              description="Controls whether ready-to-bill shipments enter the billing queue automatically or only by user action."
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
                  label="Billing Queue Transfer Schedule"
                  description="Defines how frequently the automatic billing queue transfer job runs."
                  options={transferScheduleChoices}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="max-w-[420px]">
                <NumberField
                  control={control}
                  name="billingQueueTransferBatchSize"
                  label="Billing Queue Transfer Batch Size"
                  description="Maximum number of ready items processed in a single automatic transfer batch."
                  rules={{ required: true }}
                />
              </FormControl>
            </>
          )}
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="invoiceDraftCreationMode"
              label="Invoice Draft Creation Mode"
              description="Controls whether invoice drafts are created only by users or automatically when items are transferred."
              options={invoiceDraftCreationModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="invoicePostingMode"
              label="Invoice Posting Mode"
              description="Controls whether invoice posting always requires manual review or may auto-post when no blocking issues remain."
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
                  label="Auto Invoice Batch Size"
                  description="Maximum number of invoice drafts created in a single automatic batch."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="notifyOnAutoInvoiceCreation"
                  label="Notify On Auto Invoice Creation"
                  description="Sends notifications when invoice drafts are created automatically."
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
        <CardTitle>Exception Policy</CardTitle>
        <CardDescription>
          Define how shipment billing requirement failures and rate-variance validations affect
          billing progression, review routing, and blocking behavior.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="shipmentBillingRequirementEnforcement"
              label="Shipment Billing Requirement Enforcement"
              description="Defines how missing shipment billing requirements affect readiness and billing progression."
              options={enforcementLevelChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="rateValidationEnforcement"
              label="Rate Validation Enforcement"
              description="Defines how rate-variance validation results affect invoice workflow progression."
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
                  label="Billing Exception Disposition"
                  description="Determines whether review-required billing exceptions stay with billing or are returned to operations."
                  options={billingExceptionDispositionChoices}
                  rules={{ required: true }}
                />
              </FormControl>
          )}
          <FormControl>
            <SwitchField
              control={control}
              name="notifyOnBillingExceptions"
              label="Notify On Billing Exceptions"
              description="Sends notifications when billing exceptions are recorded."
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="rateVarianceTolerancePercent"
              label="Rate Variance Tolerance Percent"
              description="Tolerance percentage used when evaluating whether a rate variance can bypass review."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="rateVarianceAutoResolutionMode"
              label="Rate Variance Auto Resolution Mode"
              description="Controls whether review is skipped for rate variances that are within the configured tolerance."
              options={rateVarianceAutoResolutionModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
