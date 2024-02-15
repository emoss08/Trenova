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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useUSStates } from "@/hooks/useQueries";
import { timezoneChoices } from "@/lib/constants";
import { organizationSchema } from "@/lib/validations/OrganizationSchema";
import { Organization, OrganizationFormValues } from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function OrganizationForm({ organization }: { organization: Organization }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.generalpage", "common"]);

  const {
    selectUSStates,
    isLoading: isStatesLoading,
    isError: isStateError,
  } = useUSStates();

  const { control, handleSubmit, reset } = useForm<OrganizationFormValues>({
    resolver: yupResolver(organizationSchema),
    defaultValues: {
      name: organization.name,
      orgType: organization.orgType,
      scacCode: organization.scacCode,
      dotNumber: organization?.dotNumber || undefined,
      addressLine1: organization.addressLine1,
      addressLine2: organization.addressLine2,
      city: organization.city,
      state: organization.state,
      zipCode: organization.zipCode,
      phoneNumber: organization.phoneNumber,
      website: organization.website,
      language: organization.language,
      currency: organization.currency,
      dateFormat: organization.dateFormat,
      timeFormat: organization.timeFormat,
      timezone: organization.timezone,
    },
  });

  const mutation = useCustomMutation<OrganizationFormValues>(
    control,
    {
      method: "PUT",
      path: `/organizations/${organization.id}/`,
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["userOrganization"],
      errorMessage: t("formErrorMessage"),
    },
    () => setIsSubmitting(false),
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
          <h1 className="text-foreground text-2xl font-semibold">
            {t("title")}
          </h1>
          <p className="text-muted-foreground text-sm">{t("subTitle")}</p>
        </div>
        <Separator />
      </div>
      <div className="grid grid-cols-1 gap-8 pt-10 md:grid-cols-3">
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
              <div className="col-span-full flex items-center gap-x-8">
                <Avatar className="size-24 flex-none rounded-lg">
                  <AvatarImage src={organization.logo || ""} />
                  <AvatarFallback className="size-24 flex-none rounded-lg">
                    {organization.scacCode}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <Button
                    size="sm"
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                    }}
                  >
                    {t("fields.logo.placeholder")}
                  </Button>
                  <p className="text-muted-foreground mt-2 text-xs leading-5">
                    {t("fields.logo.description")}
                  </p>
                </div>
              </div>
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
                    { label: "Asset", value: "ASSET" },
                    { label: "Brokerage", value: "BROKERAGE" },
                    { label: "Both", value: "BOTH" },
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
                <InputField
                  control={control}
                  name="addressLine1"
                  label={t("fields.addressLine1.label")}
                  rules={{ required: true }}
                  placeholder={t("fields.addressLine1.placeholder")}
                  description={t("fields.addressLine1.description")}
                />
              </div>

              <div className="col-span-full">
                <InputField
                  control={control}
                  name="addressLine2"
                  label={t("fields.addressLine2.label")}
                  placeholder={t("fields.addressLine2.placeholder")}
                  description={t("fields.addressLine2.description")}
                />
              </div>

              <div className="sm:col-span-2 sm:col-start-1">
                <InputField
                  control={control}
                  name="city"
                  rules={{ required: true }}
                  label={t("fields.city.label")}
                  placeholder={t("fields.city.placeholder")}
                  description={t("fields.city.description")}
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
                  label={t("fields.state.label")}
                  placeholder={t("fields.state.placeholder")}
                  description={t("fields.state.description")}
                />
              </div>

              <div className="sm:col-span-2">
                <InputField
                  control={control}
                  name="zipCode"
                  rules={{ required: true }}
                  label={t("fields.zipCode.label")}
                  placeholder={t("fields.zipCode.placeholder")}
                  description={t("fields.zipCode.description")}
                />
              </div>
              <div className="sm:col-span-full">
                <InputField
                  control={control}
                  name="phoneNumber"
                  label={t("fields.phoneNumber.label")}
                  placeholder={t("fields.phoneNumber.placeholder")}
                  description={t("fields.phoneNumber.description")}
                />
              </div>
              <div className="sm:col-span-full">
                <InputField
                  control={control}
                  name="website"
                  label={t("fields.website.label")}
                  placeholder={t("fields.website.placeholder")}
                  description={t("fields.website.description")}
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
                  label={t("fields.language.label")}
                  placeholder={t("fields.language.placeholder")}
                  description={t("fields.language.description")}
                />
              </div>
              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="currency"
                  rules={{ required: true }}
                  readOnly
                  label={t("fields.currency.label")}
                  placeholder={t("fields.currency.placeholder")}
                  description={t("fields.currency.description")}
                />
              </div>
              <div className="sm:col-span-3">
                <InputField
                  control={control}
                  name="dateFormat"
                  rules={{ required: true }}
                  readOnly
                  label={t("fields.dateFormat.label")}
                  placeholder={t("fields.dateFormat.placeholder")}
                  description={t("fields.dateFormat.description")}
                />
              </div>
              <div className="col-span-3">
                <InputField
                  control={control}
                  name="timeFormat"
                  rules={{ required: true }}
                  readOnly
                  label={t("fields.timeFormat.label")}
                  placeholder={t("fields.timeFormat.placeholder")}
                  description={t("fields.timeFormat.description")}
                />
              </div>
              <div className="col-span-3">
                <SelectInput
                  name="timezone"
                  control={control}
                  options={timezoneChoices}
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
              variant="ghost"
              disabled={isSubmitting}
            >
              {t("buttons.cancel", { ns: "common" })}
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
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
  return (
    <>
      <OrganizationForm organization={organization} />
    </>
  );
}
