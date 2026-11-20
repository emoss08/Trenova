import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  accountingBasisChoices,
  expenseRecognitionPolicyChoices,
  journalPostingModeChoices,
  journalReversalPolicyChoices,
  journalSourceEventChoices,
  journalSourceEventGroups,
  manualJournalEntryPolicyChoices,
  revenueRecognitionPolicyChoices,
} from "@/lib/choices";
import {
  journalPostingModeSchema,
  manualJournalEntryPolicySchema,
  type AccountingControl,
} from "@/types/accounting-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import {
  blockedSourceEvents,
  RECOGNITION_BY_BASIS,
  requiredSourceEvents,
  sourceEventsForRecognition,
} from "../accounting-recognition";

const CHANGED = { shouldDirty: true, shouldValidate: true } as const;

const REASONS = defineLabels({
  cashRevenue: "Revenue is recognized on cash receipt, not from invoices.",
});

/** When revenue and expense are recognized, and how journals are made and undone. */
export function PostingTab() {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<AccountingControl>();
  const basis = useWatch({ control, name: "accountingBasis" });
  const revenuePolicy = useWatch({ control, name: "revenueRecognitionPolicy" });
  const postingMode = useWatch({ control, name: "journalPostingMode" });
  const manualPolicy = useWatch({ control, name: "manualJournalEntryPolicy" });
  const automatic = postingMode === journalPostingModeSchema.enum.Automatic;

  const recognition = RECOGNITION_BY_BASIS[basis];
  const revenueOptions = revenueRecognitionPolicyChoices.filter(
    (choice) => choice.value === recognition?.revenue,
  );
  const expenseOptions = expenseRecognitionPolicyChoices.filter(
    (choice) => choice.value === recognition?.expense,
  );

  const blocked = new Set(blockedSourceEvents(revenuePolicy));
  const eventGroups = journalSourceEventGroups.map((group) => ({
    label: group.label,
    options: group.events.map((value) => {
      const choice = journalSourceEventChoices.find((entry) => entry.value === value)!;
      return blocked.has(value)
        ? {
            ...choice,
            disabled: true,
            description: REASONS.cashRevenue,
          }
        : choice;
    }),
  }));

  // Each basis allows one revenue and one expense policy, so choosing the basis chooses
  // them, and automatic posting takes the events the new policies need.
  const applyBasis = () => {
    const next = RECOGNITION_BY_BASIS[getValues("accountingBasis")];
    if (!next) return;
    const from = {
      revenue: getValues("revenueRecognitionPolicy"),
      expense: getValues("expenseRecognitionPolicy"),
    };
    setValue("revenueRecognitionPolicy", next.revenue, CHANGED);
    setValue("expenseRecognitionPolicy", next.expense, CHANGED);
    if (getValues("journalPostingMode") === journalPostingModeSchema.enum.Automatic) {
      setValue(
        "autoPostSourceEvents",
        sourceEventsForRecognition(getValues("autoPostSourceEvents") ?? [], from, next),
        CHANGED,
      );
    }
  };

  // Turning automatic posting on with nothing chosen starts from what the policies need.
  const seedSourceEvents = () => {
    if (getValues("journalPostingMode") !== journalPostingModeSchema.enum.Automatic) return;
    if ((getValues("autoPostSourceEvents") ?? []).length > 0) return;
    setValue(
      "autoPostSourceEvents",
      requiredSourceEvents({
        revenue: getValues("revenueRecognitionPolicy"),
        expense: getValues("expenseRecognitionPolicy"),
      }),
      CHANGED,
    );
  };

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Recognition")}
        description={t(
          "When revenue and expense reach the ledger. Each accounting basis allows one policy for each, so choosing the basis sets both.",
        )}
      >
        <SelectField
          control={control}
          name="accountingBasis"
          label={t("Accounting basis")}
          description={t("Accrual books on invoices and bills; cash books on money in and out.")}
          options={accountingBasisChoices}
          onValueChange={applyBasis}
          required
        />
        <SelectField
          control={control}
          name="revenueRecognitionPolicy"
          label={t("Revenue recognition")}
          description={t("The event that recognizes revenue.")}
          options={revenueOptions}
          required
        />
        <SelectField
          control={control}
          name="expenseRecognitionPolicy"
          label={t("Expense recognition")}
          description={t("The event that recognizes expense.")}
          options={expenseOptions}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Journals")}
        description={t(
          "How journals are created, who may write them by hand and how a posted journal is undone.",
        )}
      >
        <SelectField
          control={control}
          name="journalPostingMode"
          label={t("Journal posting")}
          description={t(
            "Manual journals are created only when someone posts them; automatic ones are created from the events you choose.",
          )}
          options={journalPostingModeChoices}
          onValueChange={seedSourceEvents}
          required
        />
        <SettingsNested show={automatic}>
          <MultiCheckboxField
            control={control}
            name="autoPostSourceEvents"
            label={t("Events that post journals")}
            description={t(
              "Each event chosen creates its journal entry as it happens. The events your recognition policies depend on are required.",
            )}
            groups={eventGroups}
            emptyAs="array"
            required={automatic}
          />
        </SettingsNested>
        <SelectField
          control={control}
          name="manualJournalEntryPolicy"
          label={t("Manual journal entries")}
          description={t("Whether people may write journals by hand, and for what.")}
          options={manualJournalEntryPolicyChoices}
          required
        />
        <SettingsNested show={manualPolicy !== manualJournalEntryPolicySchema.enum.Disallow}>
          <SwitchField
            control={control}
            name="requireManualJeApproval"
            label={t("Require approval for manual journals")}
            description={t("A manual journal waits for approval before it posts.")}
          />
        </SettingsNested>
        <SelectField
          control={control}
          name="journalReversalPolicy"
          label={t("Journal reversals")}
          description={t(
            "Whether a posted journal can be reversed and, if it can, the period the reversal lands in.",
          )}
          options={journalReversalPolicyChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
