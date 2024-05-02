import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { timezoneChoices } from "@/lib/choices";
import { organizationSchema } from "@/lib/validations/OrganizationSchema";
import type {
  Organization,
  OrganizationFormValues,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function OrganizationForm({ organization }: { organization: Organization }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.generalpage", "common"]);

  // const {
  //   selectUSStates,
  //   isLoading: isStatesLoading,
  //   isError: isStateError,
  // } = useUSStates();

  const { control, handleSubmit, reset } = useForm<OrganizationFormValues>({
    resolver: yupResolver(organizationSchema),
    defaultValues: {
      name: organization.name,
      orgType: organization.orgType,
      scacCode: organization.scacCode,
      dotNumber: organization?.dotNumber || undefined,
      timezone: organization.timezone,
    },
  });

  const mutation = useCustomMutation<OrganizationFormValues>(
    control,
    {
      method: "PUT",
      path: "/organization/",
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
              variant="outline"
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
  return <OrganizationForm organization={organization} />;
}
