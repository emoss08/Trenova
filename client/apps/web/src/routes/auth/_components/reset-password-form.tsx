import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@trenova/shared/services/auth";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthErrorText, AuthPasswordField, AuthSubmit } from "./auth-field";
import { AuthHeading, AuthQuietButton, StepCrumbs } from "./auth-primitives";
import { AuthCheckMark, AuthSuccess } from "./auth-success";
import { translate } from "@trenova/shared/i18n/runtime";

export const MIN_PASSWORD_LENGTH = 8;

// Mirrors the server's rule. The server is the one that enforces it; this exists so a
// too-short password is refused before it burns the single-use link.
export const resetPasswordSchema = z
  .object({
    newPassword: z.string().min(MIN_PASSWORD_LENGTH, {
      error: () => translate("Password must be at least {0} characters", MIN_PASSWORD_LENGTH),
    }),
    confirmPassword: z.string().min(1, { error: () => translate("Confirm your new password") }),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    error: () => translate("Passwords do not match"),
    path: ["confirmPassword"],
  });

export type ResetPasswordRequest = z.infer<typeof resetPasswordSchema>;

export function ResetPasswordForm({
  token,
  onDone,
  onRequestNewLink,
}: {
  token: string;
  onDone: () => void;
  onRequestNewLink: () => void;
}) {
  const t = useT();

  const form = useForm<ResetPasswordRequest>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: { newPassword: "", confirmPassword: "" },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: (data: ResetPasswordRequest) => authService.resetPassword(token, data.newPassword),
    form,
    resourceName: "Password reset",
    onSuccess: onDone,
  });

  if (!token) {
    return <InvalidLink onRequestNewLink={onRequestNewLink} />;
  }

  return (
    <>
      <StepCrumbs left="Account recovery" right="Choose a password" />
      <AuthHeading title={t("Choose a new password")}>
        {t("This link works once. Pick a password you have not used here before.")}
      </AuthHeading>

      <form
        className="flex flex-col gap-4"
        noValidate
        onSubmit={handleSubmit((data) => void mutateAsync(data))}
      >
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
        {rootError && (
          <>
            <AuthErrorText>{rootError}</AuthErrorText>
            <div>
              <AuthQuietButton onClick={onRequestNewLink}>{t("Request a new link")}</AuthQuietButton>
            </div>
          </>
        )}
        <AuthSubmit type="submit" busy={isPending} busyLabel={t("Setting password")}>
          {t("Set new password")}
        </AuthSubmit>
      </form>
    </>
  );
}

function InvalidLink({ onRequestNewLink }: { onRequestNewLink: () => void }) {
  const t = useT();

  return (
    <>
      <StepCrumbs left="Account recovery" right="Link problem" />
      <AuthHeading title={t("This link is incomplete")}>
        {t(
          "The reset link is missing its token. Some mail clients wrap long links across lines — copy the whole thing, or request a new one.",
        )}
      </AuthHeading>

      <div className="mt-4">
        <AuthSubmit onClick={onRequestNewLink}>{t("Request a new link")}</AuthSubmit>
      </div>
    </>
  );
}

export function ResetPasswordDone({ onSignIn }: { onSignIn: () => void }) {
  const t = useT();

  return (
    <AuthSuccess
      mark={<AuthCheckMark />}
      title={t("Password changed")}
      lead={t(
        "Your new password is active. Any other sessions on this account have been signed out.",
      )}
      actions={<AuthSubmit onClick={onSignIn}>{t("Sign in")}</AuthSubmit>}
    />
  );
}
