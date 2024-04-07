import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { emailProtocolChoices } from "@/lib/choices";
import { emailProfileSchema } from "@/lib/validations/OrganizationSchema";
import { type EmailProfileFormValues as FormValues } from "@/types/organization";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { CheckboxInput } from "./common/fields/checkbox";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

export function EmailProfileForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { t } = useTranslation("admin.emailprofile");

  return (
    <Form>
      <FormGroup className="lg:grid-cols-2">
        <FormControl>
          <CheckboxInput
            name="isDefault"
            control={control}
            className="mt-4"
            rules={{
              required: true,
            }}
            label={t("fields.defaultProfile.label")}
            description={t("fields.defaultProfile.description")}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="name"
            control={control}
            rules={{
              required: true,
            }}
            label={t("fields.name.label")}
            placeholder={t("fields.name.placeholder")}
            description={t("fields.name.description")}
          />
        </FormControl>
      </FormGroup>
      <FormGroup className="lg:grid-cols-1">
        <FormControl>
          <InputField
            name="email"
            control={control}
            rules={{
              required: true,
            }}
            label={t("fields.email.label")}
            placeholder={t("fields.email.placeholder")}
            description={t("fields.email.description")}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="protocol"
            options={emailProtocolChoices}
            control={control}
            label={t("fields.protocol.label")}
            placeholder={t("fields.protocol.placeholder")}
            description={t("fields.protocol.description")}
          />
        </FormControl>
      </FormGroup>
      <FormGroup className="lg:grid-cols-2">
        <FormControl>
          <InputField
            name="host"
            control={control}
            label={t("fields.host.label")}
            placeholder={t("fields.host.placeholder")}
            description={t("fields.host.description")}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="port"
            control={control}
            label={t("fields.port.label")}
            placeholder={t("fields.port.placeholder")}
            description={t("fields.port.description")}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="username"
            control={control}
            label={t("fields.username.label")}
            placeholder={t("fields.username.placeholder")}
            description={t("fields.username.description")}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="password"
            control={control}
            type="password"
            autoComplete="new-password"
            label={t("fields.password.label")}
            placeholder={t("fields.password.placeholder")}
            description={t("fields.password.description")}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function EmailProfileDialog({ onOpenChange, open }: TableSheetProps) {
  const { t } = useTranslation(["admin.emailprofile", "common"]);

  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(emailProfileSchema),
    defaultValues: {
      name: "",
      email: "",
      protocol: "UNENCRYPTED",
      host: "",
      port: undefined,
      username: "",
      password: "",
      isDefault: false,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/email-profiles/",
      successMessage: t("formMessages.postSuccess"),
      queryKeysToInvalidate: ["email-profile-table-data"],
      closeModal: true,
      errorMessage: t("formMessages.postError"),
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{t("title")}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>{t("subTitle")} </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <EmailProfileForm control={control} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={isSubmitting}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
