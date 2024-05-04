import { useCustomMutation } from "@/hooks/useCustomMutation";
import { timezoneChoices, type TimezoneChoices } from "@/lib/choices";
import { postUserProfilePicture } from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import type { QueryKeys, StatusChoiceProps } from "@/types";
import type { User, UserFormValues } from "@/types/accounts";
import { faUpload } from "@fortawesome/pro-duotone-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
import { Image } from "@unpic/react";
import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import * as yup from "yup";
import { InputField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import { Button } from "../ui/button";

function AvatarUploader() {
  const fileInputRef = useRef<HTMLInputElement>(null);
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
      <Button
        icon={faUpload}
        iconPlacement="left"
        variant="expandIcon"
        className="mr-2"
        size="sm"
        type="button"
        onClick={handleClick}
      >
        Change Avatar
      </Button>
      <Button size="sm" type="button" variant="outline">
        Remove
      </Button>
      <div className="flex gap-x-2">
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
    </div>
  );
}

export default function PersonalInformation({ user }: { user: User }) {
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
    additionalInvalidateQueries: ["authenticatedUser"],
    closeModal: false,
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
        <p className="text-muted-foreground text-xs">
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
    </>
  );
}
