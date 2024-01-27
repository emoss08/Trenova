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

import { EmailProfileForm } from "@/components/email-profile/email-profile-table-dialog";
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
import { formatDate } from "@/lib/date";
import { emailProfileSchema } from "@/lib/validations/OrganizationSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  EmailProfile,
  EmailProfileFormValues as FormValues,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function EmailProfileEditForm({
  emailProfile,
}: {
  emailProfile: EmailProfile;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(emailProfileSchema),
    defaultValues: {
      name: emailProfile.name,
      email: emailProfile.email,
      host: emailProfile.host,
      port: emailProfile.port || undefined,
      username: emailProfile.username,
      password: emailProfile.password,
      protocol: emailProfile.protocol,
      defaultProfile: emailProfile.defaultProfile,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/email_profiles/${emailProfile.id}/`,
      successMessage: "Email Profile updated successfully.",
      queryKeysToInvalidate: ["email-profile-table-data"],
      closeModal: true,
      errorMessage: "Failed to update email profile.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <EmailProfileForm control={control} />
      <DialogFooter className="mt-6">
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function EmailProfileTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [emailProfile] = useTableStore.use("currentRecord");

  if (!emailProfile) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{emailProfile.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {emailProfile && formatDate(emailProfile.modified)}
        </DialogDescription>
        {emailProfile && <EmailProfileEditForm emailProfile={emailProfile} />}
      </DialogContent>
    </Dialog>
  );
}
