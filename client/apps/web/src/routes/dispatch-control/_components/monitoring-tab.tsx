import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { serviceIncidentTypeChoices } from "@/lib/choices";
import { serviceIncidentTypeSchema, type DispatchControl } from "@/types/dispatch-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** What is watched while freight moves: service failures, quiet carriers and late arrivals. */
export function MonitoringTab() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const recordServiceFailures = useWatch({ control, name: "recordServiceFailures" });
  const enableCheckCallMonitoring = useWatch({ control, name: "enableCheckCallMonitoring" });
  const showFailureFields = recordServiceFailures !== serviceIncidentTypeSchema.enum.Never;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Service failure monitoring")}
        description={t(
          "Define which service failures to track and when they should be recorded. These settings drive operational reporting and exception visibility.",
        )}
      >
        <SelectField
          control={control}
          name="recordServiceFailures"
          label={t("Record Service failures")}
          description={t("Choose which incident types should be captured as service failures.")}
          options={serviceIncidentTypeChoices}
        />
        <SettingsNested show={showFailureFields}>
          <NumberField
            control={control}
            name="serviceFailureGracePeriod"
            label={t("Service failure grace period")}
            placeholder={t("Enter grace period in minutes")}
            description={t(
              "Defines the delay buffer before an eligible incident is recorded as a failure.",
            )}
            sideText="minutes"
            min={1}
          />
          <NumberField
            control={control}
            name="serviceFailureTarget"
            label={t("Service failure target")}
            placeholder={t("Enter target percentage")}
            description={t("Optional threshold for acceptable service failure rate.")}
            sideText="%"
            min={0}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Check call monitoring")}
        description={t(
          "Watch brokered loads in transit for carriers that stop reporting. A load with no check call, tracking update or stop time inside the window is raised to the watchtower.",
        )}
      >
        <SwitchField
          control={control}
          name="enableCheckCallMonitoring"
          label={t("Monitor brokered loads for check calls")}
          description={t(
            "When enabled, a carrier-covered load that goes quiet raises a No check call item, and one silent for twice the window is critical.",
          )}
        />
        <SettingsNested show={enableCheckCallMonitoring}>
          <NumberField
            control={control}
            name="checkCallIntervalHours"
            label={t("Check call window")}
            description={t(
              "How long a load may go without word from the carrier, from 1 to 72 hours. A customer can set its own window.",
            )}
            placeholder="4"
            sideText="hours"
            min={1}
            max={72}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Arrival projections")}
        description={t(
          "Every open stop on an active shipment carries a projected arrival, worked out from the truck's position, the carrier's check calls or the last departure, with routed drive time and the driver's hours.",
        )}
      >
        <SwitchField
          control={control}
          name="enableEtaAlerts"
          label={t("Raise at-risk loads in the watchtower")}
          description={t(
            "A load projected to arrive with less slack than the threshold raises an ETA at risk item; one projected past its window is critical.",
          )}
        />
        <NumberField
          control={control}
          name="etaRiskThresholdMinutes"
          label={t("At-risk threshold")}
          description={t(
            "Slack under this many minutes counts as at risk, from 0 to 1440. A customer can set its own threshold.",
          )}
          placeholder="60"
          sideText="minutes"
          min={0}
          max={1440}
        />
        <NumberField
          control={control}
          name="etaDwellMinutes"
          label={t("Stop dwell time")}
          description={t(
            "How long a stop takes once the truck is there, used to chain later stops, from 0 to 720 minutes.",
          )}
          placeholder="60"
          sideText="minutes"
          min={0}
          max={720}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
