import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { carrierPayTriggerChoices, payPeriodFrequencyChoices, weekdayChoices } from "@/lib/choices";
import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** When each carrier pay period ends, when it is paid, and when a load's cost counts toward it. */
export function PayPeriodTab() {
  const t = useT();
  const { control } = useFormContext<CarrierSettlementControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Schedule")}
        description={t("When each pay period ends and when carriers are paid for it.")}
      >
        <SelectField
          control={control}
          name="payPeriodFrequency"
          label={t("Pay period")}
          description={t("How long each pay period runs. Weekly is the norm for brokered freight.")}
          options={payPeriodFrequencyChoices}
          required
        />
        <SelectField
          control={control}
          name="periodEndDayOfWeek"
          label={t("Last day of the period")}
          description={t("Cost on this day still counts toward the period that ends with it.")}
          options={weekdayChoices}
          required
        />
        <NumberField
          control={control}
          name="payDelayDays"
          label={t("Pay delay")}
          description={t("How many days after a period ends its carrier settlements are paid.")}
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
            "A carrier settlement batch for the period is drafted without anyone starting it.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Carrier cost")}
        description={t("When a carrier's cost on a load counts toward its settlement.")}
      >
        <SelectField
          control={control}
          name="payTrigger"
          label={t("Cost counts when")}
          description={t(
            "Move completed counts each carrier's leg as soon as it finishes, the most accurate choice when carriers split a load.",
          )}
          options={carrierPayTriggerChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
