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
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { emailProtocolChoices } from "@/lib/choices";
import { emailProfileSchema } from "@/lib/validations/OrganizationSchema";
import { EmailProfileFormValues as FormValues } from "@/types/organization";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { CheckboxInput } from "../common/fields/checkbox";

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
            name="defaultProfile"
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
      defaultProfile: false,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/email_profiles/",
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>
        <DialogDescription>{t("subTitle")}</DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <EmailProfileForm control={control} />
          <DialogFooter className="mt-6">
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText={t("buttons.savingText", { ns: "common" })}
            >
              {t("buttons.save", { ns: "common" })}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
