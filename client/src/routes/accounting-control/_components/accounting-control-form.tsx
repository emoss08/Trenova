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
  accountingBasisChoices,
  closedPeriodPostingPolicyChoices,
  currencyChoices,
  currencyModeChoices,
  exchangeRateDatePolicyChoices,
  exchangeRateOverridePolicyChoices,
  expenseRecognitionPolicyChoices,
  journalPostingModeChoices,
  journalReversalPolicyChoices,
  journalSourceEventChoices,
  lockedPeriodPostingPolicyChoices,
  manualJournalEntryPolicyChoices,
  periodCloseModeChoices,
  reconciliationModeChoices,
  revenueRecognitionPolicyChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  AccountingControl,
  JournalSourceEvent,
} from "@/types/accounting-control";
import { accountingControlSchema } from "@/types/accounting-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";

export default function AccountingControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.accountingControl.get(),
  });

  const form = useForm<AccountingControl>({
    resolver: zodResolver(accountingControlSchema) as Resolver<AccountingControl>,
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
        <div className="flex flex-col gap-4 pb-14">
          <RecognitionPolicyCard />
          <JournalPolicyCard />
          <PeriodAndReconciliationCard />
          <CurrencyAndAccountsCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function RecognitionPolicyCard() {
  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recognition Policy</CardTitle>
        <CardDescription>
          Define the organization accounting basis and the revenue and expense recognition policies
          that must remain compatible with that basis.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="accountingBasis"
              label="Accounting Basis"
              description="Sets the organization’s primary accounting basis and constrains the valid recognition policies."
              options={accountingBasisChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="revenueRecognitionPolicy"
              label="Revenue Recognition Policy"
              description="Defines the event that recognizes revenue for organization-controlled accounting entries."
              options={revenueRecognitionPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="expenseRecognitionPolicy"
              label="Expense Recognition Policy"
              description="Defines the event that recognizes expense for organization-controlled accounting entries."
              options={expenseRecognitionPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function JournalPolicyCard() {
  const { control, getValues, setValue } = useFormContext<AccountingControl>();
  const journalPostingMode = useWatch({ control, name: "journalPostingMode" });
  const currencyMode = useWatch({ control, name: "currencyMode" });
  const autoPostSourceEvents = useWatch({ control, name: "autoPostSourceEvents" }) ?? [];

  const toggleEvent = useCallback(
    (value: JournalSourceEvent, checked: boolean) => {
      const current = getValues("autoPostSourceEvents") ?? [];
      const next = checked ? [...current, value] : current.filter((item) => item !== value);
      setValue("autoPostSourceEvents", next, { shouldDirty: true });
    },
    [getValues, setValue],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Journal Policy</CardTitle>
        <CardDescription>
          Configure automatic journal creation, manual journal policy, and the chart-of-account
          defaults required for accounting automation.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="journalPostingMode"
              label="Journal Posting Mode"
              description="Controls whether journals are created only by explicit user action or automatically from configured source events."
              options={journalPostingModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {journalPostingMode === "Automatic" && (
            <FormControl className="max-w-[720px]">
              <div className="flex flex-col gap-3">
                <Label className="text-sm font-medium">Auto-Post Source Events</Label>
                <p className="text-sm text-muted-foreground">
                  Select the posted business events that are allowed to generate journal entries automatically.
                </p>
                <FormGroup cols={2}>
                  {journalSourceEventChoices.map((option) => {
                    const checked = autoPostSourceEvents.includes(option.value);

                    return (
                      <label
                        key={option.value}
                        className="flex items-center gap-2 text-sm"
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(nextChecked) =>
                            toggleEvent(option.value, Boolean(nextChecked))
                          }
                        />
                        <span>{option.label}</span>
                      </label>
                    );
                  })}
                </FormGroup>
              </div>
            </FormControl>
          )}
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="manualJournalEntryPolicy"
              label="Manual Journal Entry Policy"
              description="Defines whether users may create manual journals broadly, only for adjustments, or not at all."
              options={manualJournalEntryPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requireManualJeApproval"
              label="Require Manual JE Approval"
              description="Requires approval before an allowed manual journal entry can be finalized."
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="journalReversalPolicy"
              label="Journal Reversal Policy"
              description="Defines whether posted journals can be reversed through workflow and, if allowed, where the reversal is booked."
              options={journalReversalPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultRevenueAccountId"
              label="Default Revenue Account"
              placeholder="Select revenue account"
              description="Default GL account used when automatic journal posting creates revenue entries."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultExpenseAccountId"
              label="Default Expense Account"
              placeholder="Select expense account"
              description="Default GL account used when automatic journal posting creates expense entries."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultArAccountId"
              label="Default AR Account"
              placeholder="Select AR account"
              description="Default accounts receivable account for invoice-related journal posting."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultApAccountId"
              label="Default AP Account"
              placeholder="Select AP account"
              description="Default accounts payable account for vendor-bill-related journal posting."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultTaxLiabilityAccountId"
              label="Default Tax Liability Account"
              placeholder="Select tax liability account"
              description="Default liability account used when tax amounts are posted from accounting flows."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultWriteOffAccountId"
              label="Default Write-Off Account"
              placeholder="Select write-off account"
              description="Default account used when approved write-offs are booked through adjustment workflows."
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultRetainedEarningsAccountId"
              label="Default Retained Earnings Account"
              placeholder="Select retained earnings account"
              description="Default retained earnings account used by closing and equity-related accounting processes."
              clearable
            />
          </FormControl>
          {currencyMode === "MultiCurrency" && (
            <>
              <FormControl className="max-w-[420px]">
                <GLAccountAutocompleteField
                  control={control}
                  name="realizedFxGainAccountId"
                  label="Realized FX Gain Account"
                  placeholder="Select FX gain account"
                  description="Default account for realized foreign exchange gains in multi-currency accounting."
                  clearable
                />
              </FormControl>
              <FormControl className="max-w-[420px]">
                <GLAccountAutocompleteField
                  control={control}
                  name="realizedFxLossAccountId"
                  label="Realized FX Loss Account"
                  placeholder="Select FX loss account"
                  description="Default account for realized foreign exchange losses in multi-currency accounting."
                  clearable
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PeriodAndReconciliationCard() {
  const { control } = useFormContext<AccountingControl>();
  const reconciliationMode = useWatch({ control, name: "reconciliationMode" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Period And Reconciliation</CardTitle>
        <CardDescription>
          Define period-close automation, posting restrictions for locked and closed periods, and
          how reconciliation exceptions affect posting and close.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="periodCloseMode"
              label="Period Close Mode"
              description="Controls whether accounting periods are closed manually or by a scheduled system job."
              options={periodCloseModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requirePeriodCloseApproval"
              label="Require Period Close Approval"
              description="Requires an approval step before a manually closed period can be finalized."
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="lockedPeriodPostingPolicy"
              label="Locked Period Posting Policy"
              description="Defines how the system handles posting attempts into a locked accounting period."
              options={lockedPeriodPostingPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="closedPeriodPostingPolicy"
              label="Closed Period Posting Policy"
              description="Defines whether posting to a closed period requires reopening or is redirected to the next open period."
              options={closedPeriodPostingPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="reconciliationMode"
              label="Reconciliation Mode"
              description="Controls whether reconciliation discrepancies are ignored, logged as warnings, or block posting."
              options={reconciliationModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {reconciliationMode !== "Disabled" && (
            <FormControl className="max-w-[420px]">
              <NumberField
                control={control}
                name="reconciliationToleranceAmount"
                label="Reconciliation Tolerance Amount"
                description="Maximum allowed discrepancy amount before the configured reconciliation response applies."
                rules={{ required: true }}
              />
            </FormControl>
          )}
          <FormControl>
            <SwitchField
              control={control}
              name="requireReconciliationToClose"
              label="Require Reconciliation To Close"
              description="Prevents period close while unresolved reconciliation discrepancies remain open."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyOnReconciliationException"
              label="Notify On Reconciliation Exception"
              description="Sends notifications when a reconciliation discrepancy is recorded."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function CurrencyAndAccountsCard() {
  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Currency Policy</CardTitle>
        <CardDescription>
          Configure functional currency, exchange-rate date selection, and override handling for
          single-currency or multi-currency accounting.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="currencyMode"
              label="Currency Mode"
              description="Determines whether the organization operates in a single functional currency or supports foreign-currency transactions."
              options={currencyModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="functionalCurrencyCode"
              label="Functional Currency"
              description="Base currency used for organization accounting and financial reporting."
              options={currencyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="exchangeRateDatePolicy"
              label="Exchange Rate Date Policy"
              description="Determines which date is used to select the exchange rate for multi-currency accounting."
              options={exchangeRateDatePolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="exchangeRateOverridePolicy"
              label="Exchange Rate Override Policy"
              description="Controls whether users may override exchange rates and whether those overrides require approval."
              options={exchangeRateOverridePolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
