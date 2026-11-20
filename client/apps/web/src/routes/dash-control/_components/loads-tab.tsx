import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  DETENTION_ALERT_MAX_MINUTES,
  DETENTION_ALERT_MIN_MINUTES,
  type DashControl,
} from "@/types/dash-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** How drivers take on a load, what they record on it, and when a long stop alerts dispatch. */
export function LoadsTab() {
  const t = useT();
  const { control } = useFormContext<DashControl>();
  const accepts = useWatch({ control, name: "requireLoadAcknowledgment" });
  const detentionAlerts = useWatch({ control, name: "enableDetentionAlerts" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Assignments")}
        description={t("How a driver takes on a load dispatch has given them.")}
      >
        <SwitchField
          control={control}
          name="requireLoadAcknowledgment"
          label={t("Accept assignments")}
          description={t("Drivers confirm each new load, so dispatch knows it was received.")}
        />
        <SettingsNested show={accepts}>
          <SwitchField
            control={control}
            name="allowLoadRefusals"
            label={t("Allow declines")}
            description={t("A driver can turn a load down with a reason instead of accepting it.")}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("On the road")}
        description={t("What drivers record on a load while they run it.")}
      >
        <SwitchField
          control={control}
          name="allowStopActions"
          label={t("Arrivals and departures")}
          description={t(
            "Drivers record their own times at each stop, which moves the load along and times detention.",
          )}
        />
        <SwitchField
          control={control}
          name="allowLoadDocumentUpload"
          label={t("Load paperwork")}
          description={t(
            "Drivers photograph signed delivery receipts and bills of lading from the cab.",
          )}
        />
        <SwitchField
          control={control}
          name="allowLoadComments"
          label={t("Load messages")}
          description={t(
            "Drivers can write to dispatch on a load; they can always read its notes.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Detention")}
        description={t("Warn dispatch when a stop runs long enough to bill detention.")}
      >
        <SwitchField
          control={control}
          name="enableDetentionAlerts"
          label={t("Detention alerts")}
          description={t("Dispatch is alerted when a driver stays at a stop past the threshold.")}
        />
        <SettingsNested show={detentionAlerts}>
          <NumberField
            control={control}
            name="detentionAlertThresholdMinutes"
            label={t("Threshold")}
            description={t(
              "How long a driver can be at a stop before dispatch is alerted, from 15 minutes to a day.",
            )}
            placeholder="120"
            sideText={t("minutes")}
            min={DETENTION_ALERT_MIN_MINUTES}
            max={DETENTION_ALERT_MAX_MINUTES}
            required={detentionAlerts}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
