import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SettingsForm } from "@/components/settings/settings-form";
import { SettingsReveal } from "@/components/settings/settings-reveal";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { usePermission } from "@/hooks/use-permission";
import { capture } from "@/lib/queries/capture";
import { documentControl } from "@/lib/queries/document-control";
import { documentControlService } from "@/services/document-control";
import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useFormContext, useWatch } from "react-hook-form";
import { useCaptureTabs, type CaptureTab } from "../capture-tabs";

const save = (values: DocumentControl) => documentControlService.update(values);

/** Whether people may scan and print at all, and what changes once they can. */
function CaptureSettings({ readOnly }: { readOnly: boolean }) {
  const t = useT();
  const { control } = useFormContext<DocumentControl>();
  const enabled = useWatch({ control, name: "enableCapture" });

  return (
    <SettingsSections>
      {readOnly ? (
        <Alert variant="info" size="sm" className="mb-5">
          <AlertDescription>
            {t("You can view these settings but not change them.")}
          </AlertDescription>
        </Alert>
      ) : null}

      <SettingsSection
        title={t("Availability")}
        description={t("Whether people can scan and print into Trenova with Trenova Capture.")}
      >
        <SwitchField
          control={control}
          name="enableCapture"
          label={t("Scanning and printing")}
          description={t(
            "Turning it off stops every paired computer until it is turned on again; nothing is unpaired.",
          )}
          disabled={readOnly}
        />
      </SettingsSection>

      <SettingsReveal show={enabled}>
        <SettingsSection
          title={t("Filing")}
          description={t("What happens to a stack once it is read, and how long unfiled pages wait.")}
        >
          <SwitchField
            control={control}
            name="captureAutoFileCoverSheets"
            label={t("File behind cover sheets on their own")}
            description={t(
              "A document that follows a Trenova cover sheet goes onto the sheet's record without waiting in Intake. Anything else still waits for a person.",
            )}
            disabled={readOnly}
          />
          <NumberField
            control={control}
            name="captureRetentionDays"
            label={t("Keep unfiled pages for")}
            description={t(
              "From 1 to 365 days. Pages nobody files by then are deleted, and whoever scanned them is reminded a week before.",
            )}
            sideText={t("days")}
            min={1}
            max={365}
            required={t("Enter how many days to keep unfiled pages")}
            disabled={readOnly}
          />
        </SettingsSection>

        <SettingsSection
          title={t("Desktop app")}
          description={t("Which versions of Trenova Capture may connect, and how they update.")}
        >
          <InputField
            control={control}
            name="captureMinAgentVersion"
            label={t("Oldest version allowed")}
            description={t(
              "A computer on an older version must update before it can scan. Leave empty to allow any version.",
            )}
            placeholder="1.0.0"
            maxLength={20}
            disabled={readOnly}
          />
          <SwitchField
            control={control}
            name="captureAllowAutoUpdate"
            label={t("Update on its own")}
            description={t(
              "New versions install as they are released. Turn off where IT rolls out software.",
            )}
            disabled={readOnly}
          />
        </SettingsSection>
      </SettingsReveal>
    </SettingsSections>
  );
}

/**
 * The organization's capture settings. They live on the document settings, as the rest of
 * how documents are handled does, so saving here saves that record; the Document extraction
 * page sends these back as it read them. Whoever may read but not update them sees them held,
 * with nothing to save.
 */
export default function CaptureSettingsForm() {
  const { data } = useSuspenseQuery({ ...documentControl.get() });
  const { allowed: canUpdate } = usePermission(Resource.DocumentControl, Operation.Update);
  const tabs = useCaptureTabs().filter((tab) => tab.value === "settings");

  return (
    <SettingsForm<DocumentControl, CaptureTab>
      values={data}
      schema={documentControlSchema}
      queryKey={documentControl.get._def}
      save={save}
      refresh={[capture.access._def]}
      resourceName="Capture settings"
      tabs={tabs}
      content={() => <CaptureSettings readOnly={!canUpdate} />}
    />
  );
}
