import { SwitchField } from "@/components/fields/switch-field";
import { SectionPanel } from "@/components/section-panel";
import { formatPlanMeterValue, planMeterLabel } from "@/lib/plan-meters";
import type { OnboardingFormValues } from "@/types/onboarding";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { InfoCircleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useWatch, type Control } from "react-hook-form";

/**
 * What "Load sample data" creates, as the server's onboarding service does. Each row
 * names the meter it counts against, so the person can see how much of the demo it
 * spends.
 */
export const SAMPLE_DATA_SET: readonly { meter: string; quantity: number }[] = [
  { meter: "customers.total", quantity: 2 },
  { meter: "locations.total", quantity: 4 },
  { meter: "workers.total", quantity: 1 },
  { meter: "tractors.total", quantity: 1 },
  { meter: "trailers.total", quantity: 1 },
  { meter: "shipments.total", quantity: 2 },
];

export function SampleDataStep({
  control,
  freePlanLimits,
}: {
  control: Control<OnboardingFormValues>;
  freePlanLimits: Readonly<Record<string, number>>;
}) {
  const t = useT();
  const loadSampleData = useWatch({ control, name: "loadSampleData" });

  return (
    <SectionPanel title={t("Sample data")}>
      <div className="flex flex-col gap-4 p-4">
        <SwitchField
          control={control}
          name="loadSampleData"
          label={t("Load sample data")}
          description={t(
            "Start with a few customers, locations, equipment and shipments so every screen has something to show. You can delete them at any time.",
          )}
          outlined
        />

        <DescriptionList layout="split" aria-label={t("Sample data set")}>
          {SAMPLE_DATA_SET.map((row) => {
            const limit = freePlanLimits[row.meter];
            return (
              <DescriptionItem key={row.meter} label={t(planMeterLabel(row.meter))} numeric>
                {limit
                  ? t(
                      "{0} of {1}",
                      formatPlanMeterValue(row.meter, row.quantity),
                      formatPlanMeterValue(row.meter, limit),
                    )
                  : formatPlanMeterValue(row.meter, row.quantity)}
              </DescriptionItem>
            );
          })}
        </DescriptionList>

        {loadSampleData ? (
          <Alert variant="info" size="sm">
            <InfoCircleIcon />
            <AlertDescription>
              {t(
                "Sample records count toward the free demo's limits, the same as records you create. Deleting one frees its slot.",
              )}
            </AlertDescription>
          </Alert>
        ) : null}
      </div>
    </SectionPanel>
  );
}
