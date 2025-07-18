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
      <FormGroup className="gap-1" cols={1}>
        <FormControl>
          <PasswordField
            control={control}
            name="currentPassword"
            label="Current password"
            placeholder="Enter your current password"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <PasswordField
            control={control}
            name="newPassword"
            label="New password"
            placeholder="Enter your new password"
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
          />
        </FormControl>
        <FormSaveButton
          size="lg"
          type="submit"
          title="Change password"
          text="Change password"
          isSubmitting={isSubmitting}
          disabled={isSubmitting}
        />
      </FormGroup>
    </Form>
  );
}

export default ChangePasswordForm;
