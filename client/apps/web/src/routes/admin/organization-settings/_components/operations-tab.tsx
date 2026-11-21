import { SettingRow, settingRowDescriptionClass } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import {
  organizationCapabilityPresetValues,
  resolveOrganizationCapabilityPreset,
  type OrganizationCapabilityPreset,
} from "@trenova/shared/types/organization-capability";
import { useFormContext, useWatch } from "react-hook-form";

/** How the organization operates, and the identifiers regulators know it by. */
export function OperationsTab() {
  const t = useT();
  const { control } = useFormContext<OrganizationSettings>();
  // Brokerage is part of the plan. Without it the server keeps the stored value whatever is
  // sent, so neither its switch nor a preset that turns it on is offered.
  const brokerage = usePlanFeature(PlanFeature.CarrierBrokerage).available;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Operating model")}
        description={t(
          "Tailors the menus to the freight this organization moves. It only hides features from menus and navigation: it never restricts permissions or API access, and never changes existing records.",
        )}
      >
        {brokerage ? <OperatingModelPreset /> : null}
        {brokerage ? (
          <SwitchField
            control={control}
            name="brokerageEnabled"
            label={t("Brokerage features")}
            description={t("Shows carriers, routing guides, tendering and carrier settlements.")}
          />
        ) : null}
        <SwitchField
          control={control}
          name="assetOperationsEnabled"
          label={t("Asset operations")}
          description={t(
            "Runs its own fleet: shows driver and equipment features, and lets drivers be assigned to shipments.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Regulatory")}
        description={t("The identifiers your organization is registered under.")}
      >
        <InputField
          control={control}
          name="scacCode"
          label={t("SCAC code")}
          description={t("The four-letter Standard Carrier Alpha Code.")}
          placeholder="ACME"
          maxLength={4}
          required
        />
        <InputField
          control={control}
          name="dotNumber"
          label={t("DOT number")}
          description={t("The USDOT number, up to eight digits.")}
          placeholder={t("Enter DOT number")}
          inputMode="numeric"
          maxLength={8}
          required
        />
        <InputField
          control={control}
          name="taxId"
          label={t("Tax ID")}
          description={t("The organization's federal tax identification number.")}
          placeholder={t("Enter tax ID")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}

/**
 * The one question most organizations answer, asked over the two switches under it: a preset
 * sets both, and either can still be changed on its own for a combination no preset covers.
 */
function OperatingModelPreset() {
  const t = useT();
  const { control, setValue } = useFormContext<OrganizationSettings>();
  const [brokerageEnabled, assetOperationsEnabled] = useWatch({
    control,
    name: ["brokerageEnabled", "assetOperationsEnabled"],
  });
  const preset = resolveOrganizationCapabilityPreset({ brokerageEnabled, assetOperationsEnabled });

  const items: SegmentedControlItem<OrganizationCapabilityPreset>[] = [
    { value: "asset", label: t("Asset") },
    { value: "brokerage", label: t("Brokerage") },
    { value: "hybrid", label: t("Hybrid") },
  ];

  const applyPreset = (value: OrganizationCapabilityPreset) => {
    // "custom" names a combination no preset covers; it is shown, never offered.
    if (value === "custom") return;

    const values = organizationCapabilityPresetValues(value);
    const options = { shouldDirty: true, shouldTouch: true, shouldValidate: true };
    setValue("brokerageEnabled", values.brokerageEnabled, options);
    setValue("assetOperationsEnabled", values.assetOperationsEnabled, options);
  };

  return (
    <SettingRow
      label={t("Preset")}
      description={
        <p className={settingRowDescriptionClass}>
          {t("Asset runs its own fleet, brokerage buys capacity, hybrid does both.")}
        </p>
      }
      controlSize="wide"
    >
      <SegmentedControl
        aria-label={t("Operating model preset")}
        items={items}
        value={preset}
        onValueChange={applyPreset}
        fullWidth
      />
    </SettingRow>
  );
}
