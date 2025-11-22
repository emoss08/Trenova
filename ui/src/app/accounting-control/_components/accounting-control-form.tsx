import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form";
import { GLAccountAutocompleteField } from "@/components/ui/autocomplete-fields";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  expenseRecognitionMethodChoices,
  journalEntryCriteriaChoices,
  reconciliationThresholdActionChoices,
  revenueRecognitionMethodChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import {
  AccountingControlSchema,
  accountingControlSchema,
} from "@/lib/schemas/accounting-control-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import React, { useCallback } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";

export default function AccountingControlForm() {
  const accountingControl = useSuspenseQuery({
    ...queries.organization.getAccountingControl(),
  });

  const form = useForm({
    resolver: zodResolver(accountingControlSchema),
    defaultValues: accountingControl.data,
  });

  const {
    handleSubmit,
    setError,
    reset,
    formState: { errors },
  } = form;

  console.log("errors", errors);

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.organization.getAccountingControl._def,
    mutationFn: async (values: AccountingControlSchema) =>
      api.accountingControl.update(values),
    successMessage: "Accounting control updated successfully",
    resourceName: "Accounting Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.organization.getAccountingControl._def],
  });

  const onSubmit = useCallback(
    async (values: AccountingControlSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <AccountingControlFormInner>
          <JournalEntrySettingsForm />
          <PeriodManagementForm />
          <ReconciliationSettingsForm />
          <RecognitionSettingsForm />
          <TaxSettingsForm />
          <FormSaveDock />
        </AccountingControlFormInner>
      </Form>
    </FormProvider>
  );
}

