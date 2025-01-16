import { resetPassword } from "@/services/auth";
import { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { InputField } from "../../../components/fields/input-field";
import { Button } from "../../../components/ui/button";
import { Form, FormControl, FormGroup } from "../../../components/ui/form";
type ResetPasswordFormProps = {
  onBack: () => void;
  email: string;
};

const resetPasswordSchema = z.object({
  emailAddress: z
    .string()
    .min(1, "Email is required")
    .email("Invalid email address"),
});

type ResetPasswordFormValues = z.infer<typeof resetPasswordSchema>;

export function ResetPasswordForm({ onBack, email }: ResetPasswordFormProps) {
  const mutation = useMutation({
    mutationFn: (values: ResetPasswordFormValues) =>
      resetPassword(values.emailAddress),
  });

  const {
    control,
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: {
      emailAddress: email,
    },
  });

  async function onSubmit(data: ResetPasswordFormValues) {
    try {
      const result = await mutation.mutateAsync(data);
      if (result.data?.message) {
        toast.success("Password reset email sent. Check your inbox.");
      }
    } catch (error) {
      const err = error as APIError;
      if (err.isAuthorizationError()) {
        toast.error(err.data?.detail);
      }
    }
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl>
          <InputField
            rules={{ required: true }}
            label="Email address"
            placeholder="Enter your email address"
            control={control}
            name="emailAddress"
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-between gap-2">
        <Button variant="ghost" onClick={onBack}>
          Back to login
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          Reset password
        </Button>
      </div>
    </Form>
  );
}
