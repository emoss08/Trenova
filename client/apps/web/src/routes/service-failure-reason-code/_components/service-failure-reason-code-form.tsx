import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  serviceFailureReasonCategoryChoices,
  serviceFailureReasonCodeAppliesToChoices,
} from "@/lib/choices";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";
import { useFormContext } from "react-hook-form";

export function ServiceFailureReasonCodeForm({
  disabled,
}: {
  disabled?: boolean;
}) {
  const { control } = useFormContext<ServiceFailureReasonCode>();

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="active"
            label="Active"
            description="Controls whether this reason is available for detected and manual service failures."
            outlined
            position="left"
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="code"
            label="Reason Code"
            placeholder="LATE_DELIVERY"
            rules={{ required: true }}
            maxLength={64}
            description="Stable identifier used for reporting, audit, and integrations."
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="label"
            label="Display Name"
            placeholder="Late Delivery"
            rules={{ required: true }}
            maxLength={120}
            description="Name shown to operations and billing users."
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="category"
            label="Category"
            placeholder="Select Category"
            options={serviceFailureReasonCategoryChoices}
            rules={{ required: true }}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="appliesTo"
            label="Applies To"
            placeholder="Select Stop Type"
            options={serviceFailureReasonCodeAppliesToChoices}
            rules={{ required: true }}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Details"
            placeholder="When should this reason be used?"
            disabled={disabled}
          />
        </FormControl>
      </FormGroup>

      <FormSection
        title="EDI Defaults"
        description="Defaults used when building a service failure EDI 214 payload."
      >
        <FormGroup cols={3}>
          <FormControl>
            <InputField
              control={control}
              name="defaultStatusCode"
              label="Status Code"
              placeholder="SD"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="defaultReasonCode"
              label="Reason Code"
              placeholder="NS"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="defaultExceptionCode"
              label="Exception Code"
              placeholder="A3"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="defaultNote"
              label="Default Note"
              placeholder="Default note applied to detected failures"
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title="Ordering" description="Lower sort values appear first.">
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="sortOrder"
              label="Sort Order"
              min={0}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