function JournalEntrySettingsForm() {
  const { control } = useFormContext<AccountingControlSchema>();
  const autoCreateJournalEntries = useWatch({
    control,
    name: "autoCreateJournalEntries",
  });

  const restrictManualJournalEntries = useWatch({
    control,
    name: "restrictManualJournalEntries",
  });

  const requireJournalEntryApproval = useWatch({
    control,
    name: "requireJournalEntryApproval",
  });

  const showRestrictedWarning =
    autoCreateJournalEntries && restrictManualJournalEntries;
  const showApprovalRecommendation =
    autoCreateJournalEntries && !requireJournalEntryApproval;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Journal Entry Settings</CardTitle>
        <CardDescription>
          Configure automation rules, default accounts, and approval workflows
          for journal entry creation and management. Define when entries should
          be automatically generated, establish default GL accounts for revenue
          and expense transactions, and implement controls for manual entry
          restrictions, approval requirements, and reversal capabilities.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoCreateJournalEntries"
              label="Auto-Create Journal Entries"
              description="Automatically generate journal entries when transactions are processed."
              position="left"
            />
          </FormControl>
          {autoCreateJournalEntries && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <SelectField
                  control={control}
                  rules={{ required: autoCreateJournalEntries }}
                  name="journalEntryCriteria"
                  label="Creation Criteria"
                  description="Define when journal entries should be automatically created."
                  options={journalEntryCriteriaChoices}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: autoCreateJournalEntries }}
                  name="defaultRevenueAccountId"
                  label="Default Revenue Account"
                  placeholder="Select Default Revenue Account"
                  clearable
                  description="Primary account for posting revenue transactions."
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: autoCreateJournalEntries }}
                  name="defaultExpenseAccountId"
                  label="Default Expense Account"
                  placeholder="Select Default Expense Account"
                  clearable
                  description="Primary account for posting expense transactions."
                />
              </FormControl>
            </div>
          )}
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="restrictManualJournalEntries"
              label="Restrict Manual Entries"
              description="Prevent users from creating journal entries manually outside of automated workflows."
              position="left"
              warning={{
                show: showRestrictedWarning,
                message:
                  "This configuration is only recommended for highly regulated environments with strict compliance requirements.",
              }}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="requireJournalEntryApproval"
              label="Require Entry Approval"
              description="Journal entries must be approved before posting to the general ledger."
              position="left"
              recommended={showApprovalRecommendation && !showRestrictedWarning}
              tooltip={
                <div className="space-y-1">
                  <p className="text-[13px] font-medium">Enable Approval</p>
                  <p className="text-xs text-muted-foreground">
                    Consider enabling journal entry approval for better control
                    over auto-generated entries. This adds an additional review
                    step before entries are posted to the general ledger.
                  </p>
                </div>
              }
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableJournalEntryReversal"
              label="Enable Entry Reversals"
              description="Allow posted journal entries to be reversed with an offsetting entry."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PeriodManagementForm() {
  const { control } = useFormContext<AccountingControlSchema>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Period Management</CardTitle>
        <CardDescription>
          Control accounting period closures, posting restrictions, and
          end-of-period approval requirements. Manage whether transactions can
          be posted to closed periods, enforce approval workflows for period-end
          closing procedures, and enable automated period closure on a scheduled
          basis to maintain fiscal period integrity.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="allowPostingToClosedPeriods"
              label="Allow Posting to Closed Periods"
              description="Permit transactions to be posted to periods that have been closed."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="requirePeriodEndApproval"
              label="Require Period-End Approval"
              description="Period close operations must be reviewed and approved before finalizing."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoClosePeriods"
              label="Auto-Close Periods"
              description="Automatically close accounting periods on a scheduled basis."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ReconciliationSettingsForm() {
  const { control } = useFormContext<AccountingControlSchema>();
  const enableReconciliation = useWatch({
    control,
    name: "enableReconciliation",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reconciliation Settings</CardTitle>
        <CardDescription>
          Define variance thresholds, automated actions, and notification
          preferences for account reconciliation processes. Establish acceptable
          variance limits, determine system responses when thresholds are
          exceeded, configure whether period closures should be blocked by
          pending reconciliations, and enable alerting for discrepancy
          detection.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableReconciliation"
              label="Enable Reconciliation"
              description="Activate automated reconciliation workflows for transaction matching and variance detection."
              position="left"
            />
          </FormControl>
          {enableReconciliation && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  rules={{ required: enableReconciliation }}
                  name="reconciliationThreshold"
                  label="Variance Threshold"
                  description="Maximum acceptable variance percentage before triggering an action."
                  placeholder="Enter variance threshold"
                  sideText="%"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <SelectField
                  control={control}
                  options={reconciliationThresholdActionChoices}
                  rules={{ required: enableReconciliation }}
                  name="reconciliationThresholdAction"
                  label="Threshold Action"
                  description="Action to take when variance exceeds the defined threshold."
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <SwitchField
                  className="px-0!"
                  control={control}
                  name="haltOnPendingReconciliation"
                  label="Halt on Pending Reconciliation"
                  description="Block period close until all reconciliation items are resolved."
                  position="left"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <SwitchField
                  className="px-0!"
                  control={control}
                  name="enableReconciliationNotifications"
                  label="Enable Notifications"
                  description="Send alerts when reconciliation discrepancies are detected."
                  position="left"
                  rules={{ required: true }}
                />
              </FormControl>
            </div>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function RecognitionSettingsForm() {
  const { control } = useFormContext<AccountingControlSchema>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Revenue & Expense Recognition</CardTitle>
        <CardDescription>
          Configure recognition methods and timing preferences for revenue and
          expense transactions in accordance with accounting standards. Select
          between accrual and cash-basis accounting methods, determine whether
          revenue should be deferred until payment receipt, and specify how
          expenses are recorded relative to when they are incurred versus paid.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SelectField
              control={control}
              name="revenueRecognitionMethod"
              label="Revenue Recognition Method"
              description="Accounting method used to recognize revenue (e.g., accrual, cash basis)."
              options={revenueRecognitionMethodChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="deferRevenueUntilPaid"
              label="Defer Revenue Until Paid"
              description="Recognize revenue only when payment is received, not when invoiced."
              position="left"
              className="px-0!"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SelectField
              control={control}
              name="expenseRecognitionMethod"
              label="Expense Recognition Method"
              description="Accounting method used to recognize expenses (e.g., accrual, cash basis)."
              options={expenseRecognitionMethodChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="accrueExpenses"
              label="Accrue Expenses"
              description="Record expenses when incurred rather than when paid."
              position="left"
              className="px-0!"
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function TaxSettingsForm() {
  const { control } = useFormContext<AccountingControlSchema>();
  const enableAutomaticTaxCalculation = useWatch({
    control,
    name: "enableAutomaticTaxCalculation",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tax Settings</CardTitle>
        <CardDescription>
          Configure automatic tax calculation and default GL account assignment
          for tax-related transactions. Enable automated tax computation on
          applicable transactions and designate the primary general ledger
          account for posting tax liabilities, credits, and related entries.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAutomaticTaxCalculation"
              label="Enable Automatic Tax Calculation"
              description="Automatically calculate and apply applicable taxes to transactions."
              position="left"
            />
          </FormControl>
          {enableAutomaticTaxCalculation && (
            <FormControl className="pl-10">
              <GLAccountAutocompleteField
                control={control}
                rules={{ required: enableAutomaticTaxCalculation }}
                name="defaultTaxAccountId"
                label="Default Tax Account"
                placeholder="Select Default Tax Account"
                clearable
                description="Primary account for posting tax liabilities and credits."
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AccountingControlFormInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col gap-4 pb-14">{children}</div>;
}
