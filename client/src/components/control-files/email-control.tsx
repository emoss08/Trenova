/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { SelectInput } from "@/components/common/fields/select-input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useEmailControl, useEmailProfiles } from "@/hooks/useQueries";
import { emailControlSchema } from "@/lib/validations/OrganizationSchema";
import {
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
      path: `/email_control/${emailControl.id}/`,
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
              name="billingEmailProfile"
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
              name="rateExpirationEmailProfile"
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
      <div className="4 flex items-center justify-end border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
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
  const { emailControlData, isError, isLoading } = useEmailControl();
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
          <ErrorLoadingData message="Failed to load dispatch control." />
        </div>
      ) : (
        emailControlData && <EmailControlForm emailControl={emailControlData} />
      )}
    </div>
  );
}
