import { ExtractionNeedsProviderAlert } from "@/components/documents/extraction-needs-provider-alert";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { DocumentControl } from "@/types/document-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** Whether uploaded documents are read at all, how far, and whether their words are searchable. */
export function ReadingTab() {
  const t = useT();
  const { control } = useFormContext<DocumentControl>();
  const reading = useWatch({ control, name: "enableDocumentIntelligence" });

  return (
    <SettingsSections>
      <ExtractionNeedsProviderAlert className="mb-5" />

      <SettingsSection
        title={t("Reading")}
        description={t("Whether Trenova reads the documents your team uploads.")}
      >
        <SwitchField
          control={control}
          name="enableDocumentIntelligence"
          label={t("Read uploaded documents")}
          description={t(
            "Each upload is read so it can be filed, filled in and drafted into a shipment.",
          )}
        />
        <SettingsNested show={reading}>
          <SwitchField
            control={control}
            name="enableOcr"
            label={t("Read scanned pages")}
            description={t(
              "Text is read from scans, photos and faxes that carry none of their own.",
            )}
          />
          <SwitchField
            control={control}
            name="enableFullTextIndexing"
            label={t("Search by content")}
            description={t("People can find a document by the words inside it, not only its name.")}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
