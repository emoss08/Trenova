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

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TIMEZONES, TimezoneChoices } from "@/lib/timezone";
import {
  clearProfilePicture,
  postUserProfilePicture,
} from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import type { StatusChoiceProps } from "@/types";
import type { User, UserFormValues } from "@/types/accounts";
import { yupResolver } from "@hookform/resolvers/yup";
import { Image } from "@unpic/react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import * as yup from "yup";
import { InputField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import { ImageUploader } from "../ui/avatar";
import { Button } from "../ui/button";

export default function PersonalInformation({ user }: { user: User }) {
  const avatarSrc = `https://avatar.vercel.sh/${user.email}`;
  const [, setUserDetails] = useUserStore.use("user");

  const schema: yup.ObjectSchema<UserFormValues> = yup.object().shape({
    status: yup
      .string<StatusChoiceProps>()
      .required("Please select your status"),
    username: yup.string().required("Please enter your username"),
    name: yup.string().required("Please enter your full name"),
    email: yup
      .string()
      .email("Please enter a valid email address")
      .required("Please enter your email address"),
    timezone: yup
      .string<TimezoneChoices>()
      .required("Please select your timezone"),
    isAdmin: yup.boolean().required("Please select your admin status"),
    phoneNumber: yup.string().optional(),
  });

  const { handleSubmit, control, reset, formState } = useForm<UserFormValues>({
    resolver: yupResolver(schema),
    defaultValues: user,
  });

  console.info("Formstate errors", formState.errors);

  const mutation = useCustomMutation<UserFormValues>(control, {
    method: "PUT",
    path: `/users/${user.id}/`,
    successMessage: "User profile updated successfully.",
    closeModal: false,
    reset,
    errorMessage: "Failed to update user profile.",
  });

  const onSubmit = (values: UserFormValues) => {
    useUserStore.set("user", values as User);
    mutation.mutate(values);
  };

  useEffect(() => {
    if (mutation.isSuccess) {
      reset(user);
    }
  }, [mutation.isSuccess, reset, user]);

  return (
    <>
      <div className="sticky top-0 z-20 mb-6 flex items-center gap-x-2">
        <h2 className="shrink-0 text-sm" id="personal-information">
          Personal Information
        </h2>
        <p className="text-xs text-muted-foreground">
          Update your personal information to keep your profile up-to-date.
        </p>
      </div>
      <form className="md:col-span-2" onSubmit={handleSubmit(onSubmit)}>
        <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
          <div className="col-span-full flex items-center gap-x-8">
            <Image
              src={user?.profilePicUrl || avatarSrc}
              layout="constrained"
              alt="User Avatar"
              className="size-24 flex-none rounded-lg bg-muted-foreground object-cover"
              width={96}
              height={96}
            />
            <ImageUploader
              callback={postUserProfilePicture}
              successCallback={(data: User) => {
                setUserDetails((prev) => ({
                  ...prev,
                  profilePicUrl: data.profilePicUrl,
                }));
                return "Profile picture uploaded successfully.";
              }}
              removeFileCallback={clearProfilePicture}
              removeSuccessCallback={() => {
                setUserDetails((prev) => ({
                  ...prev,
                  profilePicUrl: "",
                }));
                return "Profile picture removed successfully.";
              }}
            />
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
              options={TIMEZONES}
              rules={{ required: true }}
              label="Timezone"
              placeholder="Timezone"
              description="Select the timezone that corresponds to your primary location. This helps in scheduling and localizing interactions."
            />
          </div>
        </div>
        <div className="mt-8 flex justify-end gap-x-2 border-border">
          <Button variant="outline" type="button">
            Cancel
          </Button>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </div>
      </form>
    </>
  );
}
