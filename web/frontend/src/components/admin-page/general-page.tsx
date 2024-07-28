/**
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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useUserPermissions } from "@/context/user-permissions";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TIMEZONES } from "@/lib/timezone";
import { organizationSchema } from "@/lib/validations/OrganizationSchema";
import {
  clearOrganizationLogo,
  postOrganizationLogo,
} from "@/services/OrganizationRequestService";
import { QueryKeys } from "@/types";
import type {
  Organization,
  OrganizationFormValues,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  ImageUploader,
} from "../ui/avatar";

function OrganizationForm({ organization }: { organization: Organization }) {
  const { t } = useTranslation(["admin.generalpage", "common"]);
  const { userHasPermission } = useUserPermissions();
  const queryClient = useQueryClient();
  const [localLogoUrl, setLocalLogoUrl] = useState(organization.logoUrl);

  const { control, handleSubmit, reset, setValue } =
    useForm<OrganizationFormValues>({
      resolver: yupResolver(organizationSchema),
      defaultValues: { ...organization, logoUrl: localLogoUrl },
    });

  const mutation = useCustomMutation<OrganizationFormValues>(control, {
    method: "PUT",
    path: "/organizations/",
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "organization",
    reset,
    errorMessage: t("formErrorMessage"),
    onSettled: (response) => {
      reset(response?.data);
      setLocalLogoUrl(response?.data.logoUrl);
    },
  });

  const onSubmit = (values: OrganizationFormValues) => {
    mutation.mutate(values);
  };

  const handleLogoUpdate = async (file: File) => {
    try {
      const response = await postOrganizationLogo(file);
      setLocalLogoUrl(response.logoUrl);
      setValue("logoUrl", response.logoUrl);
      queryClient.invalidateQueries({
        queryKey: ["organization"] as QueryKeys,
      });
      return "Logo updated successfully.";
    } catch (error) {
      console.error("Error updating logo:", error);
      return "Failed to update logo.";
    }
  };

  const handleLogoRemoval = async () => {
    try {
      await clearOrganizationLogo();
      setLocalLogoUrl("");
      setValue("logoUrl", "");
      queryClient.invalidateQueries({
        queryKey: ["organization"] as QueryKeys,
      });
      return "Logo removed successfully.";
    } catch (error) {
      console.error("Error removing logo:", error);
      return "Failed to remove logo.";
    }
  };

  return (
    <>
      <div className="grid grid-cols-1 gap-8 md:grid-cols-3 xl:grid-cols-4">
        <div className="px-4 sm:px-0">
          <h2 className="text-foreground text-base font-semibold leading-7">
            {t("organizationDetails")}
          </h2>
          <p className="text-muted-foreground mt-1 text-sm leading-6">
            {t("organizationDetailsDescription")}
          </p>
        </div>
        <form
          className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
          onSubmit={handleSubmit(onSubmit)}
        >
          <div className="px-4 py-6 sm:p-8">
            <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
              {userHasPermission("organization:change_logo") && (
                <div className="col-span-full flex items-center gap-x-8">
                  <Avatar className="size-24 flex-none rounded-lg">
                    <AvatarImage src={organization.logoUrl || ""} />
                    <AvatarFallback className="size-24 flex-none rounded-lg">
                      {organization.scacCode}
                    </AvatarFallback>
                  </Avatar>
                  <ImageUploader
                    iconText="Change Logo"
                    callback={handleLogoUpdate}
                    successCallback={() => {
                      queryClient.invalidateQueries({
                        queryKey: ["organization"] as QueryKeys,
                      });

                      return "Logo updated successfully.";
                    }}
                    removeFileCallback={handleLogoRemoval}
                    removeSuccessCallback={() => {
                      queryClient.invalidateQueries({
                        queryKey: ["organization"] as QueryKeys,
                      });

                      return "Logo removed successfully.";
                    }}
                  />
                </div>
              )}
              <div className="col-span-full">
                <InputField
                  control={control}
                  name="name"
                  label={t("fields.name.label")}
                  rules={{ required: true }}
                  placeholder={t("fields.name.placeholder")}
                  description={t("fields.name.description")}
                />
              </div>
              <div className="col-span-full">
                <SelectInput
                  name="orgType"
                  control={control}
                  options={[
                    { label: "Asset", value: "Asset" },
                    { label: "Brokerage", value: "Brokerage" },
                    { label: "Both", value: "Both" },
                  ]}
                  rules={{ required: true }}
                  label={t("fields.orgType.label")}
                  placeholder={t("fields.orgType.placeholder")}
                  description={t("fields.orgType.description")}
                />
              </div>

              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="scacCode"
                  label={t("fields.scacCode.label")}
                  rules={{ required: true }}
                  placeholder={t("fields.scacCode.placeholder")}
                  description={t("fields.scacCode.description")}
                />
              </div>

              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="dotNumber"
                  label={t("fields.dotNumber.label")}
                  placeholder={t("fields.dotNumber.placeholder")}
                  description={t("fields.dotNumber.description")}
                />
              </div>

              <div className="col-span-full">
                <SelectInput
                  name="timezone"
                  control={control}
                  options={TIMEZONES}
                  rules={{ required: true }}
                  label={t("fields.timezone.label")}
                  placeholder={t("fields.timezone.placeholder")}
                  description={t("fields.timezone.description")}
                />
              </div>
            </div>
          </div>
          <div className="border-border flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
            <Button
              onClick={(e) => {
                e.preventDefault();
              }}
              type="button"
              variant="outline"
              disabled={mutation.isPending}
            >
              {t("buttons.cancel", { ns: "common" })}
            </Button>
            <Button type="submit" isLoading={mutation.isPending}>
              {t("buttons.save", { ns: "common" })}
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
  return <OrganizationForm organization={organization} />;
}
