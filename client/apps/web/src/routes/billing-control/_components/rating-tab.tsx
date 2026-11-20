import { FormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { unratedShipmentDispositionChoices } from "@/lib/choices";
import type { BillingControl } from "@/types/billing-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { useBillingControlPlan } from "../billing-control-tabs";

/**
 * What happens to a shipment no rate covers, and how far a rate may depart from its
 * contract. Formula templates and rate agreements come with rating, so the rows that name
 * them are offered only on a plan that includes it.
 */
export function RatingTab() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const disposition = useWatch({ control, name: "unratedShipmentDisposition" });
  const { rating } = useBillingControlPlan();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Unrated shipments")}
        description={t("What happens to a shipment that no rate agreement covers.")}
      >
        <SelectField
          control={control}
          name="unratedShipmentDisposition"
          label={t("When no rate applies")}
          description={t(
            "Price it from a formula template, save it at zero and flag it for review, or refuse the save.",
          )}
          options={unratedShipmentDispositionChoices}
          required
        />
        <SettingsNested show={rating && disposition === "FallbackFormulaTemplate"}>
          <FormulaTemplateAutocompleteField
            control={control}
            name="fallbackFormulaTemplateId"
            label={t("Fallback formula template")}
            description={t(
              "Used when the shipment has no formula template of its own. Leave empty to require one on the shipment.",
            )}
            placeholder={t("Select formula template")}
            clearable
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Rate changes")}
        description={t("How far a rate may move from what its contract says.")}
      >
        <SwitchField
          control={control}
          name="requireRateOverrideReason"
          label={t("Require a reason for rate overrides")}
          description={t("Someone who changes a rate by hand must say why, for the audit trail.")}
        />
        {rating ? (
          <SwitchField
            control={control}
            name="enforceMarginFloor"
            label={t("Enforce margin floors")}
            description={t(
              "Refuses a rate below its agreement's margin floor instead of only flagging it.",
            )}
          />
        ) : null}
      </SettingsSection>
    </SettingsSections>
  );
}
