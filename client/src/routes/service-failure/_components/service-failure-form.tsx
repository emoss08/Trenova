import { ServiceFailureReasonCodeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import type { ServiceFailureUpdate } from "@/types/service-failure";
import type { ServiceFailureReasonCodeAppliesTo } from "@/types/service-failure-reason-code";
import type { StopType } from "@/types/shipment";
import { useFormContext } from "react-hook-form";

type ServiceFailureFormProps = {
  disabled?: boolean;
  stopType?: StopType;
};

function reasonCodeAppliesToForStop(
  stopType?: StopType,
): ServiceFailureReasonCodeAppliesTo | undefined {
  if (stopType === "Pickup" || stopType === "SplitPickup") return "Pickup";
  if (stopType === "Delivery" || stopType === "SplitDelivery") return "Delivery";
  return undefined;
}

export function ServiceFailureForm({ disabled, stopType }: ServiceFailureFormProps) {
  const { control } = useFormContext<ServiceFailureUpdate>();
  const appliesTo = reasonCodeAppliesToForStop(stopType);

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <ServiceFailureReasonCodeAutocompleteField
            control={control}
            name="reasonCodeId"
            label="Reason Code"
            placeholder="Select Reason Code"
            extraSearchParams={appliesTo ? { appliesTo } : undefined}
            clearable
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="clearReasonCode"
            label="Clear Reason Code"
            description="Removes the assigned reason code while preserving the service failure record."
            outlined
            position="left"
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="notes"
            label="Operations Notes"
            placeholder="Customer-facing operational context"
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="internalNotes"
            label="Internal Notes"
            placeholder="Internal review notes"
            disabled={disabled}
          />
        </FormControl>
      </FormGroup>

      <FormSection title="EDI Overrides" description="Overrides apply only to this failure.">
        <FormGroup cols={3}>
          <FormControl>
            <InputField
              control={control}
              name="x12StatusCodeOverride"
              label="Status Code"
              placeholder="SD"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="x12ReasonCodeOverride"
              label="Reason Code"
              placeholder="NS"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="x12ExceptionCode"
              label="Exception Code"
              placeholder="A3"
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
