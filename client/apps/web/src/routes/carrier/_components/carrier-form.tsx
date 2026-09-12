import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

  const { control } = useFormContext<Carrier>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("Core identifiers used across the system to reference this carrier.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t(
                "Controls whether this carrier appears in active lookups. Do Not Use blocks the carrier from new assignments.",
              )}
              options={carrierStatusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("e.g., SWFT")}
              description={t(
                "Short alphanumeric identifier used in load references and quick-search. Must be unique across your organization.",
              )}
              maxLength={10}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("e.g., Swift Transportation Co.")}
              description={t(
                "Full legal name of the carrier. This appears on rate confirmations and all printed documents.",
              )}
              maxLength={255}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="dbaName"
              label={t("DBA Name")}
              placeholder={t("Doing business as")}
              description={t(
                "Trade name the carrier operates under when it differs from the legal name.",
              )}
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="carrierType"
              label={t("Carrier Type")}
              placeholder={t("Carrier Type")}
              description={t(
                "Operating authority classification: common, contract, broker, or exempt.",
              )}
              options={carrierTypeChoices}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Operating Authority")}
        description={t("Federal identifiers used for safety lookups and compliance monitoring.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="dotNumber"
              label={t("DOT Number")}
              placeholder={t("e.g., 1234567")}
              description={t("USDOT number issued by the FMCSA. Digits only, up to 12 characters.")}
              maxLength={12}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="mcNumber"
              label={t("MC Number")}
              placeholder={t("e.g., 987654")}
              description={t(
                "Motor carrier (operating authority) number. Digits only, up to 12 characters.",
              )}
              maxLength={12}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="scac"
              label="SCAC"
              placeholder={t("e.g., SWFT")}
              description={t(
                "Standard Carrier Alpha Code: 2-4 uppercase letters used on EDI documents and BOLs.",
              )}
              maxLength={4}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Address & Contact")}
        description={t("Primary business address and contact details for this carrier.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine1"
              label={t("Address Line 1")}
              placeholder={t("Street address")}
              description={t("Street address of the carrier's primary office.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label={t("Address Line 2")}
              placeholder={t("Suite, floor, building, etc.")}
              description={t(
                "Additional address details such as suite number, floor, or building name.",
              )}
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="city"
              label={t("City")}
              placeholder={t("City")}
              description={t("City where the carrier's primary office is located.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label={t("State")}
              placeholder={t("State")}
              description={t("U.S. state for the carrier's primary address.")}
              clearable
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="postalCode"
              label={t("Postal Code")}
              placeholder={t("e.g., 90210")}
              description={t("ZIP or ZIP+4 code for the carrier's primary address.")}
              maxLength={10}
            />
          </FormControl>
          <FormControl>
            <PhoneNumberField
              control={control}
              name="phone"
              label={t("Phone")}
              placeholder={t("Phone")}
              description={t("Main dispatch or office phone number for this carrier.")}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="email"
              label={t("Email")}
              placeholder={t("e.g., dispatch@carrier.com")}
              description={t("Primary email address used for tenders and rate confirmations.")}
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Additional Details")}
        description={t("External references and internal notes for this carrier.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="externalId"
              label={t("External ID")}
              placeholder={t("e.g., TMS-10042")}
              description={t(
                "Identifier from an external system (ERP, load board, EDI partner ID) for cross-system reconciliation.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("Internal notes about this carrier")}
              description={t("Free-form internal notes. Not shared with the carrier.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
