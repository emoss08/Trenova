import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { documentControlResourceChoices } from "@/lib/choices";
import type { DocumentControl } from "@/types/document-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { ReadingOffNotice } from "./reading-off-notice";

/** Which rate confirmations become shipments to review, and how closely they are read. */
export function DraftsTab({ onOpenReading }: { onOpenReading: () => void }) {
  const t = useT();
  const { control } = useFormContext<DocumentControl>();
  const drafts = useWatch({ control, name: "enableShipmentDraftExtraction" });

  return (
    <SettingsSections>
      <ReadingOffNotice onOpenReading={onOpenReading} />

      <SettingsSection
        title={t("Shipment drafts")}
        description={t(
          "Turn rate confirmations into shipments your team reviews before they are created.",
        )}
      >
        <SwitchField
          control={control}
          name="enableShipmentDraftExtraction"
          label={t("Draft shipments")}
          description={t("A rate confirmation that is read becomes a draft shipment to review.")}
        />
        <SettingsNested show={drafts}>
          <MultiCheckboxField
            control={control}
            name="shipmentDraftAllowedResources"
            label={t("Draft from")}
            description={t("Rate confirmations uploaded to these records become drafts.")}
            options={documentControlResourceChoices}
            emptyAs="array"
            required={drafts}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("AI reading")}
        description={t("How closely a rate confirmation is read for its stops, dates and rate.")}
      >
        <SwitchField
          control={control}
          name="enableAiAssistedExtraction"
          label={t("Read with AI")}
          description={t(
            "An AI model reads each rate confirmation, so its draft holds more of the load.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
