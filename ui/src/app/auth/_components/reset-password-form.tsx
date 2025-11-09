import { Button } from "@/components/ui/button";
import { Form, FormControl } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import {
  resetPasswordSchema,
  ResetPasswordSchema,
} from "@/lib/schemas/auth-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
type ResetPasswordFormProps = {
  onBack: () => void;
  email: string;
};

export function ResetPasswordForm({ onBack, email }: ResetPasswordFormProps) {
  const mutation = useMutation({
    mutationFn: (values: ResetPasswordSchema) =>
      api.auth.resetPassword(values.emailAddress),
  });

  const {
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<ResetPasswordSchema>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: {
      emailAddress: email,
    },
  });

  async function onSubmit(data: ResetPasswordSchema) {
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
    <Form className="flex flex-col gap-y-2" onSubmit={handleSubmit(onSubmit)}>
      <FormControl className="min-h-[3em] rounded-md border border-orange-400 bg-orange-500/20 p-2">
        <div className="flex flex-col gap-1">
          <Label>Email address</Label>
          <p className="text-sm text-muted-foreground">{email}</p>
        </div>
      </FormControl>
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
