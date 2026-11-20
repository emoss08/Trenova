import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { driverDigestCadenceChoices, weekdayChoices } from "@/lib/choices";
import { driverDigestCadenceSchema, type DashControl } from "@/types/dash-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** Whether drivers are reminded of what falls due, and how often the reminders come. */
export function RemindersTab() {
  const t = useT();
  const { control } = useFormContext<DashControl>();
  const reminds = useWatch({ control, name: "sendCredentialReminders" });
  const cadence = useWatch({ control, name: "driverDigestCadence" });
  const weekly = cadence === driverDigestCadenceSchema.enum.Weekly;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Due-date reminders")}
        description={t(
          "What drivers are told before a credential expires or a training course falls due.",
        )}
      >
        <SwitchField
          control={control}
          name="sendCredentialReminders"
          label={t("Remind drivers")}
          description={t(
            "Drivers are reminded on their phone; the office still hears about every expired credential.",
          )}
        />
        <SettingsNested show={reminds}>
          <SelectField
            control={control}
            name="driverDigestCadence"
            label={t("How often")}
            description={t("A reminder for each item, or everything a driver owes in one notice.")}
            options={driverDigestCadenceChoices}
            required
          />
        </SettingsNested>
        <SettingsNested show={reminds && weekly}>
          <SelectField
            control={control}
            name="driverDigestWeekday"
            label={t("Round-up day")}
            description={t("The day the weekly round-up goes out.")}
            options={weekdayChoices}
            required
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
