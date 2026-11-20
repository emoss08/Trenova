import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import type { ShipmentControl } from "@/types/shipment-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** What is checked when a shipment is entered, and what its record keeps. */
export function EntryTab() {
  const t = useT();
  const { control } = useFormContext<ShipmentControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Entry checks")}
        description={t("What a shipment is checked against when it is created or edited.")}
      >
        <NumberField
          control={control}
          name="maxShipmentWeightLimit"
          label={t("Weight limit")}
          description={t(
            "The heaviest shipment that can be entered, in pounds, unless its mode profile sets its own.",
          )}
          placeholder="80000"
          sideText={t("lbs")}
          min={0}
          required
        />
        <SwitchField
          control={control}
          name="checkForDuplicateBols"
          label={t("Check for duplicate BOLs")}
          description={t("Refuses a bill of lading number already on another shipment.")}
        />
        <SwitchField
          control={control}
          name="checkHazmatSegregation"
          label={t("Check hazmat segregation")}
          description={t(
            "Refuses hazardous materials that must not travel together on one shipment.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Shipment history")}
        description={t("What stays on a shipment's record when its plan changes.")}
      >
        <SwitchField
          control={control}
          name="allowMoveRemovals"
          label={t("Allow removing moves")}
          description={t(
            "Lets a move be deleted from a shipment instead of cancelled, leaving no trace of it.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
