import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ChangeMyPassword } from "@trenova/shared/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthErrorText, AuthPasswordField, AuthSubmit } from "./auth-field";
import { AuthHeading, StepCrumbs } from "./auth-primitives";
import { MIN_PASSWORD_LENGTH } from "./reset-password-form";
import { translate } from "@trenova/shared/i18n/runtime";

export const forcedChangePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, { error: () => translate("Enter your current password") }),
    newPassword: z.string().min(MIN_PASSWORD_LENGTH, {
      error: () => translate("Password must be at least {0} characters", MIN_PASSWORD_LENGTH),
    }),
    confirmPassword: z.string().min(1, { error: () => translate("Confirm your new password") }),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    error: () => translate("Passwords do not match"),
    path: ["confirmPassword"],
  })
  .refine((data) => data.newPassword !== data.currentPassword, {
    error: () => translate("Choose a password you are not already using"),
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
  const t = useT();

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
    <>
      <StepCrumbs left="Password required" right="Secure sign-in" />
      <AuthHeading title={t("Choose a new password")}>
        {t("Your account is set to require a password change before you can continue.")}
      </AuthHeading>

      <form
        className="flex flex-col gap-4"
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
        <AuthPasswordField
          name="currentPassword"
          control={control}
          label={t("Current password")}
          placeholder="••••••••"
          autoComplete="current-password"
          disabled={isPending}
        />
        <AuthPasswordField
          name="newPassword"
          control={control}
          label={t("New password")}
          placeholder={t("At least 8 characters")}
          autoComplete="new-password"
          disabled={isPending}
        />
        <AuthPasswordField
          name="confirmPassword"
          control={control}
          label={t("Confirm new password")}
          placeholder="••••••••"
          autoComplete="new-password"
          disabled={isPending}
        />
        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit type="submit" busy={isPending} busyLabel={t("Updating password")}>
          {t("Update password")}
        </AuthSubmit>
      </form>
    </>
  );
}
