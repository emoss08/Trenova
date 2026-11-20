import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsReveal } from "@/components/settings/settings-reveal";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { lateChargeAssessmentModeChoices } from "@/lib/choices";
import { MAX_STATEMENT_DAY, type BillingControl } from "@/types/billing-control";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { PaymentReminderDaysField } from "./payment-reminder-days-field";

/**
 * Getting paid on overdue invoices: late charges, reminders and monthly statements. A
 * customer can be left out of any of them from its billing profile.
 */
export function CollectionsTab({ onOpenAutomation }: { onOpenAutomation: () => void }) {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const lateChargeMode = useWatch({ control, name: "lateChargeAssessmentMode" });
  const postingMode = useWatch({ control, name: "invoicePostingMode" });
  const remindersOn = useWatch({ control, name: "paymentRemindersEnabled" });
  const statementsOn = useWatch({ control, name: "monthlyStatementsEnabled" });

  // Raised late charges post the way invoices do, so with invoices waiting on review the
  // nightly run's memos wait too.
  const memosWait = lateChargeMode === "Automatic" && postingMode === "ManualReviewRequired";

  return (
    <SettingsSections>
      <SettingsReveal show={memosWait}>
        <div className="pb-5">
          <Alert size="sm">
            <AlertDescription>
              {t(
                "Late charges raised overnight wait for someone to post them, because invoices post only after review.",
              )}
            </AlertDescription>
            <AlertAction>
              <Button type="button" variant="secondary" size="sm" onClick={onOpenAutomation}>
                {t("Open Automation")}
              </Button>
            </AlertAction>
          </Alert>
        </div>
      </SettingsReveal>

      <SettingsSection
        title={t("Late charges")}
        description={t(
          "Overdue invoices are charged once per thirty-day period at the customer's late charge rate, after its grace period, as one debit memo per customer per run.",
        )}
      >
        <SelectField
          control={control}
          name="lateChargeAssessmentMode"
          label={t("Assess late charges")}
          description={
            lateChargeMode === "Preview"
              ? t(
                  "The nightly run works out what it would raise and writes nothing; raise memos by hand from the Late Charges page.",
                )
              : lateChargeMode === "Automatic"
                ? t("The nightly run raises the debit memos.")
                : t("No late charges are assessed; the Late Charges page can still preview them.")
          }
          options={lateChargeAssessmentModeChoices}
          required
        />
        <NumberField
          control={control}
          name="lateChargeMinimumAmount"
          label={t("Minimum late charge")}
          description={t("A customer whose late charges for a run come to less is skipped.")}
          decimalScale={2}
          fixedDecimalScale
          min={0}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Payment reminders")}
        description={t(
          "One email lists every overdue invoice a customer has; disputed invoices and customers waiting on a promise to pay are left out.",
        )}
      >
        <SwitchField
          control={control}
          name="paymentRemindersEnabled"
          label={t("Send payment reminders")}
          description={t(
            "Sent each morning to the invoice recipients on the customer's email profile, from your Billing email profile.",
          )}
        />
        <SettingsNested show={remindersOn}>
          <PaymentReminderDaysField />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Monthly statements")}
        description={t(
          "A PDF of the month just closed: the opening balance, every invoice, payment and credit, what is owed and the open invoices, aged.",
        )}
      >
        <SwitchField
          control={control}
          name="monthlyStatementsEnabled"
          label={t("Send monthly statements")}
          description={t(
            "Sent once a month to every customer with a balance or activity, from your Billing email profile.",
          )}
        />
        <SettingsNested show={statementsOn}>
          <NumberField
            control={control}
            name="monthlyStatementDay"
            label={t("Day of the month to send")}
            description={t(
              "From 1 to 28, so the day falls in every month; statements are dated the last day of the month before.",
            )}
            min={1}
            max={MAX_STATEMENT_DAY}
            required={t("Choose the day of the month to send statements on")}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
