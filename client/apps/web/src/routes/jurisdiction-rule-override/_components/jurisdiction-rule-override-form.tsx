import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";
import { ShieldIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";

export function JurisdictionRuleOverrideForm() {
  const { control } = useFormContext<JurisdictionRuleOverride>();

  return (
    <div className="flex flex-col gap-4">
      <Alert>
        <ShieldIcon className="size-4" />
        <AlertDescription>
          An override can only make a state limit stricter, never looser. Leave a field blank to use
          whatever the state requires. This applies to your organization alone.
        </AlertDescription>
      </Alert>

      <FormSection title="Jurisdiction" description="Which state this override applies to">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label="State"
              placeholder="Select state"
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Tighter Limits"
        description="Leave blank to defer to the state. A value above the state limit is rejected."
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="maxWidthFeet"
              label="Max Width"
              sideText="ft"
              placeholder="Defer to state"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxHeightFeet"
              label="Max Height"
              sideText="ft"
              placeholder="Defer to state"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxLengthFeet"
              label="Max Length"
              sideText="ft"
              placeholder="Defer to state"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxWeightPounds"
              label="Max Weight"
              sideText="lbs"
              thousandSeparator
              placeholder="Defer to state"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Lead Time"
        description="This one runs the other way: an override may require more notice than the state, never less."
      >
        <FormGroup cols={1}>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="permitLeadTimeDays"
              label="Permit Lead Time"
              sideText="days"
              placeholder="Defer to state"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Added Restrictions"
        description="Restrictions the state does not impose. A restriction the state does impose cannot be lifted here."
      >
        <FormGroup cols={2}>
          <FormControl>
            <CheckboxField
              control={control}
              name="daylightOnly"
              label="Daylight Only"
              description="We do not run oversize at night in this state"
            />
          </FormControl>
          <FormControl>
            <CheckboxField
              control={control}
              name="holidayRestricted"
              label="Holiday Restricted"
              description="We do not run oversize on holidays in this state"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title="Reason" description="Why this organization runs tighter than the statute">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              description="At least 10 characters. This is what explains the override to whoever reads it next."
              placeholder="Our trailer fleet and insurance terms are narrower than this state allows"
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
