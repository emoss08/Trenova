import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { DocumentControl } from "@/types/document-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { ReadingOffNotice } from "./reading-off-notice";

/** How a document that was read is recognized and filed under one of the organization's types. */
export function FilingTab({ onOpenReading }: { onOpenReading: () => void }) {
  const t = useT();
  const { control } = useFormContext<DocumentControl>();
  const classifies = useWatch({ control, name: "enableAutoClassification" });
  const associates = useWatch({ control, name: "enableAutoDocumentTypeAssociate" });

  return (
    <SettingsSections>
      <ReadingOffNotice onOpenReading={onOpenReading} />

      <SettingsSection
        title={t("Document kinds")}
        description={t(
          "How an upload is recognized as a rate confirmation, a bill of lading or another kind.",
        )}
      >
        <SwitchField
          control={control}
          name="enableAutoClassification"
          label={t("Recognize document kinds")}
          description={t(
            "Each upload is sorted into a kind, such as a proof of delivery, as it is read.",
          )}
        />
        <SettingsNested show={classifies}>
          <SwitchField
            control={control}
            name="enableAiAssistedClassification"
            label={t("Ask AI when unsure")}
            description={t("An AI model settles the kind when the wording alone does not.")}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Document types")}
        description={t("Which of your document types an upload is filed under.")}
      >
        <SwitchField
          control={control}
          name="enableAutoDocumentTypeAssociate"
          label={t("File under a document type")}
          description={t("A recognized document is given the matching type from your list.")}
        />
        <SettingsNested show={associates}>
          <SwitchField
            control={control}
            name="enableAutoCreateDocumentTypes"
            label={t("Add missing types")}
            description={t("When no type matches, one is added so the document is still filed.")}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
