import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { TRACKING_FIELDS, trackingFieldLabel } from "@/lib/customer-tracking";
import {
  updateCustomerTrackingProfile,
  type CustomerTrackingProfile,
} from "@/lib/graphql/customer-tracking";
import { queries } from "@/lib/queries";
import {
  customerTrackingSettingsSchema,
  type CustomerTrackingSettings,
} from "@/types/customer-tracking";
import { zodResolver } from "@hookform/resolvers/zod";
import { Link01Icon, Mail01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

function settingsOf(profile: CustomerTrackingProfile): CustomerTrackingSettings {
  return {
    linkEnabled: profile.linkEnabled,
    visibleFields: profile.visibleFields,
    linkTtlDays: profile.linkTtlDays,
    notifyPickedUp: profile.notifyPickedUp,
    notifyDelayed: profile.notifyDelayed,
    notifyDelivered: profile.notifyDelivered,
    notifyPodReady: profile.notifyPodReady,
    includeLink: profile.includeLink,
    recipients: profile.recipients ?? "",
  };
}

/**
 * What a customer sees through a shipment tracking link and which status emails
 * it is sent. It saves on its own, apart from the customer record, because it
 * sits inside the customer form and a nested form cannot submit.
 */
export function CustomerTrackingSettingsTab({ customerId }: { customerId: string }) {
  const profileQuery = useQuery(queries.customer.trackingProfile(customerId));

  if (profileQuery.isLoading || !profileQuery.data) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-40" />
        <Skeleton className="h-40" />
      </div>
    );
  }
  return (
    <CustomerTrackingSettingsForm
      key={profileQuery.data.version}
      customerId={customerId}
      profile={profileQuery.data}
    />
  );
}

function CustomerTrackingSettingsForm({
  customerId,
  profile,
}: {
  customerId: string;
  profile: CustomerTrackingProfile;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canEdit } = usePermission(Resource.Customer, Operation.Update);
  const defaultValues = useMemo(() => settingsOf(profile), [profile]);
  const form = useForm<CustomerTrackingSettings>({
    resolver: zodResolver(customerTrackingSettingsSchema),
    defaultValues,
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isDirty, isSubmitting },
  } = form;

  const fieldOptions = useMemo(
    () => TRACKING_FIELDS.map((field) => ({ value: field, label: trackingFieldLabel(t, field) })),
    [t],
  );

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: CustomerTrackingSettings) =>
      updateCustomerTrackingProfile({
        customerId,
        ...values,
        expectedVersion: profile.id ? profile.version : null,
      }),
    resourceName: "Customer tracking settings",
    form,
    onSuccess: (saved) => {
      queryClient.setQueryData(queries.customer.trackingProfile(customerId).queryKey, saved);
      reset(settingsOf(saved));
      toast.success(t("Tracking settings saved"));
    },
  });

  const save = handleSubmit((values) => mutateAsync(values));

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={t("Tracking link")} icon={<Link01Icon />}>
        <FormGroup cols={1} className="p-3">
          <FormControl>
            <SwitchField
              control={control}
              name="linkEnabled"
              label={t("Allow tracking links")}
              description={t(
                "Turning this off stops every link already shared for this customer's shipments.",
              )}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <MultiCheckboxField
              control={control}
              name="visibleFields"
              label={t("The page shows")}
              description={t(
                "Every stop's city and state, and the load number, are always shown. Holds and comments appear only when marked for the customer.",
              )}
              options={fieldOptions}
              emptyAs="array"
            />
          </FormControl>
          <FormControl className="max-w-[300px]">
            <NumberField
              control={control}
              name="linkTtlDays"
              label={t("A link lasts")}
              sideText="days"
              min={1}
              max={180}
            />
          </FormControl>
        </FormGroup>
      </SectionPanel>

      <SectionPanel title={t("Status emails")} icon={<Mail01Icon />}>
        <FormGroup cols={1} className="p-3">
          <p className="text-muted-foreground text-xs">
            {t(
              "Each event is emailed once per shipment, or once per stop for a delay. Turning a rule on never sends emails about events that already happened.",
            )}
          </p>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyPickedUp"
              label={t("Picked up")}
              description={t("When the load leaves its first pickup.")}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyDelayed"
              label={t("Delayed")}
              description={t(
                "When a stop's projected arrival falls behind its appointment, with the new arrival time.",
              )}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyDelivered"
              label={t("Delivered")}
              description={t("When the load leaves its last delivery.")}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="notifyPodReady"
              label={t("Proof of delivery ready")}
              description={t("When the proof of delivery is filed on the shipment.")}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="includeLink"
              label={t("Include a tracking link in each email")}
              readOnly={!canEdit}
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="recipients"
              label={t("Send to")}
              description={t(
                "Separate addresses with commas. Leave empty to use the customer's status update recipients, then the To line of its email profile.",
              )}
              placeholder={t("shipping@customer.com, logistics@customer.com")}
              disabled={!canEdit}
            />
          </FormControl>
        </FormGroup>
      </SectionPanel>

      {canEdit && (
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={!isDirty || isSubmitting}
            onClick={() => reset(defaultValues)}
          >
            {t("Discard")}
          </Button>
          <Button
            type="button"
            disabled={!isDirty}
            isLoading={isSubmitting}
            loadingText={t("Saving...")}
            onClick={() => void save()}
          >
            {t("Save tracking settings")}
          </Button>
        </div>
      )}
    </div>
  );
}
