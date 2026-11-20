import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { ShipmentControl } from "@/types/shipment-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** The status changes the system makes on its own: late shipments delayed, stale ones cancelled. */
export function AutomationTab() {
  const t = useT();
  const { control } = useFormContext<ShipmentControl>();
  const autoDelay = useWatch({ control, name: "autoDelayShipments" });
  const autoCancel = useWatch({ control, name: "autoCancelShipments" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Delays")}
        description={t("Marks a shipment Delayed once it runs late, so nobody has to.")}
      >
        <SwitchField
          control={control}
          name="autoDelayShipments"
          label={t("Mark late shipments delayed")}
          description={t("Sets a shipment to Delayed when it falls behind its schedule.")}
        />
        <SettingsNested show={autoDelay}>
          <NumberField
            control={control}
            name="autoDelayShipmentsThreshold"
            label={t("Late after")}
            description={t("How far past its scheduled time a shipment is before it is delayed.")}
            placeholder="30"
            sideText={t("minutes")}
            min={1}
            max={32767}
            required={autoDelay}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Cancellations")}
        description={t("Cancels shipments that were entered and never moved past New.")}
      >
        <SwitchField
          control={control}
          name="autoCancelShipments"
          label={t("Cancel stale new shipments")}
          description={t("Cancels a shipment left in New for longer than the limit below.")}
        />
        <SettingsNested show={autoCancel}>
          <NumberField
            control={control}
            name="autoCancelShipmentsThreshold"
            label={t("Cancel after")}
            description={t(
              "How many days after it was created a shipment still in New is cancelled.",
            )}
            placeholder="30"
            sideText={t("days")}
            min={1}
            max={32767}
            required={autoCancel}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
