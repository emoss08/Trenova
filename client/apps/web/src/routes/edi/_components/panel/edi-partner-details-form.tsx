import { useT } from "@trenova/shared/i18n/use-t";
import {
  CustomerAutocompleteField,
  EDICommunicationProfileAutocompleteField,
  EDIMappingProfileAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { JsonEditorField } from "@/components/fields/json-editor-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { statusChoices, timezoneGroupedChoices } from "@/lib/choices";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { UseFormReturn } from "react-hook-form";
import { partnerCountryOptions, type EDIPartnerFormValues } from "../edi-schemas";

type PartnerDetailsFormProps = {
  id: string;
  form: UseFormReturn<EDIPartnerFormValues>;
  disabled: boolean;
  readOnlyInternalFields: boolean;
  onSubmit: (values: EDIPartnerFormValues) => void;
};

export function PartnerDetailsForm({
  id,
  form,
  disabled,
  readOnlyInternalFields,
  onSubmit,
}: PartnerDetailsFormProps) {
  const t = useT();

  const { control, handleSubmit } = form;

  return (
    <Form
      id={id}
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit(onSubmit)(event);
      }}
    >
      <FormSection
        title={t("Profile")}
        description={t(
          "Core identifiers and ownership used to route documents for this trading partner.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Partner Code")}
              placeholder={t("SCAC or ISA ID")}
              description={t(
                "Stable identifier used in EDI envelopes, searches, and cross-system references. Avoid changing it after documents are exchanged.",
              )}
              disabled={disabled || readOnlyInternalFields}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Partner Name")}
              placeholder={t("Partner name")}
              description={t(
                "Display name for dispatch, billing, and support teams. Internal partner names are controlled by the organization connection.",
              )}
              disabled={disabled || readOnlyInternalFields}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              description={t(
                "Controls whether this partner is available for active EDI routing and profile selection.",
              )}
              options={statusChoices}
              isReadOnly={disabled}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <CustomerAutocompleteField
              control={control}
              name="customerId"
              label={t("Customer")}
              placeholder={t("Select customer")}
              description={t(
                "Links documents from this partner to a customer record for shipment, invoice, and billing workflows.",
              )}
              clearable
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="country"
              label={t("Country")}
              description={t(
                "Primary country for this partner. Used as routing context for partner-specific defaults.",
              )}
              options={partnerCountryOptions}
              isReadOnly={disabled}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="timezone"
              label={t("Timezone")}
              description={t(
                "Local timezone used when interpreting partner schedules, acknowledgments, and operational timestamps.",
              )}
              groups={timezoneGroupedChoices}
              renderOption={(option) => (
                <span className="flex w-full items-center justify-between gap-3">
                  <span>{t(option.label)}</span>
                  {option.description && (
                    <span className="text-muted-foreground text-xs">{t(option.description)}</span>
                  )}
                </span>
              )}
              isReadOnly={disabled}
              isClearable
              placeholder={t("Select timezone")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Operational notes for this partner")}
              description={t(
                "Optional notes for operations and implementation teams, such as onboarding status or partner-specific handling rules.",
              )}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Contact")}
        description={t(
          "Operational owner used when document delivery, mapping, or transport issues need escalation.",
        )}
      >
        <FormGroup cols={3}>
          <FormControl>
            <InputField
              control={control}
              name="contactName"
              label={t("Contact Name")}
              placeholder={t("Contact name")}
              description={t("Primary business or integration contact for this partner.")}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="contactEmail"
              label={t("Contact Email")}
              placeholder={t("ops@example.com")}
              description={t(
                "Email address used for EDI coordination, delivery failures, and onboarding follow-up.",
              )}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="contactPhone"
              label={t("Contact Phone")}
              placeholder={t("Contact phone")}
              description={t("Phone number for urgent operational or implementation escalations.")}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Defaults")}
        description={t(
          "Fallback routing, transport, and translation settings used when a document does not specify a narrower profile.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="enabledForInbound"
              label={t("Inbound Enabled")}
              description={t("Allow documents received from this partner to enter EDI processing.")}
              disabled={disabled}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enabledForOutbound"
              label={t("Outbound Enabled")}
              description={t("Allow Trenova to send outbound documents to this partner.")}
              disabled={disabled}
              outlined
            />
          </FormControl>
          <FormControl>
            <EDICommunicationProfileAutocompleteField
              control={control}
              name="defaultTransportId"
              label={t("Default Transport Profile")}
              placeholder={t("Select transport profile")}
              description={t(
                "Transport profile used by default for this partner, such as AS2, SFTP, or internal delivery.",
              )}
              extraSearchParams={{ status: "Active" }}
              clearable
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <EDIMappingProfileAutocompleteField
              control={control}
              name="defaultMappingProfileId"
              label={t("Default Mapping Profile")}
              placeholder={t("Select mapping profile")}
              description={t(
                "Mapping profile used to translate partner payloads when no document-specific mapping overrides it.",
              )}
              clearable
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Advanced")}
        description={t(
          "Structured partner settings reserved for integration-specific options and runtime overrides.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <JsonEditorField
              control={control}
              name="settingsJson"
              label={t("Settings JSON")}
              placeholder="{}"
              description={t(
                "JSON object stored with this partner and sent unchanged to EDI processing services.",
              )}
              disabled={disabled}
              minHeight="220px"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </Form>
  );
}
