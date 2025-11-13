import { PasswordField } from "@/components/fields/sensitive-input-field";
import { FormSaveButton } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import {
  changePasswordSchema,
  type ChangePasswordSchema,
} from "@/lib/schemas/auth-schema";
import { api } from "@/services/api";
import type { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export function ChangePasswordForm({
  onOpenChange,
}: {
  onOpenChange: (open: boolean) => void;
}) {
  const mutation = useMutation({
    mutationFn: async (values: ChangePasswordSchema) =>
      await api.user.changePassword(values),
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<ChangePasswordSchema>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      currentPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  async function onSubmit(values: ChangePasswordSchema) {
    try {
      await mutation.mutateAsync(values);
      toast.success("Password changed successfully");
      onOpenChange(false);
    } catch (error) {
      const err = error as APIError;
      if (err.isValidationError()) {
        err.getFieldErrors().forEach((fieldError) => {
          const fieldName = fieldError.name as keyof ChangePasswordSchema;
          setError(fieldName, {
            message: fieldError.reason,
          });
        });
      } else if (err.isAuthorizationError()) {
        toast.error(err.data?.detail);
      }
    }
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <div className="flex min-h-[368px] flex-col px-4 py-2">
        <FormGroup cols={1}>
          <FormControl>
            <PasswordField
              control={control}
              name="currentPassword"
              label="Current password"
              placeholder="Enter your current password"
              description="The current password you want to change"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <PasswordField
              control={control}
              name="newPassword"
              label="New password"
              placeholder="Enter your new password"
              description="The new password you want to set"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <PasswordField
              control={control}
              name="confirmPassword"
              label="Confirm password"
              placeholder="Confirm your new password"
              rules={{ required: true }}
              description="The confirm password you want to set"
            />
          </FormControl>
        </FormGroup>
      </div>
      <div className="flex justify-end border-t px-4 py-2">
        <FormSaveButton
          size="lg"
          type="submit"
          title="Change password"
          text="Change password"
          isSubmitting={isSubmitting}
          disabled={isSubmitting}
        />
      </div>
    </Form>
  );
}

export default ChangePasswordForm;
