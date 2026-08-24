import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { carrierStatusChoices, carrierTypeChoices } from "@/lib/choices";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFormContext } from "react-hook-form";

export function CarrierForm() {
  const { control } = useFormContext<Carrier>();

  return (
    <div className="space-y-6">
      <FormSection
        title="General Information"
        description="Core identifiers used across the system to reference this carrier."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="Controls whether this carrier appears in active lookups. Do Not Use blocks the carrier from new assignments."
              options={carrierStatusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label="Code"
              placeholder="e.g., SWFT"
              description="Short alphanumeric identifier used in load references and quick-search. Must be unique across your organization."
              maxLength={10}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              placeholder="e.g., Swift Transportation Co."
              description="Full legal name of the carrier. This appears on rate confirmations and all printed documents."
              maxLength={255}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="dbaName"
              label="DBA Name"
              placeholder="Doing business as"
              description="Trade name the carrier operates under when it differs from the legal name."
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="carrierType"
              label="Carrier Type"
              placeholder="Carrier Type"
              description="Operating authority classification: common, contract, broker, or exempt."
              options={carrierTypeChoices}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Operating Authority"
        description="Federal identifiers used for safety lookups and compliance monitoring."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="dotNumber"
              label="DOT Number"
              placeholder="e.g., 1234567"
              description="USDOT number issued by the FMCSA. Digits only, up to 12 characters."
              maxLength={12}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="mcNumber"
              label="MC Number"
              placeholder="e.g., 987654"
              description="Motor carrier (operating authority) number. Digits only, up to 12 characters."
              maxLength={12}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="scac"
              label="SCAC"
              placeholder="e.g., SWFT"
              description="Standard Carrier Alpha Code: 2-4 uppercase letters used on EDI documents and BOLs."
              maxLength={4}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Address & Contact"
        description="Primary business address and contact details for this carrier."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine1"
              label="Address Line 1"
              placeholder="Street address"
              description="Street address of the carrier's primary office."
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Address Line 2"
              placeholder="Suite, floor, building, etc."
              description="Additional address details such as suite number, floor, or building name."
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="city"
              label="City"
              placeholder="City"
              description="City where the carrier's primary office is located."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label="State"
              placeholder="State"
              description="U.S. state for the carrier's primary address."
              clearable
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="postalCode"
              label="Postal Code"
              placeholder="e.g., 90210"
              description="ZIP or ZIP+4 code for the carrier's primary address."
              maxLength={10}
            />
          </FormControl>
          <FormControl>
            <PhoneNumberField
              control={control}
              name="phone"
              label="Phone"
              placeholder="Phone"
              description="Main dispatch or office phone number for this carrier."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="email"
              label="Email"
              placeholder="e.g., dispatch@carrier.com"
              description="Primary email address used for tenders and rate confirmations."
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Additional Details"
        description="External references and internal notes for this carrier."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="externalId"
              label="External ID"
              placeholder="e.g., TMS-10042"
              description="Identifier from an external system (ERP, load board, EDI partner ID) for cross-system reconciliation."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label="Notes"
              placeholder="Internal notes about this carrier"
              description="Free-form internal notes. Not shared with the carrier."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
