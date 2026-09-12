import { useT } from "@trenova/shared/i18n/use-t";
import { Checkbox } from "@/components/animate-ui/components/base/checkbox";
import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { usePermissions } from "@/hooks/use-permission";
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
import { OANDAExchangeRatesIntegrationModal } from "@/routes/admin/integrations/_components/oanda/oanda-integration-modal";
import { apiService } from "@/services/api";
import type { AccountingControl, JournalSourceEvent } from "@/types/accounting-control";
import { accountingControlSchema } from "@/types/accounting-control";
import { Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { Settings2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { FormProvider, type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";

const OANDA_INTEGRATION_TYPE = "OANDAExchangeRates";

function sanitizeAccountingControlForSubmit(values: AccountingControl): AccountingControl {
  if (values.currencyMode !== "SingleCurrency") {
    return values;
  }

  return {
    ...values,
    exchangeRateOverridePolicy: "Disallow",
    realizedFxGainAccountId: null,
    realizedFxLossAccountId: null,
  };
}

export default function AccountingControlForm() {
  const t = useT();

  const { data } = useSuspenseQuery({
    ...queries.accountingControl.get(),
  });

  const form = useForm<AccountingControl>({
    resolver: zodResolver(accountingControlSchema) as Resolver<AccountingControl>,
    defaultValues: data,
  });

  const { handleSubmit, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.accountingControl.get._def,
    mutationFn: async (values: AccountingControl) =>
      apiService.accountingControlService.update(values),
    resourceName: "Accounting Control",
    resetForm: reset,
    form,
    invalidateQueries: [queries.accountingControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: AccountingControl) => {
      await mutateAsync(sanitizeAccountingControlForSubmit(values));
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <RecognitionPolicyCard />
          <JournalPolicyCard />
          <DriverSettlementPostingCard />
          <PeriodAndReconciliationCard />
          <CurrencyAndAccountsCard />
          <FormSaveDock saveButtonContent={t("Save Changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function RecognitionPolicyCard() {
  const t = useT();

  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Recognition Policy")}</CardTitle>
        <CardDescription>
          {t(
            "Define the organization accounting basis and the revenue and expense recognition policies that must remain compatible with that basis.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="accountingBasis"
              label={t("Accounting Basis")}
              description={t(
                "Sets the organization’s primary accounting basis and constrains the valid recognition policies.",
              )}
              options={accountingBasisChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="revenueRecognitionPolicy"
              label={t("Revenue Recognition Policy")}
              description={t(
                "Defines the event that recognizes revenue for organization-controlled accounting entries.",
              )}
              options={revenueRecognitionPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="expenseRecognitionPolicy"
              label={t("Expense Recognition Policy")}
              description={t(
                "Defines the event that recognizes expense for organization-controlled accounting entries.",
              )}
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
  const t = useT();

  const { control, getValues, setValue } = useFormContext<AccountingControl>();
  const journalPostingMode = useWatch({ control, name: "journalPostingMode" });
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
        <CardTitle>{t("Journal Policy")}</CardTitle>
        <CardDescription>
          {t(
            "Configure automatic journal creation, manual journal policy, and the chart-of-account defaults required for accounting automation.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="journalPostingMode"
              label={t("Journal Posting Mode")}
              description={t(
                "Controls whether journals are created only by explicit user action or automatically from configured source events.",
              )}
              options={journalPostingModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {journalPostingMode === "Automatic" && (
            <FormControl className="max-w-[720px]">
              <div className="flex flex-col gap-3">
                <Label className="text-sm font-medium">{t("Auto-Post Source Events")}</Label>
                <p className="text-muted-foreground text-sm">
                  {t(
                    "Select the posted business events that are allowed to generate journal entries automatically.",
                  )}
                </p>
                <FormGroup cols={2}>
                  {journalSourceEventChoices.map((option) => {
                    const checked = autoPostSourceEvents.includes(option.value);

                    return (
                      <label key={option.value} className="flex items-center gap-2 text-sm">
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(nextChecked) =>
                            toggleEvent(option.value, Boolean(nextChecked))
                          }
                        />
                        <span>{t(option.label)}</span>
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
              label={t("Manual Journal Entry Policy")}
              description={t(
                "Defines whether users may create manual journals broadly, only for adjustments, or not at all.",
              )}
              options={manualJournalEntryPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requireManualJeApproval"
              label={t("Require Manual JE Approval")}
              description={t(
                "Requires approval before an allowed manual journal entry can be finalized.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="journalReversalPolicy"
              label={t("Journal Reversal Policy")}
              description={t(
                "Defines whether posted journals can be reversed through workflow and, if allowed, where the reversal is booked.",
              )}
              options={journalReversalPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultRevenueAccountId"
              label={t("Default Revenue Account")}
              placeholder={t("Select revenue account")}
              description={t(
                "Default GL account used when automatic journal posting creates revenue entries.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultCashAccountId"
              label={t("Default Cash Account")}
              placeholder={t("Select cash account")}
              description={t(
                "GL account debited when customer payments are posted. Required to record customer payments.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultUnappliedCashAccountId"
              label={t("Default Unapplied Cash Account")}
              placeholder={t("Select unapplied cash account")}
              description={t(
                "Holding account credited for the unapplied portion of customer payments until it is applied to invoices. Required to record customer payments.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultExpenseAccountId"
              label={t("Default Expense Account")}
              placeholder={t("Select expense account")}
              description={t(
                "Default GL account used when automatic journal posting creates expense entries.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultArAccountId"
              label={t("Default AR Account")}
              placeholder={t("Select AR account")}
              description={t(
                "Default accounts receivable account for invoice-related journal posting.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultApAccountId"
              label={t("Default AP Account")}
              placeholder={t("Select AP account")}
              description={t(
                "Default accounts payable account for vendor-bill-related journal posting.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultTaxLiabilityAccountId"
              label={t("Default Tax Liability Account")}
              placeholder={t("Select tax liability account")}
              description={t(
                "Default liability account used when tax amounts are posted from accounting flows.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultWriteOffAccountId"
              label={t("Default Write-Off Account")}
              placeholder={t("Select write-off account")}
              description={t(
                "Default account used when approved write-offs are booked through adjustment workflows.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultRetainedEarningsAccountId"
              label={t("Default Retained Earnings Account")}
              placeholder={t("Select retained earnings account")}
              description={t(
                "Default retained earnings account used by closing and equity-related accounting processes.",
              )}
              clearable
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function DriverSettlementPostingCard() {
  const t = useT();

  const { control } = useFormContext<AccountingControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Driver Settlement Posting")}</CardTitle>
        <CardDescription>
          {t(
            "GL accounts used when driver settlements post to the ledger. These allocations feed the DriverWages and DriverBenefits cost categories, so cost-per-mile in Cost Control reflects actual driver pay.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultDriverPayExpenseAccountId"
              label={t("Driver Pay Expense Account")}
              placeholder={t("Select driver pay expense account")}
              description={t(
                "Expense account debited for company-driver earnings when a settlement posts. Required to post settlements.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultPurchasedTransportationAccountId"
              label={t("Purchased Transportation Account")}
              placeholder={t("Select purchased transportation account")}
              description={t(
                "Expense account debited for owner-operator earnings instead of driver pay expense. Required to post owner-operator settlements.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultDriverReimbursementAccountId"
              label={t("Driver Reimbursement Account")}
              placeholder={t("Select reimbursement account")}
              description={t(
                "Expense account debited for non-taxable reimbursements such as per diem and stipends; falls back to the driver pay expense account when unset.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultSettlementsPayableAccountId"
              label={t("Settlements Payable Account")}
              placeholder={t("Select settlements payable account")}
              description={t(
                "Liability account credited for the net pay owed to the driver until the settlement is paid. Required to post settlements.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultDriverAdvanceAccountId"
              label={t("Driver Advance Receivable Account")}
              placeholder={t("Select advance receivable account")}
              description={t(
                "Asset account tracking outstanding driver advances; credited when advances are recovered and debited for negative-balance carry-forwards.",
              )}
              clearable
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <GLAccountAutocompleteField
              control={control}
              name="defaultEscrowLiabilityAccountId"
              label={t("Escrow Liability Account")}
              placeholder={t("Select escrow liability account")}
              description={t(
                "Liability account credited for driver escrow contributions withheld from settlements.",
              )}
              clearable
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PeriodAndReconciliationCard() {
  const t = useT();

  const { control } = useFormContext<AccountingControl>();
  const reconciliationMode = useWatch({ control, name: "reconciliationMode" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Period And Reconciliation")}</CardTitle>
        <CardDescription>
          {t(
            "Define period-close automation, posting restrictions for locked and closed periods, and how reconciliation exceptions affect posting and close.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="periodCloseMode"
              label={t("Period Close Mode")}
              description={t(
                "Controls whether accounting periods are closed manually or by a scheduled system job.",
              )}
              options={periodCloseModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requirePeriodCloseApproval"
              label={t("Require Period Close Approval")}
              description={t(
                "Requires an approval step before a manually closed period can be finalized.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="lockedPeriodPostingPolicy"
              label={t("Locked Period Posting Policy")}
              description={t(
                "Defines how the system handles posting attempts into a locked accounting period.",
              )}
              options={lockedPeriodPostingPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="closedPeriodPostingPolicy"
              label={t("Closed Period Posting Policy")}
              description={t(
                "Defines whether posting to a closed period requires reopening or is redirected to the next open period.",
              )}
              options={closedPeriodPostingPolicyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <SelectField
              control={control}
              name="reconciliationMode"
              label={t("Reconciliation Mode")}
              description={t(
                "Controls whether reconciliation discrepancies are ignored, logged as warnings, or block posting.",
              )}
              options={reconciliationModeChoices}
              rules={{ required: true }}
            />
          </FormControl>
          {reconciliationMode !== "Disabled" && (
            <FormControl className="max-w-[420px]">
              <NumberField
                control={control}
                name="reconciliationToleranceAmount"
                label={t("Reconciliation Tolerance Amount")}
                description={t(
                  "Maximum allowed discrepancy amount before the configured reconciliation response applies.",
                )}
                rules={{ required: true }}
              />
            </FormControl>
          )}
          <FormControl>
            <SwitchField
              control={control}
              name="requireReconciliationToClose"
              label={t("Require Reconciliation To Close")}
              description={t(
                "Prevents period close while unresolved reconciliation discrepancies remain open.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyOnReconciliationException"
              label={t("Notify On Reconciliation Exception")}
              description={t("Sends notifications when a reconciliation discrepancy is recorded.")}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function CurrencyAndAccountsCard() {
  const t = useT();

  const { control, getValues, setValue } = useFormContext<AccountingControl>();
  const [isOANDAModalOpen, setIsOANDAModalOpen] = useState(false);
  const currencyMode = useWatch({ control, name: "currencyMode" });
  const isMultiCurrency = currencyMode === "MultiCurrency";
  const integrationPermissions = usePermissions(Resource.Integration);
  const canReadIntegrations = integrationPermissions.canRead;

  const runtimeConfigQuery = useQuery({
    ...queries.integration.runtimeConfig(OANDA_INTEGRATION_TYPE),
    enabled: isMultiCurrency,
    throwOnError: false,
  });

  useEffect(() => {
    if (currencyMode !== "SingleCurrency") {
      return;
    }

    if (getValues("exchangeRateOverridePolicy") !== "Disallow") {
      setValue("exchangeRateOverridePolicy", "Disallow", {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
    if (getValues("realizedFxGainAccountId")) {
      setValue("realizedFxGainAccountId", null, { shouldDirty: true, shouldValidate: true });
    }
    if (getValues("realizedFxLossAccountId")) {
      setValue("realizedFxLossAccountId", null, { shouldDirty: true, shouldValidate: true });
    }
  }, [currencyMode, getValues, setValue]);

  const oandaEnabled = runtimeConfigQuery.data?.enabled ?? false;
  const oandaConfigured = runtimeConfigQuery.data?.configured ?? false;
  const oandaReady = runtimeConfigQuery.data?.ready ?? false;
  const hasOANDAApiKey =
    runtimeConfigQuery.data?.missingRequiredFields.includes("apiKey") === false;
  const showCurrencyPolicy = isMultiCurrency && oandaReady;

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>{t("Currency Settings")}</CardTitle>
          <CardDescription>
            {t(
              "Configure the accounting currency mode and functional currency for financial reporting.",
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className="max-w-prose">
          <FormGroup cols={1}>
            {isMultiCurrency ? (
              <OANDAReadinessPanel
                enabled={oandaEnabled}
                configured={oandaConfigured}
                hasApiKey={hasOANDAApiKey}
                loading={runtimeConfigQuery.isLoading}
                canReadIntegrations={canReadIntegrations}
                ready={oandaReady}
                onConfigure={() => setIsOANDAModalOpen(true)}
              />
            ) : null}
            <FormControl className="max-w-[420px]">
              <SelectField
                control={control}
                name="currencyMode"
                label={t("Currency Mode")}
                description={t(
                  "Determines whether the organization operates in a single functional currency or supports foreign-currency transactions.",
                )}
                options={currencyModeChoices}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl className="max-w-[420px]">
              <SelectField
                control={control}
                name="functionalCurrencyCode"
                label={t("Functional Currency")}
                description={t(
                  "Base currency used for organization accounting and financial reporting.",
                )}
                options={currencyChoices}
                rules={{ required: true }}
              />
            </FormControl>
            {isMultiCurrency && showCurrencyPolicy && (
              <>
                <div className="flex flex-col gap-1 border-t pt-4">
                  <h3 className="text-sm font-medium">{t("Currency Policy")}</h3>
                  <p className="text-muted-foreground text-sm">
                    {t(
                      "Configure exchange-rate date selection, override handling, and realized FX accounts.",
                    )}
                  </p>
                </div>
                <FormControl className="max-w-[420px]">
                  <SelectField
                    control={control}
                    name="exchangeRateDatePolicy"
                    label={t("Exchange Rate Date Policy")}
                    description={t(
                      "Determines which date is used to select the exchange rate for multi-currency accounting.",
                    )}
                    options={exchangeRateDatePolicyChoices}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl className="max-w-[420px]">
                  <SelectField
                    control={control}
                    name="exchangeRateOverridePolicy"
                    label={t("Exchange Rate Override Policy")}
                    description={t(
                      "Controls whether users may override exchange rates and whether those overrides require approval.",
                    )}
                    options={exchangeRateOverridePolicyChoices}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl className="max-w-[420px]">
                  <GLAccountAutocompleteField
                    control={control}
                    name="realizedFxGainAccountId"
                    label={t("Realized FX Gain Account")}
                    placeholder={t("Select FX gain account")}
                    description={t(
                      "Default account for realized foreign exchange gains in multi-currency accounting.",
                    )}
                    clearable
                  />
                </FormControl>
                <FormControl className="max-w-[420px]">
                  <GLAccountAutocompleteField
                    control={control}
                    name="realizedFxLossAccountId"
                    label={t("Realized FX Loss Account")}
                    placeholder={t("Select FX loss account")}
                    description={t(
                      "Default account for realized foreign exchange losses in multi-currency accounting.",
                    )}
                    clearable
                  />
                </FormControl>
              </>
            )}
          </FormGroup>
        </CardContent>
      </Card>
      <OANDAExchangeRatesIntegrationModal
        open={isOANDAModalOpen && canReadIntegrations}
        onOpenChange={setIsOANDAModalOpen}
      />
    </>
  );
}

function OANDAReadinessPanel({
  enabled,
  configured,
  hasApiKey,
  loading,
  canReadIntegrations,
  ready,
  onConfigure,
}: {
  enabled: boolean;
  configured: boolean;
  hasApiKey: boolean;
  loading: boolean;
  canReadIntegrations: boolean;
  ready: boolean;
  onConfigure: () => void;
}) {
  const t = useT();

  const title = ready ? "OANDA connected" : "OANDA required";
  let description = "Connect OANDA to unlock multi-currency exchange-rate policy fields.";
  if (loading) {
    description = "Checking OANDA readiness.";
  } else if (ready) {
    description = "Multi-currency exchange-rate policy fields are available.";
  } else if (enabled && configured && hasApiKey) {
    description = "Enable OANDA to unlock multi-currency exchange-rate policy fields.";
  }

  return (
    <div className="border-border bg-background max-w-[420px] rounded-md border px-3.5 py-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <span
              className={
                ready
                  ? "bg-success size-1.5 shrink-0 rounded-full"
                  : "bg-muted-foreground/50 size-1.5 shrink-0 rounded-full"
              }
            />
            <p className="text-foreground text-sm font-medium">{title}</p>
          </div>
          <p className="text-muted-foreground text-sm">{description}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="w-fit shrink-0"
          onClick={onConfigure}
          disabled={!canReadIntegrations}
          title={!canReadIntegrations ? "Integration permission required" : "Configure OANDA"}
        >
          <Settings2 className="size-4" />
          {t("Configure")}
        </Button>
      </div>
    </div>
  );
}
