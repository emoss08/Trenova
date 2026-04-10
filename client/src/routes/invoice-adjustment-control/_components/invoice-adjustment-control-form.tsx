import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  adjustmentAccountingDatePolicyChoices,
  adjustmentAttachmentPolicyChoices,
  adjustmentEligibilityPolicyChoices,
  approvalPolicyChoices,
  closedPeriodAdjustmentPolicyChoices,
  customerCreditBalancePolicyChoices,
  overCreditPolicyChoices,
  replacementInvoiceReviewPolicyChoices,
  requirementPolicyChoices,
  supersededInvoiceVisibilityPolicyChoices,
  writeOffApprovalPolicyChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  type InvoiceAdjustmentControl,
  invoiceAdjustmentControlSchema,
} from "@/types/invoice-adjustment-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";

export default function InvoiceAdjustmentControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.invoiceAdjustmentControl.get(),
  });

  const form = useForm<InvoiceAdjustmentControl>({
    resolver: zodResolver(invoiceAdjustmentControlSchema) as Resolver<InvoiceAdjustmentControl>,
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation<
    InvoiceAdjustmentControl,
    InvoiceAdjustmentControl,
    unknown,
    InvoiceAdjustmentControl
  >({
    queryKey: queries.invoiceAdjustmentControl.get._def,
    mutationFn: async (values: InvoiceAdjustmentControl) =>
      apiService.invoiceAdjustmentControlService.update(values),
    resourceName: "Invoice Adjustment Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.invoiceAdjustmentControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: InvoiceAdjustmentControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <EligibilityCard />
          <DocumentationCard />
          <ApprovalCard />
          <CreditAndVisibilityCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function EligibilityCard() {
  const { control } = useFormContext<InvoiceAdjustmentControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Eligibility Policy</CardTitle>
        <CardDescription>
          Define which invoice states may be adjusted and how accounting dates are assigned when
          credits, rebills, or related adjustments are created.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="partiallyPaidInvoiceAdjustmentPolicy"
              label="Partially Paid Invoice Adjustment Policy"
              description="Controls whether partially paid invoices can be adjusted and whether approval is required."
              options={adjustmentEligibilityPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="paidInvoiceAdjustmentPolicy"
              label="Paid Invoice Adjustment Policy"
              description="Controls whether fully paid invoices may be adjusted through the formal adjustment workflow."
              options={adjustmentEligibilityPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="disputedInvoiceAdjustmentPolicy"
              label="Disputed Invoice Adjustment Policy"
              description="Controls whether disputed invoices can be adjusted and whether approval is required."
              options={adjustmentEligibilityPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="adjustmentAccountingDatePolicy"
              label="Adjustment Accounting Date Policy"
              description="Defines whether adjustments use the original invoice accounting date when open or always book in the next open period."
              options={adjustmentAccountingDatePolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[520px]">
            <SelectField
              control={control}
              name="closedPeriodAdjustmentPolicy"
              label="Closed Period Adjustment Policy"
              description="Defines whether closed-period adjustments are disallowed, require reopen, or must post in the next open period with approval."
              options={closedPeriodAdjustmentPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="rerateVarianceTolerancePercent"
              label="Rerate Variance Tolerance Percent"
              description="Tolerance percentage used when comparing rerated replacement invoice economics to the superseded invoice."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[520px]">
            <SelectField
              control={control}
              name="replacementInvoiceReviewPolicy"
              label="Replacement Invoice Review Policy"
              description="Defines when a replacement invoice must be reviewed after a credit and rebill workflow changes economic terms."
              options={replacementInvoiceReviewPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function DocumentationCard() {
  const { control } = useFormContext<InvoiceAdjustmentControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Documentation Requirements</CardTitle>
        <CardDescription>
          Define the minimum supporting documentation and business justification required before an
          adjustment can be completed.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="adjustmentReasonRequirement"
              label="Adjustment Reason Requirement"
              description="Determines whether a structured reason is mandatory before an adjustment can be completed."
              options={requirementPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[520px]">
            <SelectField
              control={control}
              name="adjustmentAttachmentRequirement"
              label="Adjustment Attachment Requirement"
              description="Defines the organization default for when supporting documents are required for invoice adjustments. Customer billing profiles may override this when they set an explicit supporting-document policy."
              options={adjustmentAttachmentPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ApprovalCard() {
  const { control } = useFormContext<InvoiceAdjustmentControl>();
  const standardApprovalPolicy = useWatch({ control, name: "standardAdjustmentApprovalPolicy" });
  const writeOffApprovalPolicy = useWatch({ control, name: "writeOffApprovalPolicy" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Approval Policy</CardTitle>
        <CardDescription>
          Define which adjustments require approval and where amount thresholds apply for standard
          adjustments and write-offs.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="standardAdjustmentApprovalPolicy"
              label="Standard Adjustment Approval Policy"
              description="Defines whether standard invoice adjustments require approval always, never, or only above a configured threshold."
              options={approvalPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {standardApprovalPolicy === "AmountThreshold" && (
            <FormControl className="max-w-[420px]">
              <NumberField
                control={control}
                name="standardAdjustmentApprovalThreshold"
                label="Standard Adjustment Approval Threshold"
                description="Adjustment amount above which approval is required when the standard approval policy uses an amount threshold."
                rules={{ required: true }}
              />
            </FormControl>
          )}
          <FormControl className="max-w-[520px]">
            <SelectField
              control={control}
              name="writeOffApprovalPolicy"
              label="Write-Off Approval Policy"
              description="Defines whether write-offs are disallowed, always require approval, or require approval only above a threshold."
              options={writeOffApprovalPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {writeOffApprovalPolicy === "RequireApprovalAboveThreshold" && (
            <FormControl className="max-w-[420px]">
              <NumberField
                control={control}
                name="writeOffApprovalThreshold"
                label="Write-Off Approval Threshold"
                description="Write-off amount above which approval is required when the write-off approval policy uses an amount threshold."
                rules={{ required: true }}
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function CreditAndVisibilityCard() {
  const { control } = useFormContext<InvoiceAdjustmentControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credit And Visibility</CardTitle>
        <CardDescription>
          Define whether unapplied customer credits are allowed, whether over-crediting can occur,
          and what external users may see after invoice replacement.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="customerCreditBalancePolicy"
              label="Customer Credit Balance Policy"
              description="Defines whether invoice adjustments may leave an unapplied customer credit balance."
              options={customerCreditBalancePolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="overCreditPolicy"
              label="Over-Credit Policy"
              description="Controls unapplied customer-credit outcomes caused by payment state and does not permit credit beyond true eligible invoice line or item scope."
              options={overCreditPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[520px]">
            <SelectField
              control={control}
              name="supersededInvoiceVisibilityPolicy"
              label="Superseded Invoice Visibility Policy"
              description="Defines whether external customer-facing views show only the current invoice or also expose superseded invoices with status."
              options={supersededInvoiceVisibilityPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
