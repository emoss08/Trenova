import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { DispatchControl } from "@/types/dispatch-control";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * Whether the planner assigns on its own, and how far it may go: what it does without
 * being asked, then the limits it plans inside. What it learns is on the Learning tab.
 */
export function AssignmentTab() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const enableAutoAssignment = useWatch({ control, name: "enableAutoAssignment" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Automated resource assignment")}
        description={t(
          "Configure how the system chooses workers and equipment for shipments. These controls influence assignment consistency, utilization, and dispatch throughput.",
        )}
      >
        <FormControl>
          <SwitchField
            control={control}
            name="enableAutoAssignment"
            label={t("Enable automated assignment")}
            description={t(
              "When enabled, the system can automatically assign available resources to shipments.",
            )}
            position="left"
          />
        </FormControl>
        {enableAutoAssignment ? (
          <SettingsNested>
            <FormControl>
              <SwitchField
                control={control}
                name="autoProposeCoverage"
                label={t("Keep coverage proposed")}
                description={t(
                  "Plans again when loads are created or left without a driver, and every hour, so each uncovered move has a proposed driver waiting in the dispatch console and in Decisions. Nothing is assigned until someone approves it, unless the dispatch agent is trusted to act on its own.",
                )}
                position="left"
              />
            </FormControl>
            <FormGroup cols={2}>
              <FormControl>
                <NumberField
                  control={control}
                  name="autoAssignConfidenceThreshold"
                  label={t("Auto-execute confidence threshold")}
                  description={t(
                    "Minimum confidence (0 to 1) an automatic assignment must reach before it executes without dispatcher review.",
                  )}
                  placeholder="0.85"
                  decimalScale={2}
                />
              </FormControl>
            </FormGroup>
          </SettingsNested>
        ) : null}
      </SettingsSection>
      {enableAutoAssignment ? (
        <SettingsSection
          title={t("Planning limits")}
          description={t(
            "How far away and how far ahead the planner looks, and when a move with nobody on it becomes urgent.",
          )}
        >
          <FormGroup cols={2}>
            <FormControl>
              <NumberField
                control={control}
                name="autoAssignMaxDeadheadMiles"
                label={t("Maximum deadhead miles")}
                description={t(
                  "Candidates beyond this many empty miles from the pickup are never auto-assigned. Leave empty for no cap.",
                )}
                placeholder={t("No limit")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="autoAssignPlanningHorizonHours"
                label={t("Planning horizon (hours)")}
                description={t("How far ahead the optimizer plans moves, from 1 to 336 hours.")}
                placeholder="48"
                sideText="hours"
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="coverageRiskWindowHours"
                label={t("Coverage risk window (hours)")}
                description={t(
                  "A move this close to its start with nobody on it is raised to the watchtower and handed to the dispatch agent. Shorter than the planning horizon on purpose.",
                )}
                placeholder="12"
                sideText="hours"
              />
            </FormControl>
          </FormGroup>
        </SettingsSection>
      ) : null}
    </SettingsSections>
  );
}
