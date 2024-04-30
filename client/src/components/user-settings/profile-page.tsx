import { InternalLink } from "@/components/ui/link";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { timezoneChoices, TimezoneChoices } from "@/lib/choices";
import { postUserProfilePicture } from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import { QueryKeys, StatusChoiceProps } from "@/types";
import { User, UserFormValues } from "@/types/accounts";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
import { Image } from "@unpic/react";
import React from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import * as yup from "yup";
import { InputField, PasswordField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import { Button } from "../ui/button";
import { Separator } from "../ui/separator";

function AvatarUploader() {
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const [, setUserDetails] = useUserStore.use("user");
  const queryClient = useQueryClient();

  // Handle file change event
  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      const file = files[0];
      toast.promise(postUserProfilePicture(file), {
        loading: "Uploading your profile picture...",
        success: (data) => {
          setUserDetails((prev) => ({
            ...prev,
            profilePicUrl: data.profilePicUrl,
          }));
          queryClient.invalidateQueries({
            queryKey: ["currentUser"] as QueryKeys,
          });
          return "Profile picture uploaded successfully.";
        },
        error: "Failed to upload profile picture.",
      });
    }
  };

  // Function to trigger file input
  const handleClick = () => {
    if (fileInputRef.current) {
      // Check if the ref is not null
      fileInputRef.current.click();
    }
  };

  return (
    <div>
      <Button size="sm" type="button" onClick={handleClick}>
        Change Avatar
      </Button>
      <input
        ref={fileInputRef}
        type="file"
        accept=".jpg, .gif, .png"
        style={{ display: "none" }}
        onChange={handleFileChange}
      />
      <p className="text-muted-foreground mt-2 text-xs leading-5">
        JPG, GIF or PNG. Max size 1MB.
      </p>
    </div>
  );
}

function PersonalInformation({ user }: { user: User }) {
  const avatarSrc = `https://avatar.vercel.sh/${user.email}`;

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
    isAdmin: yup.boolean().required("Please select your role"),
    isSuperAdmin: yup.boolean().required("Please select your role"),
    phoneNumber: yup.string().optional(),
  });

  const { handleSubmit, control, reset } = useForm<UserFormValues>({
    resolver: yupResolver(schema),
    defaultValues: user,
  });

  const mutation = useCustomMutation<UserFormValues>(control, {
    method: "PUT",
    path: `/users/${user.id}/`,
    successMessage: "User profile updated successfully.",
    queryKeysToInvalidate: ["users"],
    additionalInvalidateQueries: ["currentUser"],
    closeModal: false,
    errorMessage: "Failed to update user profile.",
  });

  const onSubmit = (values: UserFormValues) => {
    reset(values);
    mutation.mutate(values);
  };

  React.useEffect(() => {
    if (mutation.isSuccess) {
      reset(user);
    }
  }, [mutation.isSuccess, reset, user]);

  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-foreground text-2xl font-semibold">
            Manage Your User Profile
          </h1>
          <p className="text-muted-foreground text-sm">
            Update your personal information here. Rest assured, your privacy is
            our priority. For more details, read our{" "}
            <InternalLink to="#">Privacy Policy</InternalLink>.
          </p>
        </div>
        <Separator />
      </div>
      <div className="mt-6 grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 md:grid-cols-3">
        <div>
          <h2 className="text-foreground text-base font-semibold leading-7">
            Personal Information
          </h2>
          <p className="text-muted-foreground mt-1 text-sm leading-6">
            Provide accurate personal details to ensure seamless communication
            and service delivery.
          </p>
        </div>

        <form className="md:col-span-2" onSubmit={handleSubmit(onSubmit)}>
          <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
            <div className="col-span-full flex items-center gap-x-8">
              <Image
                src={user?.profilePicUrl || avatarSrc}
                layout="constrained"
                alt="User Avatar"
                className="bg-muted-foreground size-24 flex-none rounded-lg object-cover"
                width={96}
                height={96}
              />
              <AvatarUploader />
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
            <Button type="submit" isLoading={mutation.isPending}>
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

  const { handleSubmit, control, reset } = useForm<ChangePasswordFormValues>({
    resolver: yupResolver(schema),
    defaultValues: {
      oldPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  const mutation = useCustomMutation<ChangePasswordFormValues>(
    control,
    {
      method: "POST",
      path: "/users/change-password",
      successMessage: "Password updated successfully.",
      errorMessage: "Failed to update password.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: ChangePasswordFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <div className="grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 py-16 sm:px-6 md:grid-cols-3 lg:px-8">
      <div>
        <h2 className="text-foreground text-base font-semibold leading-7">
          Change password
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Update your password associated with your account.
        </p>
      </div>

      <form className="md:col-span-2" onSubmit={handleSubmit(onSubmit)}>
        <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
          <div className="col-span-full">
            <PasswordField
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
            <PasswordField
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
            <PasswordField
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
