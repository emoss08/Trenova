import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@trenova/shared/services/auth";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthCardBody } from "./auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { StepCrumbs, StepHeading } from "./auth-primitives";

export const MIN_PASSWORD_LENGTH = 8;

// Mirrors the server's rule. The server is the one that enforces it; this exists so a
// too-short password is refused before it burns the single-use link.
export const resetPasswordSchema = z
  .object({
    newPassword: z
      .string()
      .min(MIN_PASSWORD_LENGTH, `Password must be at least ${MIN_PASSWORD_LENGTH} characters`),
    confirmPassword: z.string().min(1, "Confirm your new password"),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords do not match",
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
    <AuthCardBody>
      <StepCrumbs left="Account recovery" right="Choose a password" />
      <StepHeading title={t("Choose a new password")}>
        {t("This link works once. Pick a password you have not used here before.")}
      </StepHeading>

      <form
        className="mt-4 flex flex-col gap-3.5"
        noValidate
        onSubmit={handleSubmit((data) => void mutateAsync(data))}
      >
        <AuthTextField
          name="newPassword"
          control={control}
          label={t("New password")}
          type="password"
          required
          revealable
          placeholder={t("At least 8 characters")}
          autoComplete="new-password"
          disabled={isPending}
        />
        <AuthTextField
          name="confirmPassword"
          control={control}
          label={t("Confirm new password")}
          type="password"
          required
          revealable
          placeholder="••••••••"
          autoComplete="new-password"
          disabled={isPending}
        />
        {rootError && (
          <>
            <AuthErrorText>{rootError}</AuthErrorText>
            <button
              type="button"
              onClick={onRequestNewLink}
              className="text-muted-foreground hover:text-foreground cursor-pointer text-left text-[11.5px] underline underline-offset-[3px] transition-colors duration-150"
            >
              {t("Request a new link")}
            </button>
          </>
        )}
        <AuthSubmit type="submit" isLoading={isPending} loadingText={t("Setting password")}>
          {t("Set new password")}
        </AuthSubmit>
      </form>
    </AuthCardBody>
  );
}

function InvalidLink({ onRequestNewLink }: { onRequestNewLink: () => void }) {
  const t = useT();

  return (
    <AuthCardBody>
      <StepCrumbs left="Account recovery" right="Link problem" />
      <StepHeading title={t("This link is incomplete")}>
        {t(
          "The reset link is missing its token. Some mail clients wrap long links across lines — copy the whole thing, or request a new one.",
        )}
      </StepHeading>

      <div className="mt-4">
        <AuthSubmit onClick={onRequestNewLink}>{t("Request a new link")}</AuthSubmit>
      </div>
    </AuthCardBody>
  );
}

export function ResetPasswordDone({ onSignIn }: { onSignIn: () => void }) {
  const t = useT();

  return (
    <AuthCardBody>
      <StepCrumbs left="Account recovery" right="Done" />
      <StepHeading title={t("Password changed")}>
        {t("Your new password is active. Any other sessions on this account have been signed out.")}
      </StepHeading>

      <div className="mt-4">
        <AuthSubmit onClick={onSignIn}>{t("Sign in")}</AuthSubmit>
      </div>
    </AuthCardBody>
  );
}
