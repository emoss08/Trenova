/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { SelectInput } from "@/components/common/fields/select-input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useEmailControl, useEmailProfiles } from "@/hooks/useQueries";
import { emailControlSchema } from "@/lib/validations/OrganizationSchema";
import type {
  EmailControlFormValues,
  EmailControl as EmailControlType,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ComponentLoader } from "./ui/component-loader";

function EmailControlForm({
  emailControl,
}: {
  emailControl: EmailControlType;
}) {
  const { selectEmailProfile, isLoading, isError } = useEmailProfiles();

  const { control, handleSubmit, reset } = useForm<EmailControlFormValues>({
    resolver: yupResolver(emailControlSchema),
    defaultValues: emailControl,
  });

  const mutation = useCustomMutation<EmailControlFormValues>(control, {
    method: "PUT",
    path: `/email-control/${emailControl.id}/`,
    successMessage: "Email Control updated successfully.",
    queryKeysToInvalidate: "emailControl",
    reset,
    errorMessage: "Failed to update email control.",
  });

  const onSubmit = (values: EmailControlFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
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
      <div className="border-muted flex items-center justify-end gap-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={mutation.isPending}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
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
        <h2 className="text-foreground text-base font-semibold leading-7">
          Email Control
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Manage and streamline your organization's email communications with
          our Email Control Panel. This tool facilitates the customization of
          email profiles for various operational needs, ensuring consistent and
          professional communication for billing, rate notifications, and more.
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <ComponentLoader className="h-[30em]" />
        </div>
      ) : isError ? (
        <ErrorLoadingData />
      ) : (
        data && <EmailControlForm emailControl={data} />
      )}
    </div>
  );
}
