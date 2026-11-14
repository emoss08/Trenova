import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { DISPATCH_PREFERENCE_META, type DispatchControl } from "@/types/dispatch-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/**
 * What the dispatch optimizer plans with, in money: what each pairing costs and what leaving
 * a move uncovered costs. The planner picks the plan that costs least overall, so these are
 * the organization's real trade-offs, not relative weights. Each list reads as a price list:
 * what is priced on the left, its price down the right edge.
 */
export function CostsTab() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Costs and prices")}
        description={t(
          "The planner prices every driver for every move and picks the plan that costs least overall. Set what each thing costs your operation; a cost of $0 is ignored.",
        )}
      >
        <MoneyField
          control={control}
          name="economics.emptyMileCents"
          label={t("Empty mile")}
          description={t("What a mile driven empty to the pickup costs.")}
        />
        <MoneyField
          control={control}
          name="economics.lateMinuteCents"
          label={t("Late minute")}
          description={t("What each minute past a pickup or delivery window costs.")}
        />
        <MoneyField
          control={control}
          name="economics.driverHourCents"
          label={t("Driver hour")}
          description={t(
            "What an hour of a driver waiting, or resting before they may drive, costs.",
          )}
        />
        <NumberField
          control={control}
          name="economics.earlyGraceMinutes"
          label={t("Early grace (minutes)")}
          description={t("Arriving this early is free; waiting longer is a driver hour.")}
        />
        <NumberField
          control={control}
          name="economics.homeTimeTargetDays"
          label={t("Home time target (days)")}
          description={t("Days out before home time starts to cost.")}
        />
        <MoneyField
          control={control}
          name="economics.homeTimeDayCents"
          label={t("Day past home time")}
          description={t("What each day past the target costs, less for a move that heads home.")}
        />
        <MoneyField
          control={control}
          name="economics.fairnessMileCents"
          label={t("Mile above the median")}
          description={t(
            "What each mile a driver has run this week above the fleet's median costs, so work spreads.",
          )}
        />
        <NumberField
          control={control}
          name="economics.candidatesPerMove"
          label={t("Drivers priced per move")}
          description={t(
            "How many of the nearest eligible drivers the planner prices for each move.",
          )}
        />
        <MoneyField
          control={control}
          name="economics.replanChangeCents"
          label={t("Changing a planned driver")}
          description={t(
            "What moving a move to another driver than the last plan chose costs, so proposals don't change for a few dollars.",
          )}
        />
        <MoneyField
          control={control}
          name="economics.anticipationMaxCents"
          label={t("Most a likely load is worth")}
          placeholder={t("Automatic")}
          description={t(
            "Caps what ending near likely freight can be worth on one move. Leave empty to use what a typical load is worth to you, from its revenue and your share of revenue.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Preferences")}
        description={t(
          "What the worst case of each costs against the best. A driver with no history counts as halfway.",
        )}
      >
        {DISPATCH_PREFERENCE_META.map((preference) => (
          <MoneyField
            key={preference.key}
            control={control}
            name={`economics.preferences.${preference.key}`}
            label={t(preference.label)}
            description={t(preference.description)}
          />
        ))}
      </SettingsSection>

      <SettingsSection
        title={t("Leaving a move uncovered")}
        description={t(
          "A move goes uncovered only when every driver would cost more than this. When drivers are short, the moves that cost most to leave are covered first.",
        )}
      >
        <NumberField
          control={control}
          name="economics.coverageMarginPercent"
          label={t("Share of revenue (%)")}
          description={t("The part of a move's revenue lost when it is not covered.")}
        />
        <MoneyField
          control={control}
          name="economics.coverageFloorCents"
          label={t("At least")}
          description={t("The least leaving any move uncovered costs.")}
        />
        <MoneyField
          control={control}
          name="economics.latePremiumCents"
          label={t("Late premium")}
          description={t("Added for a move whose pickup window has closed.")}
        />
        <MoneyField
          control={control}
          name="economics.dueNowPremiumCents"
          label={t("Due now premium")}
          description={t("Added for a move whose window is open or opens within 4 hours.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
