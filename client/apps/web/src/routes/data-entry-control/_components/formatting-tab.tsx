import { SelectField } from "@/components/fields/select-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { caseFormatChoices, emailCaseFormatChoices } from "@/lib/choices";
import type { DataEntryControl } from "@/types/data-entry-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** How each kind of text is capitalized when a record is saved, wherever it was entered. */
export function FormattingTab() {
  const t = useT();
  const { control } = useFormContext<DataEntryControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Case formatting")}
        description={t("How entered text is capitalized when a record is saved.")}
      >
        <SelectField
          control={control}
          name="codeCase"
          label={t("Codes")}
          description={t(
            "Applies to the codes of customers, carriers, factoring companies, locations, fleet codes, equipment types, service and shipment types, accessorials, hazardous materials, document types, account types and GL accounts.",
          )}
          options={caseFormatChoices}
          required
        />
        <SelectField
          control={control}
          name="nameCase"
          label={t("Names")}
          description={t(
            "Applies to the names of workers and their emergency contacts, customers, carriers and their contacts, factoring companies, locations, commodities, hazardous materials, document types, account types, GL accounts and fiscal years.",
          )}
          options={caseFormatChoices}
          required
        />
        <SelectField
          control={control}
          name="emailCase"
          label={t("Email addresses")}
          description={t(
            "Applies to worker, carrier, carrier contact and factoring company emails and to customer status update and billing email recipients, never to sign-in emails.",
          )}
          options={emailCaseFormatChoices}
          required
        />
        <SelectField
          control={control}
          name="cityCase"
          label={t("Cities")}
          description={t(
            "Applies to the city on workers, customers, carriers (including the remit-to city), factoring companies and locations.",
          )}
          options={caseFormatChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
