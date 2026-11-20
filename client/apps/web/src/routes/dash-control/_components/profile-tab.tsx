import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import type { DashControl } from "@/types/dash-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** What drivers keep current on their own record, and asking for time off. */
export function ProfileTab() {
  const t = useT();
  const { control } = useFormContext<DashControl>();
  const editsContact = useWatch({ control, name: "allowContactInfoEdit" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Driver record")}
        description={t(
          "What drivers update on their own record; license and medical dates are always set by the office.",
        )}
      >
        <SwitchField
          control={control}
          name="allowProfileDocumentUpload"
          label={t("Qualification documents")}
          description={t(
            "Drivers upload a renewed license, medical card or other qualification document.",
          )}
        />
        <SwitchField
          control={control}
          name="allowContactInfoEdit"
          label={t("Contact details")}
          description={t("Drivers keep their phone, address and emergency contact current.")}
        />
        <SettingsNested show={editsContact}>
          <SwitchField
            control={control}
            name="requireContactChangeApproval"
            label={t("Approve changes")}
            description={t(
              "A driver's change waits for the office to approve it before it reaches their record.",
            )}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Time off")}
        description={t("Whether drivers can ask for time off from Dash.")}
      >
        <SwitchField
          control={control}
          name="allowPtoRequests"
          label={t("Time-off requests")}
          description={t("Drivers ask for time off from Dash, and the request goes for approval.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
