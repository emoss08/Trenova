import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { escrowInterestFrequencyChoices } from "@/lib/choices";
import { SETTLEMENT_PERCENT_SCALE, type SettlementControl } from "@/types/settlement-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** How pay advances are taken back, and the interest owner-operators earn on escrow. */
export function DeductionsTab() {
  const t = useT();
  const { control } = useFormContext<SettlementControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Advance recovery")}
        description={t("How a pay advance is taken back out of a driver's settlements.")}
      >
        <SwitchField
          control={control}
          name="allowNegativeNet"
          label={t("Recover advances in full")}
          description={t(
            "An advance larger than a settlement's net pay is taken in full and the shortfall carries to the next settlement. Off, recovery stops at zero net pay.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Escrow interest")}
        description={t(
          "49 CFR 376.12(k) requires interest on owner-operator escrow at least every quarter.",
        )}
      >
        <NumberField
          control={control}
          name="defaultEscrowInterestRate"
          valueType="string"
          label={t("Yearly interest rate")}
          description={t("The rate a new escrow account starts with. Each account can change it.")}
          placeholder="0"
          sideText="%"
          decimalScale={SETTLEMENT_PERCENT_SCALE}
          min={0}
          max={100}
          required
        />
        <SelectField
          control={control}
          name="escrowInterestFrequencyMonths"
          label={t("Pay interest")}
          description={t("How often interest is added to each escrow balance.")}
          options={escrowInterestFrequencyChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
