import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { JurisdictionRule } from "@/types/jurisdiction-rule";
import { GlobeIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";

const STATUS_CHOICES = [
  { value: "Active", label: "Active" },
  { value: "Inactive", label: "Inactive" },
  { value: "Draft", label: "Draft" },
];

export function JurisdictionRuleForm() {
  const { control } = useFormContext<JurisdictionRule>();

  return (
    <div className="flex flex-col gap-4">
      {/* This is the one thing an editor most needs to know and would not
          otherwise see: the row is not theirs. */}
      <Alert>
        <GlobeIcon className="size-4" />
        <AlertDescription>
          These limits are shared by every organization on the platform, not just yours. To hold
          your fleet to something stricter, record a carrier override instead. Changing any limit
          below clears the verification on this rule.
        </AlertDescription>
      </Alert>

      <FormSection title="Jurisdiction" description="Which state these limits apply to">
        <FormGroup cols={2}>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label="State"
              placeholder="Select state"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              description="Inactive and draft rules are not used by the permit engine."
              options={STATUS_CHOICES}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Legal Limits"
        description="A load exceeding any of these needs a permit in this state"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="maxWidthFeet"
              label="Max Width"
              sideText="ft"
              placeholder="8.5"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxHeightFeet"
              label="Max Height"
              sideText="ft"
              placeholder="13.5"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxLengthFeet"
              label="Max Length"
              sideText="ft"
              placeholder="53"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxWeightPounds"
              label="Max Weight"
              sideText="lbs"
              thousandSeparator
              placeholder="80000"
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Superload Thresholds"
        description="Above these a load needs superload review rather than an ordinary permit. Leave blank if unknown."
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="superloadWidthFeet"
              label="Superload Width"
              sideText="ft"
              placeholder="Leave blank if unknown"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="superloadWeightPounds"
              label="Superload Weight"
              sideText="lbs"
              thousandSeparator
              placeholder="Leave blank if unknown"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Permit Terms"
        description="How long a permit takes to obtain and how long it lasts"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="permitLeadTimeDays"
              label="Lead Time"
              sideText="days"
              description="Quoting a pickup sooner than this is a missed appointment."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="permitValidityDays"
              label="Validity"
              sideText="days"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <MoneyField control={control} name="permitBaseFee" label="Base Fee" />
          </FormControl>
          <FormControl>
            <MoneyField control={control} name="permitPerMileFee" label="Per Mile Fee" />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Travel Restrictions"
        description="When an oversize load may legally move in this state"
      >
        <FormGroup cols={2}>
          <FormControl>
            <CheckboxField
              control={control}
              name="daylightOnly"
              label="Daylight Only"
              description="Movement confined to daylight hours"
            />
          </FormControl>
          <FormControl>
            <CheckboxField
              control={control}
              name="rushHourRestricted"
              label="Rush Hour Restricted"
              description="Barred through metro areas at peak times"
            />
          </FormControl>
          <FormControl>
            <CheckboxField
              control={control}
              name="weekendRestricted"
              label="Weekend Restricted"
              description="Movement restricted at weekends"
            />
          </FormControl>
          <FormControl>
            <CheckboxField
              control={control}
              name="holidayRestricted"
              label="Holiday Restricted"
              description="Movement restricted on public holidays"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Source"
        description="What these numbers were taken from. Editing only this keeps the rule's verification."
      >
        <FormGroup cols={1}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="sourceNote"
              label="Source Note"
              placeholder="Which statute or permit office publication these limits came from"
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="sourceUrl"
              label="Source URL"
              placeholder="https://"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
