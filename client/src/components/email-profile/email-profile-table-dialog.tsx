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
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableSheetProps } from "@/types/tables";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { EmailProfileFormValues as FormValues } from "@/types/organization";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { SelectInput } from "@/components/common/fields/select-input";
import { emailProtocolChoices } from "@/lib/choices";
import { emailProfileSchema } from "@/lib/validations/OrganizationSchema";
import { yupResolver } from "@hookform/resolvers/yup";

export function EmailProfileForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  return (
    <Form>
      <FormGroup className="lg:grid-cols-1">
        <FormControl>
          <InputField
            name="name"
            control={control}
            label="Name"
            rules={{
              required: true,
            }}
            description="The name of the email profile."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="email"
            control={control}
            label="Email"
            rules={{
              required: true,
            }}
            description="The email address that will be used for outgoing emails."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="protocol"
            options={emailProtocolChoices}
            control={control}
            label="Protocol"
            description="The protocol that will be used to send emails."
          />
        </FormControl>
      </FormGroup>
      <FormGroup className="lg:grid-cols-2">
        <FormControl>
          <InputField
            name="host"
            control={control}
            label="Host"
            description="The host that will be used to send emails."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="port"
            control={control}
            label="Port"
            description="The port that will be used to send emails."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="username"
            control={control}
            label="Username"
            description="The username that will be used to send emails."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="password"
            control={control}
            label="Password"
            type="password"
            autoComplete="new-password"
            description="The password that will be used to send emails."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function EmailProfileDialog({ onOpenChange, open }: TableSheetProps) {
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
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/email_profiles/",
      successMessage: "Email Profile created successfully.",
      queryKeysToInvalidate: ["email-profile-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new email profile.",
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
          <DialogTitle>Create New Email Profile</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Email Profile.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <EmailProfileForm control={control} />
          <DialogFooter className="mt-6">
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
            >
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
