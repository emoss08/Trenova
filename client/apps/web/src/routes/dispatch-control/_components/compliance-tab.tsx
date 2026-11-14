import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { complianceEnforcementLevelChoices } from "@/lib/choices";
import type { DispatchControl } from "@/types/dispatch-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * What is checked before a driver is put on a move: the DOT checks, then the
 * organization's own rules.
 */
export function ComplianceTab() {
  const t = useT();
  const { control, setValue } = useFormContext<DispatchControl>();
  const enforceHosCompliance = useWatch({ control, name: "enforceHosCompliance" });

  useEffect(() => {
    if (!enforceHosCompliance) {
      setValue("enforceMedicalCertCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceDriverQualificationCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceHazmatCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceDrugAndAlcoholCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
  }, [enforceHosCompliance, setValue]);

  return (
    <SettingsSections>
      <SettingsSection
        title={t("DOT compliance enforcement")}
        description={t(
          "Configure dispatch-time compliance checks for worker qualification, medical certification, hazmat eligibility, and testing requirements.",
        )}
      >
        <SwitchField
          control={control}
          name="enforceHosCompliance"
          label={t("Enable DOT compliance enforcement")}
          description={t(
            "When enabled, the system applies configured compliance checks before assignments proceed.",
          )}
        />
        <SettingsNested show={enforceHosCompliance}>
          <SwitchField
            control={control}
            name="enforceMedicalCertCompliance"
            label={t("Medical certification validation")}
            description={t("Require current medical certification before assignment.")}
          />
          <SwitchField
            control={control}
            name="enforceDriverQualificationCompliance"
            label={t("Driver qualification verification")}
            description={t(
              "Require valid driver qualification and license state before assignment.",
            )}
          />
          <SwitchField
            control={control}
            name="enforceHazmatCompliance"
            label={t("Hazardous materials compliance")}
            description={t("Require hazmat-specific compliance checks for regulated loads.")}
          />
          <SwitchField
            control={control}
            name="enforceDrugAndAlcoholCompliance"
            label={t("Drug and alcohol testing compliance")}
            description={t("Require testing compliance checks before assignment.")}
          />
          <SelectField
            control={control}
            name="complianceEnforcementLevel"
            label={t("Compliance enforcement level")}
            description={t("Select whether violations should warn, block, or be audit-only.")}
            options={complianceEnforcementLevelChoices}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Assignment rules")}
        description={t(
          "What a move must have, and a driver must be free of, before it is dispatched.",
        )}
      >
        <SwitchField
          control={control}
          name="enforceWorkerAssign"
          label={t("Require worker assignment")}
          description={t("Prevent dispatching without an assigned worker.")}
        />
        <SwitchField
          control={control}
          name="enforceTrailerContinuity"
          label={t("Require trailer continuity")}
          description={t("Enforce trailer continuity rules across movement chains.")}
        />
        <SwitchField
          control={control}
          name="enforceWorkerPtaRestrictions"
          label={t("Enforce worker PTA restrictions")}
          description={t(
            "Apply worker availability and paid-time-away restrictions during assignment.",
          )}
        />
        <SwitchField
          control={control}
          name="enforceWorkerTractorFleetContinuity"
          label={t("Enforce worker tractor fleet continuity")}
          description={t("Require worker-to-tractor fleet continuity where configured.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
