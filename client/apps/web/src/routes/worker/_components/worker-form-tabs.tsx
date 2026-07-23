import {
  FleetCodeAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { CustomFieldsSection } from "@/components/custom-fields-section";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  cdlClassChoices,
  complianceStatusChoices,
  driverTypeChoices,
  endorsementTypeChoices,
  genderChoices,
  statusChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { cn } from "@/lib/utils";
import type { Worker } from "@/types/worker";
import { useFormContext, useWatch } from "react-hook-form";

export function GeneralTab() {
  const { control } = useFormContext<Worker>();

  return (
    <div className="space-y-6">
      <FormSection
        title="General Information"
        description="General information for the worker."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              options={statusChoices}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="Current employment status of the worker."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={workerTypeChoices}
              rules={{ required: true }}
              name="type"
              label="Worker Type"
              placeholder="Worker Type"
              description="Whether the worker is an employee or contractor."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="firstName"
              label="First Name"
              placeholder="First Name"
              description="Worker's legal first name."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="lastName"
              label="Last Name"
              placeholder="Last Name"
              description="Worker's legal last name."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={genderChoices}
              rules={{ required: true }}
              name="gender"
              label="Gender"
              placeholder="Gender"
              description="Worker's gender."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={driverTypeChoices}
              rules={{ required: true }}
              name="driverType"
              label="Driver Type"
              placeholder="Driver Type"
              description="Type of driving operations (Local, Regional, OTR, Team)."
            />
          </FormControl>
          <FormControl cols="full" className="pb-2">
            <FleetCodeAutocompleteField<Worker>
              name="fleetCodeId"
              control={control}
              clearable
              label="Fleet Code"
              placeholder="Fleet Code"
              description="The fleet code associated with this worker."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Address Information"
        description="Address information for the worker's residence."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="addressLine1"
              label="Address Line 1"
              placeholder="Address Line 1"
              description="Street address."
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Address Line 2"
              placeholder="Address Line 2"
              description="Apartment, suite, or unit number (optional)."
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label="City"
              placeholder="City"
              description="City of residence."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label="State"
              rules={{ required: true }}
              placeholder="State"
              description="U.S. state of residence."
            />
          </FormControl>
          <FormControl className="pb-2">
            <InputField
              control={control}
              rules={{ required: true }}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="5-digit ZIP code (or ZIP+4)."
              maxLength={10}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Contact Information"
        description="Contact information for the worker."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="email"
              label="Email"
              placeholder="Email"
              description="Worker's email address."
              maxLength={255}
            />
          </FormControl>
          <FormControl className="w-full">
            <PhoneNumberField
              control={control}
              name="phoneNumber"
              label="Phone Number"
              placeholder="Phone Number"
              description="Worker's primary phone number."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emergencyContactName"
              label="Emergency Contact Name"
              placeholder="Emergency Contact Name"
              description="Name of emergency contact."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <PhoneNumberField
              control={control}
              name="emergencyContactPhone"
              label="Emergency Contact Phone"
              placeholder="Emergency Contact Phone"
              description="Phone number of emergency contact."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <CustomFieldsSection resourceType="worker" control={control} />
    </div>
  );
}

export function EmploymentTab() {
  const { control } = useFormContext<Worker>();
  const endorsement = useWatch({ control, name: "profile.endorsement" });
  const requiresHazmatExpiry = endorsement === "H" || endorsement === "X";

  return (
    <div className="space-y-6">
      <FormSection
        title="Employment Information"
        description="Employment information for the worker."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.dob"
              label="Date of Birth"
              rules={{ required: true }}
              description="Worker's date of birth."
              placeholder="Date of Birth"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.hireDate"
              label="Hire Date"
              rules={{ required: true }}
              description="Date the worker was hired."
              placeholder="Hire Date"
            />
          </FormControl>
          <FormControl className="pb-2">
            <AutoCompleteDateField
              control={control}
              name="profile.terminationDate"
              label="Termination Date"
              description="Date employment ended (if applicable)."
              placeholder="Termination Date"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="License Details"
        description="License information for the worker."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="profile.licenseNumber"
              label="License Number"
              placeholder="License Number"
              description="Driver's license number."
              maxLength={50}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="profile.licenseStateId"
              label="License State"
              placeholder="License State"
              description="State that issued the license."
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.licenseExpiry"
              label="License Expiry"
              rules={{ required: true }}
              description="Expiration date of the license."
              placeholder="License Expiry"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              options={cdlClassChoices}
              rules={{ required: true }}
              name="profile.cdlClass"
              label="CDL Class"
              placeholder="CDL Class"
              description="Commercial driver's license class (A, B, or C)."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.cdlRestrictions"
              label="CDL Restrictions"
              placeholder="CDL Restrictions"
              description="Any restrictions on the CDL."
              maxLength={100}
            />
          </FormControl>
          <FormControl className={cn(!requiresHazmatExpiry ? "pb-2" : "")}>
            <SelectField
              control={control}
              options={endorsementTypeChoices}
              rules={{ required: true }}
              name="profile.endorsement"
              label="Endorsement"
              placeholder="Endorsement"
              description="CDL endorsement type."
            />
          </FormControl>
          {requiresHazmatExpiry && (
            <FormControl className="pb-2">
              <AutoCompleteDateField
                control={control}
                name="profile.hazmatExpiry"
                label="Hazmat Expiry"
                rules={{ required: requiresHazmatExpiry }}
                description="Expiration date of hazmat endorsement."
                placeholder="Hazmat Expiry"
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title="Medical Certification"
        description="Medical certification information for the worker."
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.medicalCardExpiry"
              label="Medical Card Expiry"
              description="Expiration date of medical examiner's certificate."
              placeholder="Medical Card Expiry"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.physicalDueDate"
              label="Physical Due Date"
              description="Next physical examination due date."
              placeholder="Physical Due Date"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.medicalExaminerName"
              label="Medical Examiner Name"
              placeholder="Medical Examiner Name"
              description="Name of the medical examiner."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="profile.medicalExaminerNpi"
              label="Medical Examiner NPI"
              placeholder="Medical Examiner NPI"
              description="National Provider Identifier of the medical examiner."
              maxLength={20}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}

export function ComplianceTab() {
  const { control } = useFormContext<Worker>();

  return (
    <div className="space-y-6">
      <FormSection
        title="Compliance Status"
        description="Compliance status information for the worker."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              options={complianceStatusChoices}
              rules={{ required: true }}
              name="profile.complianceStatus"
              label="Compliance Status"
              placeholder="Compliance Status"
              description="Current compliance status of the worker."
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.mvrDueDate"
              label="MVR Due Date"
              description="Next motor vehicle record check due date."
              placeholder="MVR Due Date"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.isQualified"
              label="Is Qualified"
              description="Whether the worker is qualified to drive."
            />
          </FormControl>
          <FormControl className="pb-2">
            <InputField
              control={control}
              name="profile.disqualificationReason"
              label="Disqualification Reason"
              placeholder="Disqualification Reason"
              description="Reason for disqualification (if applicable)."
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="TWIC Credentials"
        description="TWIC credentials information for the worker."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="profile.twicCardNumber"
              label="TWIC Card Number"
              placeholder="TWIC Card Number"
              description="Transportation Worker Identification Credential number."
              maxLength={50}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.twicExpiry"
              label="TWIC Expiry"
              description="Expiration date of TWIC card."
              placeholder="TWIC Expiry"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Exemptions & Availability"
        description="Exemptions and availability information for the worker."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.eldExempt"
              label="ELD Exempt"
              description="Whether the worker is exempt from ELD requirements."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="profile.shortHaulExempt"
              label="Short Haul Exempt"
              description="Whether the worker qualifies for short-haul exemption."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="availableForDispatch"
              label="Available for Dispatch"
              description="Whether the worker can be dispatched."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="canBeAssigned"
              label="Can Be Assigned"
              description="Whether the worker can be assigned to equipment."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
