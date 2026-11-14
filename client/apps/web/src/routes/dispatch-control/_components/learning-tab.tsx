import { SwitchField } from "@/components/fields/switch-field";
import { Kept } from "@/components/kept";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import type { DispatchControl } from "@/types/dispatch-control";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { FreightForecast } from "./freight-forecast";
import { LearnedStyle } from "./learned-style";

/**
 * What the planner learns from the organization's own history, each switch beside what it
 * has learned so far: the dispatchers' style with its weights, and positioning with the
 * freight forecast it acts on.
 *
 * The switches belong to automated assignment and are offered only while it is on. What
 * was learned is read from the server the first time the tab is opened, not with the page.
 */
export function LearningTab({
  shown,
  onOpenAssignment,
}: {
  shown: boolean;
  onOpenAssignment: () => void;
}) {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const enableAutoAssignment = useWatch({ control, name: "enableAutoAssignment" });

  return (
    <SettingsSections>
      {enableAutoAssignment ? null : (
        <Alert size="sm">
          <AlertDescription>
            {t("Turn on automated assignment to choose what the planner learns.")}
          </AlertDescription>
          <AlertAction>
            <Button type="button" variant="outline" size="sm" onClick={onOpenAssignment}>
              {t("Open Assignment")}
            </Button>
          </AlertAction>
        </Alert>
      )}

      <SettingsSection
        title={t("Learned dispatch style")}
        description={t(
          "How your dispatchers weigh each preference against your costs, learned from the driver they assign each time they take, change or pass over the plan's choice. The planner uses these weights while learning is on.",
        )}
      >
        {enableAutoAssignment ? (
          <FormControl>
            <SwitchField
              control={control}
              name="learnDispatchStyle"
              label={t("Learn our dispatch style")}
              description={t(
                "Learns how much your dispatchers weigh each preference from the drivers they assign against the plan: the plan's driver, another it priced, or one it passed over. Shown and reset under Learned dispatch style.",
              )}
              position="left"
            />
          </FormControl>
        ) : null}
        <Kept shown={shown}>
          <Readout ruled={enableAutoAssignment}>
            <LearnedStyle />
          </Readout>
        </Kept>
      </SettingsSection>

      {enableAutoAssignment ? (
        <SettingsSection
          title={t("Positioning")}
          description={t(
            "Whether a plan looks past the move in front of it, to where it leaves the driver.",
          )}
        >
          <FormControl>
            <SwitchField
              control={control}
              name="planForEndMarket"
              label={t("Plan for where drivers end up")}
              description={t(
                "Credits a move that leaves its driver where drivers have been worth more to the plan at that time of the week, learned from the planner's own history.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="anticipateFreight"
              label={t("Anticipate freight")}
              description={t(
                "Position drivers for freight your history says is coming, before it is booked.",
              )}
              position="left"
            />
          </FormControl>
        </SettingsSection>
      ) : null}

      <SettingsSection
        title={t("Freight forecast")}
        description={t(
          "Where freight usually appears, learned from your shipment history and tested every night on the weeks it has not seen. Anticipate freight changes plans only while the forecast is Ready.",
        )}
      >
        <Kept shown={shown}>
          <Readout>
            <FreightForecast />
          </Readout>
        </Kept>
      </SettingsSection>
    </SettingsSections>
  );
}

/** What was learned, ruled off from the switch above it when there is one. */
function Readout({ ruled = false, children }: { ruled?: boolean; children: ReactNode }) {
  return (
    <div className={cn("flex flex-col gap-4", ruled && "border-t border-border pt-4")}>
      {children}
    </div>
  );
}
