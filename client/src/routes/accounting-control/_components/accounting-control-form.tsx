import { Checkbox } from "@/components/animate-ui/components/base/checkbox";
import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  accountingMethodChoices,
  currencyChoices,
  expenseRecognitionMethodChoices,
  journalEntryCriteriaChoices,
  reconciliationThresholdActionChoices,
  revenueRecognitionMethodChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AccountingControl, JournalEntryCriteria } from "@/types/accounting-control";
import { accountingControlSchema } from "@/types/accounting-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, useWatch } from "react-hook-form";

export default function AccountingControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.accountingControl.get(),
  });

  const form = useForm({
    resolver: zodResolver(accountingControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.accountingControl.get._def,
    mutationFn: async (values: AccountingControl) =>
      apiService.accountingControlService.update(values),
    resourceName: "Accounting Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.accountingControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: AccountingControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <AccountingControlFormInner>
          <AccountingMethodForm />
          <JournalEntrySettingsForm />
          <PeriodManagementForm />
          <ReconciliationSettingsForm />
          <RecognitionSettingsForm />
          <TaxSettingsForm />
          <MultiCurrencyForm />
          <AuditComplianceForm />
          <FormSaveDock saveButtonContent="Save Changes" />
        </AccountingControlFormInner>
      </Form>
    </FormProvider>
  );
}

function AccountingMethodForm() {
  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Accounting Method</CardTitle>
        <CardDescription>
          Select the primary accounting method used by your organization. This determines which
          revenue and expense recognition options are available and constrains downstream settings
          to ensure compliance with the selected methodology.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <SelectField
              control={control}
              name="accountingMethod"
              label="Accounting Method"
              description="Primary accounting methodology for your organization."
              options={accountingMethodChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function JournalEntrySettingsForm() {
  const { control, setValue, getValues } = useFormContext<AccountingControl>();
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

  const journalEntryCriteria = useWatch({
    control,
    name: "journalEntryCriteria",
  });

  const showRestrictedWarning = autoCreateJournalEntries && restrictManualJournalEntries;
  const showApprovalRecommendation = autoCreateJournalEntries && !requireJournalEntryApproval;

  const handleCriteriaToggle = useCallback(
    (value: JournalEntryCriteria, checked: boolean) => {
      const current = getValues("journalEntryCriteria") ?? [];
      if (checked) {
        setValue("journalEntryCriteria", [...current, value], { shouldDirty: true });
      } else {
        setValue(
          "journalEntryCriteria",
          current.filter((v) => v !== value),
          { shouldDirty: true },
        );
      }
    },
    [getValues, setValue],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Journal Entry Settings</CardTitle>
        <CardDescription>
          Configure automation rules, default accounts, and approval workflows for journal entry
          creation and management. Define when entries should be automatically generated, establish
          default GL accounts for revenue and expense transactions, and implement controls for
          manual entry restrictions, approval requirements, and reversal capabilities.
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
                <div className="flex flex-col gap-2">
                  <Label className="text-sm font-medium">Creation Criteria</Label>
                  <p className="text-xs text-muted-foreground">
                    Select when journal entries should be automatically created.
                  </p>
                  <FormGroup cols={2} className="mb-2">
                    {journalEntryCriteriaChoices.map((option) => {
                      const isChecked = journalEntryCriteria?.includes(
                        option.value as JournalEntryCriteria,
                      );

                      return (
                        <label
                          key={option.value}
                          className="flex cursor-pointer items-center gap-2"
                        >
                          <Checkbox
                            checked={isChecked}
                            onCheckedChange={(checked) =>
                              handleCriteriaToggle(option.value as JournalEntryCriteria, !!checked)
                            }
                          />
                          <span className="text-sm">{option.label}</span>
                        </label>
                      );
                    })}
                  </FormGroup>
                </div>
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
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: autoCreateJournalEntries }}
                  name="defaultArAccountId"
                  label="Default AR Account"
                  placeholder="Select Default AR Account"
                  clearable
                  description="Primary account for posting accounts receivable transactions."
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: autoCreateJournalEntries }}
                  name="defaultApAccountId"
                  label="Default AP Account"
                  placeholder="Select Default AP Account"
                  clearable
                  description="Primary account for posting accounts payable transactions."
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  name="defaultCostOfServiceAccountId"
                  label="Default Cost of Service Account"
                  placeholder="Select Default Cost of Service Account"
                  clearable
                  description="Primary account for posting cost of service transactions."
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
                  <p className="text-xs text-background/80">
                    Consider enabling journal entry approval for better control over auto-generated
                    entries. This adds an additional review step before entries are posted to the
                    general ledger.
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
  const { control } = useFormContext<AccountingControl>();
  const autoClosePeriods = useWatch({
    control,
    name: "autoClosePeriods",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Period Management</CardTitle>
        <CardDescription>
          Control accounting period closures, posting restrictions, and end-of-period approval
          requirements. Manage whether transactions can be posted to closed periods, enforce
          approval workflows for period-end closing procedures, and enable automated period closure
          on a scheduled basis to maintain fiscal period integrity.
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
          {autoClosePeriods && (
            <div className="pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: autoClosePeriods }}
                  name="defaultRetainedEarningsAccountId"
                  label="Default Retained Earnings Account"
                  placeholder="Select Default Retained Earnings Account"
                  clearable
                  description="Account for posting retained earnings when periods are automatically closed."
                />
              </FormControl>
            </div>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ReconciliationSettingsForm() {
  const { control } = useFormContext<AccountingControl>();
  const enableReconciliation = useWatch({
    control,
    name: "enableReconciliation",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reconciliation Settings</CardTitle>
        <CardDescription>
          Define variance thresholds, automated actions, and notification preferences for account
          reconciliation processes. Establish acceptable variance limits, determine system responses
          when thresholds are exceeded, configure whether period closures should be blocked by
          pending reconciliations, and enable alerting for discrepancy detection.
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
  const { control } = useFormContext<AccountingControl>();
  const deferRevenueUntilPaid = useWatch({
    control,
    name: "deferRevenueUntilPaid",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Revenue & Expense Recognition</CardTitle>
        <CardDescription>
          Configure recognition methods and timing preferences for revenue and expense transactions
          in accordance with accounting standards. Select between accrual and cash-basis accounting
          methods, determine whether revenue should be deferred until payment receipt, and specify
          how expenses are recorded relative to when they are incurred versus paid.
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
          {deferRevenueUntilPaid && (
            <div className="pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: deferRevenueUntilPaid }}
                  name="defaultDeferredRevenueAccountId"
                  label="Default Deferred Revenue Account"
                  placeholder="Select Default Deferred Revenue Account"
                  clearable
                  description="Account for posting deferred revenue until payment is received."
                />
              </FormControl>
            </div>
          )}
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
  const { control } = useFormContext<AccountingControl>();
  const enableAutomaticTaxCalculation = useWatch({
    control,
    name: "enableAutomaticTaxCalculation",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tax Settings</CardTitle>
        <CardDescription>
          Configure automatic tax calculation and default GL account assignment for tax-related
          transactions. Enable automated tax computation on applicable transactions and designate
          the primary general ledger account for posting tax liabilities, credits, and related
          entries.
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

function MultiCurrencyForm() {
  const { control } = useFormContext<AccountingControl>();
  const enableMultiCurrency = useWatch({
    control,
    name: "enableMultiCurrency",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Multi-Currency</CardTitle>
        <CardDescription>
          Configure multi-currency support for handling transactions in foreign currencies. Enable
          currency conversion, set the default currency, and designate GL accounts for recording
          currency exchange gains and losses.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableMultiCurrency"
              label="Enable Multi-Currency"
              description="Allow transactions to be processed in multiple currencies."
              position="left"
            />
          </FormControl>
          {enableMultiCurrency && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <SelectField
                  control={control}
                  name="defaultCurrencyCode"
                  label="Default Currency"
                  description="Primary currency for your organization."
                  options={currencyChoices}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: enableMultiCurrency }}
                  name="currencyGainAccountId"
                  label="Currency Gain Account"
                  placeholder="Select Currency Gain Account"
                  clearable
                  description="Account for posting unrealized and realized currency exchange gains."
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <GLAccountAutocompleteField
                  control={control}
                  rules={{ required: enableMultiCurrency }}
                  name="currencyLossAccountId"
                  label="Currency Loss Account"
                  placeholder="Select Currency Loss Account"
                  clearable
                  description="Account for posting unrealized and realized currency exchange losses."
                />
              </FormControl>
            </div>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AuditComplianceForm() {
  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Audit & Compliance</CardTitle>
        <CardDescription>
          Configure audit trail and compliance settings for accounting operations. Control document
          attachment requirements and data retention policies for deleted entries.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="requireDocumentAttachment"
              label="Require Document Attachment"
              description="Require supporting documentation to be attached to journal entries before posting."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="retainDeletedEntries"
              label="Retain Deleted Entries"
              description="Preserve soft-deleted journal entries for audit trail purposes instead of permanent removal."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AccountingControlFormInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4 pb-14">{children}</div>;
}
