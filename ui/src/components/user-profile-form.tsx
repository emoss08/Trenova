import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveButton } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { timeFormatChoices } from "@/lib/choices";
import {
  updateMeSchema,
  UpdateMeSchema,
  type UserSchema,
} from "@/lib/schemas/user-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { api } from "@/services/api";
import { useAuthActions } from "@/stores/user-store";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export function UserProfileForm({
  user,
  onSuccess,
}: {
  user: UserSchema;
  onSuccess?: () => void;
}) {
  const {
    control,
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitSuccessful, isSubmitting, isDirty },
  } = useForm<UpdateMeSchema>({
    resolver: zodResolver(updateMeSchema),
    defaultValues: user,
  });

  const { setUser } = useAuthActions();

  const mutation = useApiMutation({
    mutationFn: async (values: UpdateMeSchema) =>
      await api.user.updateMe(values),
    onSuccess: (data) => {
      setUser(data);
      toast.success("Profile updated successfully");
      onSuccess?.();
    },
    resourceName: "User",
    setFormError: setError,
  });

  useEffect(() => {
    if (isSubmitSuccessful) {
      reset(user);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isSubmitSuccessful, user]);

  console.log(errors);

  const onSubmit = useCallback(
    async (values: UpdateMeSchema) => {
      await mutation.mutateAsync(values);
    },
    [mutation],
  );

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <div className="flex flex-col px-4 py-2">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Full Name"
              placeholder="Enter your full name"
              rules={{ required: true }}
              description="The name you want to be displayed in the system"
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
              description="The username you want to use to login to the system"
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
              description="The email address you want to use to receive notifications and other communications from the system"
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
              description="The timezone you want to use to display the time in the system"
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
              description="The time format you want to use to display the time in the system"
            />
          </FormControl>
        </FormGroup>
      </div>
      <div className="flex justify-end border-t px-4 py-2">
        <FormSaveButton
          size="lg"
          type="submit"
          title="Save changes"
          text="Save changes"
          isSubmitting={isSubmitting}
          disabled={isSubmitting || !isDirty}
        />
      </div>
    </Form>
  );
}
