import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsReveal } from "@/components/settings/settings-reveal";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  billingExceptionDispositionChoices,
  enforcementLevelChoices,
  rateVarianceAutoResolutionModeChoices,
} from "@/lib/choices";
import type { BillingControl } from "@/types/billing-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** What a shipment that fails a billing check does to its way through billing. */
export function ExceptionsTab() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const requirementLevel = useWatch({ control, name: "shipmentBillingRequirementEnforcement" });
  const rateLevel = useWatch({ control, name: "rateValidationEnforcement" });
  const resolution = useWatch({ control, name: "rateVarianceAutoResolutionMode" });

  const ratesReviewed = rateLevel === "RequireReview";
  const skipsWithinTolerance = ratesReviewed && resolution === "BypassReviewWithinTolerance";
  // A customer that validates its rates raises rate validation to review whatever is set
  // here, short of Block, so where review exceptions go matters unless both checks block.
  const reviewsHappen = requirementLevel === "RequireReview" || rateLevel !== "Block";

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Checks")}
        description={t(
          "What a shipment that fails a billing check does: nothing, a warning, a review or a stop.",
        )}
      >
        <SelectField
          control={control}
          name="shipmentBillingRequirementEnforcement"
          label={t("Billing requirements")}
          description={t("When a shipment is missing the paperwork or details billing needs.")}
          options={enforcementLevelChoices}
          required
        />
        <SelectField
          control={control}
          name="rateValidationEnforcement"
          label={t("Rate validation")}
          description={t("When a shipment's charges differ from the rate it was calculated at.")}
          options={enforcementLevelChoices}
          required
        />
        <SettingsNested show={ratesReviewed}>
          <SelectField
            control={control}
            name="rateVarianceAutoResolutionMode"
            label={t("Skip review for small differences")}
            description={t(
              "Whether a difference within the tolerance goes through without a review.",
            )}
            options={rateVarianceAutoResolutionModeChoices}
            required={ratesReviewed}
          />
        </SettingsNested>
        <SettingsNested show={skipsWithinTolerance}>
          <NumberField
            control={control}
            name="rateVarianceTolerancePercent"
            label={t("Tolerance")}
            description={t("The largest difference, as a share of the rate, that skips review.")}
            sideText="%"
            decimalScale={2}
            min={0}
            required={skipsWithinTolerance}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Review")}
        description={t("Where a shipment waiting on review goes, and who hears about it.")}
      >
        <SettingsReveal show={reviewsHappen}>
          <SelectField
            control={control}
            name="billingExceptionDisposition"
            label={t("Send for review to")}
            description={t(
              "Billing reviews it in the queue, or operations gets it back to fix before it can be billed.",
            )}
            options={billingExceptionDispositionChoices}
            required
          />
        </SettingsReveal>
        <SwitchField
          control={control}
          name="notifyOnBillingExceptions"
          label={t("Notify on exceptions")}
          description={t("Sends a notification each time a billing exception is recorded.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
