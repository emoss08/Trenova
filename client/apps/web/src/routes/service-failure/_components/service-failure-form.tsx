import { useT } from "@trenova/shared/i18n/use-t";
import { ServiceFailureReasonCodeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { ServiceFailureUpdate } from "@/types/service-failure";
import type { ServiceFailureReasonCodeAppliesTo } from "@/types/service-failure-reason-code";
import type { StopType } from "@trenova/shared/types/shipment";
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
  const t = useT();

  const { control } = useFormContext<ServiceFailureUpdate>();
  const appliesTo = reasonCodeAppliesToForStop(stopType);

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <ServiceFailureReasonCodeAutocompleteField
            control={control}
            name="reasonCodeId"
            label={t("Reason code")}
            placeholder={t("Select reason code")}
            extraSearchParams={appliesTo ? { appliesTo } : undefined}
            clearable
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="clearReasonCode"
            label={t("Clear reason code")}
            description={t(
              "Removes the assigned reason code while preserving the service failure record.",
            )}
            outlined
            position="left"
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="notes"
            label={t("Operations notes")}
            placeholder={t("Customer-facing operational context")}
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="internalNotes"
            label={t("Internal notes")}
            placeholder={t("Internal review notes")}
            disabled={disabled}
          />
        </FormControl>
      </FormGroup>

      <FormSection
        title={t("EDI overrides")}
        description={t("Overrides apply only to this failure.")}
      >
        <FormGroup cols={3}>
          <FormControl>
            <InputField
              control={control}
              name="x12StatusCodeOverride"
              label={t("Status code")}
              placeholder={t("SD")}
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="x12ReasonCodeOverride"
              label={t("Reason code")}
              placeholder={t("NS")}
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="x12ExceptionCode"
              label={t("Exception code")}
              placeholder={t("A3")}
              maxLength={3}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
