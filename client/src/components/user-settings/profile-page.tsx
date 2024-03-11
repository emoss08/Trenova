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

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { timezoneChoices, TimezoneChoices } from "@/lib/choices";
import { QueryKeyWithParams } from "@/types";
import { User } from "@/types/accounts";
import { faPaperPlane } from "@fortawesome/pro-solid-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import * as yup from "yup";
import { InputField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import { Button } from "../ui/button";
import { Separator } from "../ui/separator";
import { InternalLink } from "@/components/ui/link";

function PersonalInformation({ user }: { user: User }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const avatarSrc =
    user.thumbnailUrl || `https://avatar.vercel.sh/${user.email}`;

  type UserSettingFormValues = {
    email: string;
    timezone: TimezoneChoices;
    name: string;
  };

  const schema: yup.ObjectSchema<UserSettingFormValues> = yup.object().shape({
    email: yup
      .string()
      .email("Please enter a valid email address")
      .required("Please enter your email address"),
    timezone: yup
      .string<TimezoneChoices>()
      .required("Please select your timezone"),
    name: yup.string().required("Please enter your last name"),
  });

  const { handleSubmit, control, reset } = useForm<UserSettingFormValues>({
    resolver: yupResolver(schema),
    defaultValues: {
      email: user.email,
      timezone: user.timezone,
      name: user.name,
    },
  });

  const mutation = useCustomMutation<UserSettingFormValues>(
    control,
    {
      method: "PATCH",
      path: `/users/${user.id}/`,
      successMessage: "User profile updated successfully.",
      queryKeysToInvalidate: ["users", user.id] as QueryKeyWithParams<
        "users",
        [string]
      >,
      closeModal: true,
      errorMessage: "Failed to update user profile.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: UserSettingFormValues) => {
    setIsSubmitting(true);
    reset(values);
    mutation.mutate(values);
  };

  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">
            Manage Your User Profile
          </h1>
          <p className="text-sm text-muted-foreground">
            Update your personal information here. Rest assured, your privacy is
            our priority. For more details, read our{" "}
            <InternalLink to="#">Privacy Policy</InternalLink>.
          </p>
        </div>
        <Separator />
      </div>
      <div className="mt-6 grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 md:grid-cols-3">
        <div>
          <h2 className="text-base font-semibold leading-7 text-foreground">
            Personal Information
          </h2>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">
            Provide accurate personal details to ensure seamless communication
            and service delivery.
          </p>
        </div>

        <form className="md:col-span-2" onSubmit={handleSubmit(onSubmit)}>
          <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
            <div className="col-span-full flex items-center gap-x-8">
              <img
                src={avatarSrc}
                alt="User Avatar"
                className="size-24 flex-none rounded-lg bg-muted-foreground object-cover"
              />
              <div>
                <Button size="sm" type="button">
                  Change Avatar
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
                label="Full Name"
                rules={{ required: true }}
                placeholder="First Name"
                description="Your first name as it should appear in your profile and communications."
              />
            </div>

            <div className="col-span-full">
              <InputField
                control={control}
                name="email"
                label="Email Address"
                rules={{ required: true }}
                placeholder="Email Address"
                description="Your primary email address for account notifications and correspondence."
              />
            </div>

            <div className="col-span-full">
              <SelectInput
                name="timezone"
                control={control}
                options={timezoneChoices}
                rules={{ required: true }}
                label="Timezone"
                placeholder="Timezone"
                description="Select the timezone that corresponds to your primary location. This helps in scheduling and localizing interactions."
              />
            </div>
          </div>
          <div className="mt-8 flex">
            <Button
              type="submit"
              variant="expandIcon"
              icon={faPaperPlane}
              isLoading={isSubmitting}
              iconPlacement="right"
            >
              Save Changes
            </Button>
          </div>
        </form>
      </div>
    </>
  );
}

function ChangePasswordForm() {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  type ChangePasswordFormValues = {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  };

  const schema: yup.ObjectSchema<ChangePasswordFormValues> = yup
    .object()
    .shape({
      oldPassword: yup
        .string()
        .required("Please enter your current password to continue"),
      newPassword: yup.string().required("Please enter a new password"),
      confirmPassword: yup
        .string()
        .oneOf([yup.ref("newPassword"), undefined], "Passwords must match")
        .required("Please confirm your new password"),
    });

  const { handleSubmit, control } = useForm<ChangePasswordFormValues>({
    resolver: yupResolver(schema),
  });

  const onSubmit = (values: ChangePasswordFormValues) => {
    setIsSubmitting(true);
  };

  return (
    <div className="grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 py-16 sm:px-6 md:grid-cols-3 lg:px-8">
      <div>
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Change password
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Update your password associated with your account.
        </p>
      </div>

      <form className="md:col-span-2" onSubmit={handleSubmit(onSubmit)}>
        <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
          <div className="col-span-full">
            <InputField
              control={control}
              type="password"
              name="oldPassword"
              label="Current Password"
              rules={{ required: true }}
              placeholder="Current Password"
              description="Enter the password you are currently using. This is required to verify your identity and secure your account."
            />
          </div>

          <div className="col-span-full">
            <InputField
              control={control}
              type="password"
              name="newPassword"
              label="New Password"
              rules={{ required: true }}
              placeholder="New Password"
              description="Create a new password that you haven't previously used. Ensure it is strong and secure, ideally a mix of letters, numbers, and special characters."
            />
          </div>

          <div className="col-span-full">
            <InputField
              control={control}
              type="password"
              name="confirmPassword"
              label="Confirm Password"
              rules={{ required: true }}
              placeholder="Confirm Password"
              description="Re-enter your new password to confirm it. This helps ensure that you haven't mistyped your new password."
            />
          </div>
        </div>

        <div className="mt-8 flex">
          <Button type="submit" isLoading={isSubmitting}>
            Change Password
          </Button>
        </div>
      </form>
    </div>
  );
}

export default function UserProfilePage({ user }: { user: User }) {
  return (
    <>
      <PersonalInformation user={user} />
      <ChangePasswordForm />
    </>
  );
}
