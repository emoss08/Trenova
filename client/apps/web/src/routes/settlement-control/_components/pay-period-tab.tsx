import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import {
  payPeriodFrequencyChoices,
  settlementPayTriggerChoices,
  weekdayChoices,
} from "@/lib/choices";
import type { SettlementControl } from "@/types/settlement-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** When each pay period ends, when it is paid, and when a move's pay counts toward it. */
export function PayPeriodTab() {
  const t = useT();
  const { control } = useFormContext<SettlementControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Schedule")}
        description={t("When each pay period ends and when drivers are paid for it.")}
      >
        <SelectField
          control={control}
          name="payPeriodFrequency"
          label={t("Pay period")}
          description={t("How long each pay period runs. Weekly is the norm for asset carriers.")}
          options={payPeriodFrequencyChoices}
          required
        />
        <SelectField
          control={control}
          name="periodEndDayOfWeek"
          label={t("Last day of the period")}
          description={t("Work on this day still counts toward the period that ends with it.")}
          options={weekdayChoices}
          required
        />
        <NumberField
          control={control}
          name="payDelayDays"
          label={t("Pay delay")}
          description={t("How many days after a period ends its settlements are paid.")}
          placeholder="5"
          sideText={t("days")}
          min={0}
          max={30}
          required
        />
        <SwitchField
          control={control}
          name="autoGenerateBatches"
          label={t("Generate settlements when a period ends")}
          description={t(
            "A settlement batch for the period is drafted without anyone starting it.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Earning pay")}
        description={t("When the pay for a move counts toward a driver's settlement.")}
      >
        <SelectField
          control={control}
          name="payTrigger"
          label={t("Pay is earned when")}
          description={t(
            "Move completed pays each driver as soon as their own move finishes, the most accurate choice when drivers split a load.",
          )}
          options={settlementPayTriggerChoices}
          required
        />
        <SwitchField
          control={control}
          name="autoAttachAccruals"
          label={t("Add new pay to open drafts")}
          description={t(
            "Pay a driver earns after their draft settlement was made joins it, without anyone moving it across.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
