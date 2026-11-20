import { FuelIndexAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import type { CostControlFormValues } from "@/types/cost-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** What every estimate assumes: how fuel is priced, and what a shipment's cost and margin count. */
export function BasisTab() {
  const t = useT();
  const { control } = useFormContext<CostControlFormValues>();
  const useLiveFuelPrice = useWatch({ control, name: "useLiveFuelPrice" });
  const marginsWithheld = !useProtectedFieldAccess("Margins");

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Fuel")}
        description={t("How the fuel cost of a mile is worked out.")}
      >
        <NumberField
          control={control}
          name="milesPerGallon"
          label={t("Fleet miles per gallon")}
          description={t(
            "The fleet's average fuel economy, which turns a gallon's price into a mile's.",
          )}
          sideText={t("mpg")}
          decimalScale={2}
          min={0}
          max={20}
          required
        />
        <SwitchField
          control={control}
          name="useLiveFuelPrice"
          label={t("Use live fuel price")}
          description={t("Prices fuel from a diesel index instead of the industry benchmark.")}
        />
        <SettingsNested show={useLiveFuelPrice}>
          <FuelIndexAutocompleteField
            control={control}
            name="fuelIndexId"
            label={t("Fuel index")}
            placeholder={t("Select fuel index")}
            description={t("The diesel price index the fuel cost is read from.")}
            required={useLiveFuelPrice}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Profitability")}
        description={t(
          "What a shipment's estimated cost counts, and the margin it is measured against.",
        )}
      >
        <SwitchField
          control={control}
          name="includeDeadheadMiles"
          label={t("Include deadhead miles")}
          description={t("Charges the empty miles to reach a pickup to the shipment's cost.")}
        />
        <NumberField
          control={control}
          name="targetMarginPercent"
          label={t("Target margin")}
          description={
            marginsWithheld
              ? t("Your role does not include margins.")
              : t("Shipments below this margin show as thin. Empty means 10%.")
          }
          placeholder={marginsWithheld ? t("Hidden") : "10"}
          sideText="%"
          decimalScale={2}
          min={0}
          max={100}
          disabled={marginsWithheld}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
