import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveButton } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { timeFormatChoices } from "@/lib/choices";
import type { UserSchema } from "@/lib/schemas/user-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { api } from "@/services/api";
import { useAuthActions } from "@/stores/user-store";
import type { APIError } from "@/types/errors";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import * as z from "zod";

// Schema for profile update - only editable fields
const profileUpdateSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .regex(
      /^[a-zA-Z]+(\s[a-zA-Z]+)*$/,
      "Name can only contain letters and spaces",
    ),
  username: z
    .string()
    .min(1, "Username is required")
    .max(20, "Username must be less than 20 characters")
    .regex(/^[a-zA-Z0-9]+$/, "Username must be alphanumeric"),
  emailAddress: z.string().email("Invalid email address"),
  timezone: z.string().min(1, "Timezone is required"),
  timeFormat: z.enum(TimeFormat),
});

type ProfileUpdateSchema = z.infer<typeof profileUpdateSchema>;

interface UserProfileFormProps {
  user: UserSchema;
  onSuccess?: () => void;
}

export function UserProfileForm({ user, onSuccess }: UserProfileFormProps) {
  const { setUser } = useAuthActions();

  const mutation = useMutation({
    mutationFn: async (values: ProfileUpdateSchema) =>
      await api.user.updateMe(values),
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting, isDirty },
  } = useForm<ProfileUpdateSchema>({
    resolver: zodResolver(profileUpdateSchema),
    defaultValues: {
      name: user.name,
      username: user.username,
      emailAddress: user.emailAddress,
      timezone: user.timezone,
      timeFormat: user.timeFormat,
    },
  });

  async function onSubmit(values: ProfileUpdateSchema) {
    try {
      const updatedUser = await mutation.mutateAsync(values);
      setUser(updatedUser);
      toast.success("Profile updated successfully");
      onSuccess?.();
    } catch (error) {
      const err = error as APIError;
      if (err.isValidationError()) {
        err.getFieldErrors().forEach((fieldError) => {
          const fieldName = fieldError.name as keyof ProfileUpdateSchema;
          setError(fieldName, {
            message: fieldError.reason,
          });
        });
      } else {
        toast.error(err.data?.detail || "Failed to update profile");
      }
    }
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup className="gap-4" cols={1}>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label="Full Name"
            placeholder="Enter your full name"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="username"
            label="Username"
            placeholder="Enter your username"
            rules={{ required: true }}
            maxLength={20}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailAddress"
            label="Email Address"
            type="email"
            placeholder="Enter your email address"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="timezone"
            label="Timezone"
            options={TIMEZONES}
            placeholder="Select timezone"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="timeFormat"
            label="Time Format"
            options={timeFormatChoices}
            placeholder="Select time format"
            rules={{ required: true }}
          />
        </FormControl>
        <FormSaveButton
          size="lg"
          type="submit"
          title="Save changes"
          text="Save changes"
          isSubmitting={isSubmitting}
          disabled={isSubmitting || !isDirty}
        />
      </FormGroup>
    </Form>
  );
}
