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
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { UseFormReturn } from "react-hook-form";
import {
  partnerCountryOptions,
  partnerTimezoneOptions,
  type EDIPartnerFormValues,
} from "../edi-schemas";

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
        title="Profile"
        description="Core identifiers and ownership used to route documents for this trading partner."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label="Partner Code"
              placeholder="SCAC or ISA ID"
              description="Stable identifier used in EDI envelopes, searches, and cross-system references. Avoid changing it after documents are exchanged."
              disabled={disabled || readOnlyInternalFields}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Partner Name"
              placeholder="Partner name"
              description="Display name for dispatch, billing, and support teams. Internal partner names are controlled by the organization connection."
              disabled={disabled || readOnlyInternalFields}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              description="Controls whether this partner is available for active EDI routing and profile selection."
              options={statusChoices}
              isReadOnly={disabled}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <CustomerAutocompleteField
              control={control}
              name="customerId"
              label="Customer"
              placeholder="Select customer"
              description="Links documents from this partner to a customer record for shipment, invoice, and billing workflows."
              clearable
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="country"
              label="Country"
              description="Primary country for this partner. Used as routing context for partner-specific defaults."
              options={partnerCountryOptions}
              isReadOnly={disabled}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="timezone"
              label="Timezone"
              description="Local timezone used when interpreting partner schedules, acknowledgments, and operational timestamps."
              options={partnerTimezoneOptions}
              isReadOnly={disabled}
              isClearable
              placeholder="Select timezone"
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Operational notes for this partner"
              description="Optional notes for operations and implementation teams, such as onboarding status or partner-specific handling rules."
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Contact"
        description="Operational owner used when document delivery, mapping, or transport issues need escalation."
      >
        <FormGroup cols={3}>
          <FormControl>
            <InputField
              control={control}
              name="contactName"
              label="Contact Name"
              placeholder="Contact name"
              description="Primary business or integration contact for this partner."
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="contactEmail"
              label="Contact Email"
              placeholder="ops@example.com"
              description="Email address used for EDI coordination, delivery failures, and onboarding follow-up."
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="contactPhone"
              label="Contact Phone"
              placeholder="Contact phone"
              description="Phone number for urgent operational or implementation escalations."
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Defaults"
        description="Fallback routing, transport, and translation settings used when a document does not specify a narrower profile."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="enabledForInbound"
              label="Inbound Enabled"
              description="Allow documents received from this partner to enter EDI processing."
              disabled={disabled}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enabledForOutbound"
              label="Outbound Enabled"
              description="Allow Trenova to send outbound documents to this partner."
              disabled={disabled}
              outlined
            />
          </FormControl>
          <FormControl>
            <EDICommunicationProfileAutocompleteField
              control={control}
              name="defaultTransportId"
              label="Default Transport Profile"
              placeholder="Select transport profile"
              description="Transport profile used by default for this partner, such as AS2, SFTP, or internal delivery."
              extraSearchParams={{ status: "Active" }}
              clearable
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <EDIMappingProfileAutocompleteField
              control={control}
              name="defaultMappingProfileId"
              label="Default Mapping Profile"
              placeholder="Select mapping profile"
              description="Mapping profile used to translate partner payloads when no document-specific mapping overrides it."
              clearable
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Advanced"
        description="Structured partner settings reserved for integration-specific options and runtime overrides."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <JsonEditorField
              control={control}
              name="settingsJson"
              label="Settings JSON"
              placeholder="{}"
              description="JSON object stored with this partner and sent unchanged to EDI processing services."
              disabled={disabled}
              minHeight="220px"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </Form>
  );
}
