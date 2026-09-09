import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ChangeMyPassword } from "@trenova/shared/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthCardBody } from "./auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { StepCrumbs, StepHeading } from "./auth-primitives";
import { MIN_PASSWORD_LENGTH } from "./reset-password-form";

export const forcedChangePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, "Enter your current password"),
    newPassword: z
      .string()
      .min(MIN_PASSWORD_LENGTH, `Password must be at least ${MIN_PASSWORD_LENGTH} characters`),
    confirmPassword: z.string().min(1, "Confirm your new password"),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })
  .refine((data) => data.newPassword !== data.currentPassword, {
    message: "Choose a password you are not already using",
    path: ["newPassword"],
  });

export type ForcedChangePasswordRequest = z.infer<typeof forcedChangePasswordSchema>;

/**
 * The step a session sees when its account is flagged to change its password — set by
 * an administrator, or on a newly provisioned account.
 *
 * The API refuses everything but this endpoint while the flag is set, so there is
 * nothing to skip to; the screen simply matches what the server already enforces.
 */
export function ChangePasswordForm({ onChanged }: { onChanged?: () => void }) {
  const setUser = useAuthStore((state) => state.setUser);

  const form = useForm<ForcedChangePasswordRequest>({
    resolver: zodResolver(forcedChangePasswordSchema),
    defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending } = useApiMutation({
    // Written as an arrow rather than a bound method reference: binding erases the
    // return type to unknown, which leaves setUser below with nothing to infer from.
    mutationFn: (data: ChangeMyPassword) => apiService.userService.changeMyPassword(data),
    form,
    resourceName: "Password",
    onSuccess: (user) => {
      // The response carries the user with the demand cleared, which is what takes the
      // gate down.
      setUser(user);
      onChanged?.();
    },
  });

  return (
    <AuthCardBody>
      <StepCrumbs left="Password required" right="Secure sign-in" />
      <StepHeading title="Choose a new password">
        Your account is set to require a password change before you can continue.
      </StepHeading>

      <form
        className="mt-4 flex flex-col gap-3.5"
        noValidate
        onSubmit={handleSubmit(
          (data) =>
            void mutateAsync({
              currentPassword: data.currentPassword,
              newPassword: data.newPassword,
              confirmPassword: data.confirmPassword,
            }),
        )}
      >
        <AuthTextField
          name="currentPassword"
          control={control}
          label="Current password"
          type="password"
          required
          revealable
          placeholder="••••••••"
          autoComplete="current-password"
          disabled={isPending}
        />
        <AuthTextField
          name="newPassword"
          control={control}
          label="New password"
          type="password"
          required
          revealable
          placeholder="At least 8 characters"
          autoComplete="new-password"
          disabled={isPending}
        />
        <AuthTextField
          name="confirmPassword"
          control={control}
          label="Confirm new password"
          type="password"
          required
          revealable
          placeholder="••••••••"
          autoComplete="new-password"
          disabled={isPending}
        />
        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit type="submit" isLoading={isPending} loadingText="Updating password">
          Update password
        </AuthSubmit>
      </form>
    </AuthCardBody>
  );
}
