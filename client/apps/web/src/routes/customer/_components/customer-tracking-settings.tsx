import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { TRACKING_FIELDS, trackingFieldLabel } from "@/lib/customer-tracking";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Customer } from "@trenova/shared/types/customer";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * What a customer sees through a shipment tracking link and which status emails
 * it is sent. Part of the customer form: the panel's Save writes it with the
 * rest of the customer.
 */
export function CustomerTrackingSettingsForm() {
  const t = useT();
  const { control } = useFormContext<Customer>();
  const linkEnabled = useWatch({ control, name: "trackingProfile.linkEnabled" });

  const fieldOptions = TRACKING_FIELDS.map((field) => ({
    value: field,
    label: trackingFieldLabel(t, field),
  }));

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("Tracking link")}
        description={t("What this customer sees when you share a shipment's tracking page.")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.linkEnabled"
              label={t("Allow tracking links")}
              description={t(
                "Turning this off stops every link already shared for this customer's shipments.",
              )}
              outlined
            />
          </FormControl>
          <FormControl>
            <MultiCheckboxField
              control={control}
              name="trackingProfile.visibleFields"
              label={t("The page shows")}
              description={t(
                "Every stop's city and state, and the load number, are always shown. Holds and comments appear only when marked for the customer.",
              )}
              options={fieldOptions}
              emptyAs="array"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="trackingProfile.linkTtlDays"
              label={t("A link lasts")}
              sideText="days"
              min={1}
              max={180}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Status emails")}
        description={t(
          "Each event is emailed once per shipment, or once per stop for a delay. Turning a rule on never sends emails about events that already happened.",
        )}
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.notifyPickedUp"
              label={t("Picked up")}
              description={t("When the load leaves its first pickup.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.notifyDelayed"
              label={t("Delayed")}
              description={t(
                "When a stop's projected arrival falls behind its appointment, with the new arrival time.",
              )}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.notifyDelivered"
              label={t("Delivered")}
              description={t("When the load leaves its last delivery.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.notifyPodReady"
              label={t("Proof of delivery ready")}
              description={t("When the proof of delivery is filed on the shipment.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="trackingProfile.includeLink"
              label={t("Include a tracking link in each email")}
              description={
                linkEnabled === false ? t("Allow tracking links to include one.") : undefined
              }
              disabled={linkEnabled === false}
              outlined
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="trackingProfile.recipients"
              label={t("Send to")}
              description={t(
                "Separate addresses with commas. Leave empty to use the customer's status update recipients, then the To line of its email profile.",
              )}
              placeholder={t("shipping@customer.com, logistics@customer.com")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
