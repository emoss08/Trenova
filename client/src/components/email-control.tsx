import { SelectInput } from "@/components/common/fields/select-input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useEmailControl, useEmailProfiles } from "@/hooks/useQueries";
import { emailControlSchema } from "@/lib/validations/OrganizationSchema";
import type {
  EmailControlFormValues,
  EmailControl as EmailControlType,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function EmailControlForm({
  emailControl,
}: {
  emailControl: EmailControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { selectEmailProfile, isLoading, isError } = useEmailProfiles();

  const { control, handleSubmit, reset } = useForm<EmailControlFormValues>({
    resolver: yupResolver(emailControlSchema),
    defaultValues: emailControl,
  });

  const mutation = useCustomMutation<EmailControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/email-control/${emailControl.id}/`,
      successMessage: "Email Control updated successfully.",
      queryKeysToInvalidate: ["emailControl"],
      errorMessage: "Failed to update email control.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: EmailControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="m-4 border border-border bg-card sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="billingEmailProfileId"
              control={control}
              options={selectEmailProfile}
              isLoading={isLoading}
              isFetchError={isError}
              rules={{ required: true }}
              label="Billing Email Profile"
              placeholder="Billing Email Profile"
              description="Select the email profile for sending billing-related emails."
              hasPopoutWindow
              popoutLink="/admin/email-profiles/"
              popoutLinkLabel="Email Profile"
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="rateExpirtationEmailProfileId"
              control={control}
              options={selectEmailProfile}
              isLoading={isLoading}
              isFetchError={isError}
              rules={{ required: true }}
              label="Rate Expiration Email Profile"
              placeholder="Rate Expiration Email Profile"
              description="Choose the email profile for sending rate expiration notifications."
              hasPopoutWindow
              popoutLink="/admin/email-profiles/"
              popoutLinkLabel="Email Profile"
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-4 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function EmailControl() {
  const { data, isError, isLoading } = useEmailControl();
  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Email Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Manage and streamline your organization's email communications with
          our Email Control Panel. This tool facilitates the customization of
          email profiles for various operational needs, ensuring consistent and
          professional communication for billing, rate notifications, and more.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="m-4 bg-background p-8 ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <ErrorLoadingData />
        </div>
      ) : (
        data && <EmailControlForm emailControl={data} />
      )}
    </div>
  );
}
