/*
 * COPYRIGHT(c) 2024 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useUSStates } from "@/hooks/useQueries";
import { timezoneChoices } from "@/lib/constants";
import { Organization, OrganizationFormValues } from "@/types/organization";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";

function OrganizationForm({ organization }: { organization: Organization }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const {
    selectUSStates,
    isLoading: isStatesLoading,
    isError: isStateError,
  } = useUSStates();

  const { control, handleSubmit, reset } = useForm<OrganizationFormValues>({
    defaultValues: organization,
  });

  const mutation = useCustomMutation<OrganizationFormValues>(
    control,
    {
      method: "PUT",
      path: `/organizations/${organization.id}/`,
      successMessage: "Organization updated successfully.",
      queryKeysToInvalidate: ["userOrganization"],
      errorMessage: "Failed to update organization.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: OrganizationFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">
            Manage Your Organization Information
          </h1>
          <p className="text-sm text-muted-foreground">
            Keep your organization's profile up to date. This section allows you
            to maintain accurate information about your organization, which is
            crucial for efficient management, communication, and service
            delivery within the transportation management system. We are
            committed to safeguarding your data. For more information on our
            data handling practices, please review our Privacy Policy.
          </p>
        </div>
        <Separator />
      </div>
      <div className="grid grid-cols-1 gap-8 pt-10 md:grid-cols-3">
        <div className="px-4 sm:px-0">
          <h2 className="text-base font-semibold leading-7 text-foreground">
            General Information
          </h2>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">
            Essential for operational efficiency and compliance, this section
            captures your organization's core details. Accurate information
            ensures effective communication and customized service in the
            transportation sector.
          </p>
        </div>

        <form
          className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
          onSubmit={handleSubmit(onSubmit)}
        >
          <div className="px-4 py-6 sm:p-8">
            <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
              <div className="col-span-full flex items-center gap-x-8">
                <Avatar className="h-24 w-24 flex-none rounded-lg">
                  <AvatarImage src={organization.logo} />
                  <AvatarFallback className="h-24 w-24 flex-none rounded-lg">
                    {organization.scacCode}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <Button
                    size="sm"
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      console.log("Change Logo");
                    }}
                  >
                    Change Logo
                  </Button>
                  <p className="mt-2 text-xs leading-5 text-muted-foreground">
                    JPG, GIF or PNG. 1MB max.
                  </p>
                </div>
              </div>
              <div className="sm:col-span-full">
                <InputField
                  control={control}
                  name="name"
                  label="Name"
                  rules={{ required: true }}
                  placeholder="Name"
                  description="Enter the official name of your organization. This name will be used in all system communications and documents."
                />
              </div>

              <div className="col-span-full">
                <SelectInput
                  name="orgType"
                  control={control}
                  options={[
                    { label: "Asset", value: "ASSET" },
                    { label: "Brokerage", value: "BROKERAGE" },
                    { label: "Both", value: "BOTH" },
                  ]}
                  rules={{ required: true }}
                  disabled
                  label="Organization Type"
                  placeholder="Organization Type"
                  description="Specify the nature of your organization's operations. Choose 'Asset' for asset-based operations, 'Brokerage' for brokerage services, or 'Both' if your organization handles both operations."
                />
              </div>

              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="scacCode"
                  label="SCAC Code"
                  rules={{ required: true }}
                  placeholder="SCAC Code"
                  description="Input your Standard Carrier Alpha Code. This unique code is vital for identifying your organization in transport-related activities."
                />
              </div>

              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="dotNumber"
                  label="DOT Number"
                  placeholder="DOT Number"
                  description="Provide your Department of Transportation number. This unique identifier is critical for legal and regulatory compliance in transportation."
                />
              </div>

              <div className="col-span-full">
                <InputField
                  control={control}
                  name="addressLine1"
                  label="Address Line 1"
                  rules={{ required: true }}
                  placeholder="Address Line 1"
                  description="Specify the primary street address of your organization. This address is used for official correspondence and location-based services."
                />
              </div>

              <div className="col-span-full">
                <InputField
                  control={control}
                  name="addressLine2"
                  label="Address Line 2"
                  placeholder="Address Line 2"
                  description="Additional address information (if needed). Include suite numbers, building names, or other pertinent details."
                />
              </div>

              <div className="sm:col-span-2 sm:col-start-1">
                <InputField
                  control={control}
                  name="city"
                  rules={{ required: true }}
                  label="City"
                  placeholder="City"
                  description="Enter the city where your organization is based."
                />
              </div>

              <div className="sm:col-span-2">
                <SelectInput
                  name="state"
                  control={control}
                  options={selectUSStates}
                  isLoading={isStatesLoading}
                  isFetchError={isStateError}
                  rules={{ required: true }}
                  label="State"
                  placeholder="State"
                  description="Select the state in which your organization operates."
                />
              </div>

              <div className="sm:col-span-2">
                <InputField
                  control={control}
                  name="zipCode"
                  rules={{ required: true }}
                  label="Zip Code"
                  placeholder="Zip Code"
                  description="Input the zip code of your organization's primary location."
                />
              </div>
              <div className="sm:col-span-full">
                <InputField
                  control={control}
                  name="phoneNumber"
                  label="Phone Number"
                  placeholder="Phone Number"
                  description="Your organization's primary contact number. This will be used for official communications and urgent contacts."
                />
              </div>
              <div className="sm:col-span-full">
                <InputField
                  control={control}
                  name="website"
                  label="Website"
                  placeholder="Website"
                  description="The official website of your organization. It will be referenced in your profile and used for digital communications."
                />
              </div>
              <div className="sm:col-span-3">
                <SelectInput
                  name="language"
                  control={control}
                  options={[
                    { label: "English", value: "en" },
                    { label: "Spanish", value: "es" },
                  ]}
                  rules={{ required: true }}
                  label="Language"
                  placeholder="Language"
                  description="Choose the primary language used in your organization. This facilitates effective communication and system localization."
                />
              </div>
              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="currency"
                  label="Currency"
                  rules={{ required: true }}
                  disabled
                  placeholder="Currency"
                  description="Specify the currency used for your financial transactions. This ensures accurate financial management within the system."
                />
              </div>
              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="dateFormat"
                  label="Date Format"
                  rules={{ required: true }}
                  disabled
                  placeholder="Date Format"
                  description="Select the date format preferred by your organization. This setting will apply to all date references in the system."
                />
              </div>
              <div className="col-span-3">
                <InputField
                  control={control}
                  name="timeFormat"
                  label="Time Format"
                  rules={{ required: true }}
                  disabled
                  placeholder="Time Format"
                  description="Choose the time format for your organization's operations. This affects how time is displayed across the system's interfaces."
                />
              </div>
              <div className="col-span-3">
                <SelectInput
                  name="timezone"
                  control={control}
                  options={timezoneChoices}
                  rules={{ required: true }}
                  label="Timezone"
                  placeholder="Timezone"
                  description="Set the timezone corresponding to your organization’s primary operation area. Accurate timezone settings are essential for synchronized scheduling and coordination in transportation activities."
                />
              </div>
            </div>
          </div>
          <div className="flex items-center justify-end gap-x-6 border-t border-gray-900/10 p-4 sm:px-8">
            <Button
              onClick={(e) => {
                e.preventDefault();
                console.log("cancel");
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
      </div>
    </>
  );
}

export default function GeneralPage({
  organization,
}: {
  organization: Organization;
}) {
  return (
    <>
      <OrganizationForm organization={organization} />
    </>
  );
}
