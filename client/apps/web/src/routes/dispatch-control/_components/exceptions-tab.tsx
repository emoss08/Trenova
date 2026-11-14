import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import type { DispatchControl } from "@/types/dispatch-control";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/**
 * The watchtower's exception queue, in the order an exception lives: when it is due, what
 * happens when it runs over, who owns it, and when a late stop or missing paperwork raises one.
 */
export function ExceptionsTab() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Due times")}
        description={t(
          "Every exception on the watchtower has a due time, set by its severity when it is raised.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionDueCriticalMinutes"
              label={t("Critical exceptions are due in")}
              placeholder="60"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionDueWarningMinutes"
              label={t("Warnings are due in")}
              placeholder="240"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionDueInfoMinutes"
              label={t("Information is due in")}
              description={t(
                "From 5 minutes to a week, counted from when the exception is raised.",
              )}
              placeholder="1440"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
        </FormGroup>
      </SettingsSection>

      <SettingsSection
        title={t("Escalation")}
        description={t(
          "An overdue exception escalates to its owner and the escalation contact until it is dealt with or the limit is reached.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionEscalationMinutes"
              label={t("Escalate an overdue exception every")}
              description={t(
                "Each escalation raises its severity, tells its owner and the escalation contact, and gives it this long again.",
              )}
              placeholder="60"
              sideText="minutes"
              min={5}
              max={1440}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionMaxEscalations"
              label={t("Escalate at most")}
              description={t("0 turns escalation off; at most 10.")}
              placeholder="3"
              sideText="times"
              min={0}
              max={10}
            />
          </FormControl>
          <FormControl>
            <UserAutocompleteField
              control={control}
              name="exceptionEscalationContactId"
              label={t("Escalation contact")}
              description={t("Hears about every escalation, alongside the owner.")}
              placeholder={t("Nobody")}
            />
          </FormControl>
        </FormGroup>
      </SettingsSection>

      <SettingsSection
        title={t("Ownership")}
        description={t("Who an exception goes to when nothing else decides.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <UserAutocompleteField
              control={control}
              name="exceptionDefaultAssigneeId"
              label={t("Default owner")}
              description={t(
                "Owns an exception no rule assigns. A shipment's own exceptions go to its owner first.",
              )}
              placeholder={t("Nobody")}
            />
          </FormControl>
        </FormGroup>
      </SettingsSection>

      <SettingsSection
        title={t("Late stops and paperwork")}
        description={t(
          "How long past a stop's window or a delivery before the watchtower raises an exception.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionMissedStopGraceMinutes"
              label={t("A stop is missed after its window closes by")}
              description={t(
                "With no arrival recorded by then, a missed pickup or missed delivery is raised.",
              )}
              placeholder="30"
              sideText="minutes"
              min={0}
              max={1440}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="exceptionPodGraceHours"
              label={t("A POD is missing after delivery by")}
              description={t("With no proof of delivery on file by then, POD missing is raised.")}
              placeholder="24"
              sideText="hours"
              min={1}
              max={720}
            />
          </FormControl>
        </FormGroup>
      </SettingsSection>
    </SettingsSections>
  );
}
