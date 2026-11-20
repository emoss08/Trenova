import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsReveal } from "@/components/settings/settings-reveal";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { CostControlFormValues } from "@/types/cost-control";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * Reading rates from the ledger: whether categories may, over how many months, and the
 * floor fixed costs divide by. Which categories do is chosen on the Cost rates tab, which
 * this tab opens.
 */
export function ActualsTab({ onOpenRates }: { onOpenRates: () => void }) {
  const t = useT();
  const { control } = useFormContext<CostControlFormValues>();
  const glActualsEnabled = useWatch({ control, name: "glActualsEnabled" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("GL actuals")}
        description={t(
          "Replaces a category's benchmark with what you actually spent, from expenses posted to its GL accounts.",
        )}
      >
        <SwitchField
          control={control}
          name="glActualsEnabled"
          label={t("Use GL actuals")}
          description={t("Lets cost categories read their rate from posted expenses.")}
        />
        <SettingsNested show={glActualsEnabled}>
          <NumberField
            control={control}
            name="glRollingMonths"
            label={t("Rolling window")}
            description={t("How many past months of expenses a rate is worked out from.")}
            sideText={t("months")}
            decimalScale={0}
            min={1}
            max={12}
            required
          />
          <NumberField
            control={control}
            name="plannedMonthlyMiles"
            label={t("Planned monthly miles")}
            description={t(
              "The fewest miles fixed costs are spread over, so a slow month does not inflate them.",
            )}
            sideText={t("miles")}
            decimalScale={0}
            min={1}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsReveal show={glActualsEnabled}>
        <div className="pb-5">
          <Alert size="sm">
            <AlertDescription>
              {t("Choose which categories use GL actuals, and their accounts, on Cost rates.")}
            </AlertDescription>
            <AlertAction>
              <Button type="button" variant="secondary" size="sm" onClick={onOpenRates}>
                {t("Open Cost rates")}
              </Button>
            </AlertAction>
          </Alert>
        </div>
      </SettingsReveal>
    </SettingsSections>
  );
}
