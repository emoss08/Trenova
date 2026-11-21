import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { TimezoneField } from "@/components/fields/timezone-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { useT } from "@trenova/shared/i18n/use-t";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import { useFormContext, useWatch } from "react-hook-form";

/** Who the organization is, the address its people sign in at, and where it is registered. */
export function ProfileTab() {
  const t = useT();
  const { control } = useFormContext<OrganizationSettings>();
  const loginSlug = useWatch({ control, name: "loginSlug" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Profile")}
        description={t("How the organization is named and where its day starts and ends.")}
      >
        <InputField
          control={control}
          name="name"
          label={t("Name")}
          description={t("Shown to your people, and on the documents and emails you send.")}
          placeholder={t("Enter organization name")}
          maxLength={100}
          required
        />
        <TimezoneField
          control={control}
          name="timezone"
          label={t("Timezone")}
          description={t(
            "The organization's home timezone, for times that have no place of their own.",
          )}
          placeholder={t("Select timezone")}
          required
        />
        <InputField
          control={control}
          name="loginSlug"
          label={t("Sign-in address")}
          description={
            loginSlug
              ? t("Your people sign in at /login/{0}.", loginSlug)
              : t("Gives your people a sign-in page of their own, such as /login/acme-logistics.")
          }
          placeholder="acme-logistics"
          maxLength={100}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Registered address")}
        description={t(
          "The legal headquarters, printed on your invoices and used for correspondence and compliance.",
        )}
      >
        <InputField
          control={control}
          name="addressLine1"
          label={t("Address line 1")}
          placeholder={t("Enter address")}
          maxLength={150}
          required
        />
        <InputField
          control={control}
          name="addressLine2"
          label={t("Suite or unit")}
          placeholder={t("Enter suite or unit number")}
        />
        <InputField
          control={control}
          name="city"
          label={t("City")}
          placeholder={t("Enter city")}
          required
        />
        <UsStateAutocompleteField
          control={control}
          name="stateId"
          label={t("State")}
          placeholder={t("Select state")}
          required
        />
        <InputField
          control={control}
          name="postalCode"
          label={t("ZIP code")}
          placeholder={t("Enter ZIP code")}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
