import { useT } from "@trenova/shared/i18n/use-t";
import {
  FleetCodeAutocompleteField,
  JobPositionAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { CustomFieldsSection } from "@/components/custom-fields-section";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import {
  cdlClassChoices,
  complianceStatusChoices,
  driverTypeChoices,
  endorsementTypeChoices,
  genderChoices,
  statusChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { cn } from "@trenova/shared/lib/utils";
import type { Worker } from "@trenova/shared/types/worker";
import { useFormContext, useWatch } from "react-hook-form";

export function GeneralTab() {
  const t = useT();

  const { control } = useFormContext<Worker>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("General information for the worker.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              options={statusChoices}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Set from the timeline")}
              isReadOnly
              description={t(
                "Employment status moves through the Timeline tab — record a Terminated, Rehired or similar event.",
              )}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={workerTypeChoices}
              rules={{ required: true }}
              name="type"
              label={t("Worker Type")}
              placeholder={t("Employee or contractor")}
              description={t("Whether the worker is an employee or contractor.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="firstName"
              label={t("First Name")}
              placeholder={t("e.g. Maria")}
              description={t("Legal first name as it appears on the CDL.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="lastName"
              label={t("Last Name")}
              placeholder={t("e.g. Alvarez")}
              description={t("Legal last name as it appears on the CDL.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={genderChoices}
              rules={{ required: true }}
              name="gender"
              label={t("Gender")}
              placeholder={t("Pick a gender")}
              description={t("Gender as recorded on the license.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={driverTypeChoices}
              rules={{ required: true }}
              name="driverType"
              label={t("Driver Type")}
              placeholder={t("Local, regional, OTR or team")}
              description={t("Type of driving operations (Local, Regional, OTR, Team).")}
            />
          </FormControl>
          <FormControl cols="full" className="pb-2">
            <FleetCodeAutocompleteField<Worker>
              name="fleetCodeId"
              control={control}
              clearable
              label={t("Fleet Code")}
              placeholder={t("Search fleet codes")}
              description={t("The fleet code associated with this worker.")}
            />
          </FormControl>
          <FormControl cols="full" className="pb-2">
            <JobPositionAutocompleteField<Worker>
              name="positionId"
              control={control}
              driving
              clearable
              label={t("Position")}
              placeholder={t("Driving position")}
              description={t(
                "The title the roster is counted by. Front-office titles are held by users, not workers.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Address Information")}
        description={t("Address information for the worker's residence.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="addressLine1"
              label={t("Address Line 1")}
              placeholder={t("e.g. 1200 W Main St")}
              description={t("Street address of the worker's residence.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label={t("Address Line 2")}
              placeholder={t("e.g. Apt 4B")}
              description={t("Apartment, suite, or unit number (optional).")}
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label={t("City")}
              placeholder={t("e.g. Joliet")}
              description={t("City of residence.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label={t("State")}
              rules={{ required: true }}
              placeholder={t("Search states")}
              description={t("U.S. state of residence.")}
            />
          </FormControl>
          <FormControl className="pb-2">
            <InputField
              control={control}
              rules={{ required: true }}
              name="postalCode"
              label={t("Postal Code")}
              placeholder={t("e.g. 60432")}
              description={t("5-digit ZIP code (or ZIP+4).")}
              maxLength={10}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Contact Information")}
        description={t("Contact information for the worker.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="email"
              label={t("Email")}
              placeholder={t("e.g. driver@example.com")}
              description={t("Worker's email address.")}
              maxLength={255}
            />
          </FormControl>
          <FormControl className="w-full">
            <PhoneNumberField
              control={control}
              name="phoneNumber"
              label={t("Phone Number")}
              placeholder="(555) 555-0100"
              description={t("Worker's primary phone number.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emergencyContactName"
              label={t("Emergency Contact Name")}
              placeholder={t("e.g. Ana Alvarez")}
              description={t("Name of emergency contact.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <PhoneNumberField
              control={control}
              name="emergencyContactPhone"
              label={t("Emergency Contact Phone")}
              placeholder="(555) 555-0100"
              description={t("Phone number of emergency contact.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <CustomFieldsSection resourceType="worker" control={control} />
    </div>
  );
}

export function EmploymentTab() {
  const t = useT();

  const { control } = useFormContext<Worker>();
  const endorsement = useWatch({ control, name: "profile.endorsement" });
  const requiresHazmatExpiry = endorsement === "H" || endorsement === "X";

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Employment Information")}
        description={t("Employment information for the worker.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.dob"
              label={t("Date of Birth")}
              rules={{ required: true }}
              description={t("Date of birth as shown on the license.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.hireDate"
              label={t("Hire Date")}
              rules={{ required: true }}
              description={t("Date the worker was hired; tenure is counted from here.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl className="pb-2">
            <AutoCompleteDateField
              control={control}
              name="profile.terminationDate"
              label={t("Termination Date")}
              description={t("Set by a Terminated event in the employment history.")}
              placeholder={t("Not terminated")}
              disabled
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("License Details")}
        description={t("License information for the worker.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="profile.licenseNumber"
              label={t("License Number")}
              placeholder={t("e.g. A123-4567-8901")}
              description={t("CDL number as printed on the license.")}
              maxLength={50}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="profile.licenseStateId"
              label={t("License State")}
              placeholder={t("Search states")}
              description={t("State that issued the license.")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.licenseExpiry"
              label={t("License Expiry")}
              rules={{ required: true }}
              description={t("When the CDL expires.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={cdlClassChoices}
              rules={{ required: true }}
              name="profile.cdlClass"
              label={t("CDL Class")}
              placeholder={t("A, B or C")}
              description={t("Commercial driver's license class (A, B, or C).")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.cdlRestrictions"
              label={t("CDL Restrictions")}
              placeholder={t("e.g. L, Z")}
              description={t("Restriction codes printed on the CDL, such as L for no air brakes.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl className={cn(!requiresHazmatExpiry ? "pb-2" : "")}>
            <SelectField
              control={control}
              options={endorsementTypeChoices}
              rules={{ required: true }}
              name="profile.endorsement"
              label={t("Endorsement")}
              placeholder={t("Pick an endorsement")}
              description={t("CDL endorsement; H and X require a hazmat expiry date.")}
            />
          </FormControl>
          {requiresHazmatExpiry && (
            <FormControl className="pb-2">
              <AutoCompleteDateField
                control={control}
                name="profile.hazmatExpiry"
                label={t("Hazmat Expiry")}
                rules={{ required: requiresHazmatExpiry }}
                description={t("Expiration date of hazmat endorsement.")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Medical Certification")}
        description={t("Medical certification information for the worker.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.medicalCardExpiry"
              label={t("Medical Card Expiry")}
              description={t("Expiration date of medical examiner's certificate.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.physicalDueDate"
              label={t("Physical Due Date")}
              description={t("Next physical examination due date.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.medicalExaminerName"
              label={t("Medical Examiner Name")}
              placeholder={t("e.g. Dr. J. Patel")}
              description={t("Name of the medical examiner.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.medicalExaminerNpi"
              label={t("Medical Examiner NPI")}
              placeholder={t("10-digit NPI")}
              description={t("National Provider Identifier of the medical examiner.")}
              maxLength={20}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}

export function ComplianceTab() {
  const t = useT();

  const { control } = useFormContext<Worker>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Compliance Status")}
        description={t("Compliance status information for the worker.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              options={complianceStatusChoices}
              rules={{ required: true }}
              name="profile.complianceStatus"
              label={t("Compliance Status")}
              placeholder={t("Pick a status")}
              description={t("Current compliance status of the worker.")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.mvrDueDate"
              label={t("MVR Due Date")}
              description={t("Next motor vehicle record check due date.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.isQualified"
              label={t("Is Qualified")}
              description={t("Whether the worker is qualified to drive.")}
            />
          </FormControl>
          <FormControl className="pb-2">
            <InputField
              control={control}
              name="profile.disqualificationReason"
              label={t("Disqualification Reason")}
              placeholder={t("e.g. Medical certificate lapsed")}
              description={t("Reason for disqualification (if applicable).")}
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("TWIC Credentials")}
        description={t("TWIC credentials information for the worker.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="profile.twicCardNumber"
              label={t("TWIC Card Number")}
              placeholder={t("e.g. 1234567890")}
              description={t("Transportation Worker Identification Credential number.")}
              maxLength={50}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.twicExpiry"
              label={t("TWIC Expiry")}
              description={t("Expiration date of TWIC card.")}
              placeholder={t("MM/DD/YYYY")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Exemptions & Availability")}
        description={t("Exemptions and availability information for the worker.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.eldExempt"
              label={t("ELD Exempt")}
              description={t("Whether the worker is exempt from ELD requirements.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.shortHaulExempt"
              label={t("Short Haul Exempt")}
              description={t("Whether the worker qualifies for short-haul exemption.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="availableForDispatch"
              label={t("Available for Dispatch")}
              description={t("Whether the worker can be dispatched.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="canBeAssigned"
              label={t("Can Be Assigned")}
              description={t("Whether the worker can be assigned to equipment.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
