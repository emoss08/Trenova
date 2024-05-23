import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TIMEZONES } from "@/lib/timezone";
import { organizationSchema } from "@/lib/validations/OrganizationSchema";
import { postOrganizationLogo } from "@/services/OrganizationRequestService";
import { QueryKeys } from "@/types";
import type {
  Organization,
  OrganizationFormValues,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
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
  const queryClient = useQueryClient();

  const { control, handleSubmit, reset } = useForm<OrganizationFormValues>({
    resolver: yupResolver(organizationSchema),
    defaultValues: organization,
  });

  const mutation = useCustomMutation<OrganizationFormValues>(control, {
    method: "PUT",
    path: `/organizations/${organization.id}`,
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "userOrganization",
    reset,
    errorMessage: t("formErrorMessage"),
  });

  const onSubmit = (values: OrganizationFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <>
      <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
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
                  <AvatarImage src={organization.logoUrl || ""} />
                  <AvatarFallback className="size-24 flex-none rounded-lg">
                    {organization.scacCode}
                  </AvatarFallback>
                </Avatar>
                <ImageUploader
                  iconText="Change Logo"
                  callback={postOrganizationLogo}
                  successCallback={() => {
                    queryClient.invalidateQueries({
                      queryKey: ["userOrganization"] as QueryKeys,
                    });
                    return "Logo uploaded successfully.";
                  }}
                />
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
                    { label: "Asset", value: "A" },
                    { label: "Brokerage", value: "B" },
                    { label: "Both", value: "X" },
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
