import { useApiMutation } from "@/hooks/use-api-mutation";
import { emailSchema } from "@/lib/auth-validation";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowLeftIcon } from "@trenova/shared/components/icons";
import { useRichT } from "@trenova/shared/i18n/rich";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { authService } from "@trenova/shared/services/auth";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { AuthBackButton, AuthHeading, AuthQuietButton } from "./auth-primitives";
import { AuthHint, AuthMailMark, AuthSuccess } from "./auth-success";
import { useAuthStage } from "./stage/auth-stage-context";

export const forgotPasswordSchema = z.object({
  emailAddress: emailSchema(() => translate("Please enter a valid email address.")),
});

export type ForgotPasswordRequest = z.infer<typeof forgotPasswordSchema>;

/**
 * Step 1 of recovery: ask for the address.
 *
 * The confirmation this shows on success is deliberately non-committal — "if that
 * address has an account". The server answers the same way for an address that
 * belongs to nobody as for one that does, and a screen that said "sent!" only for
 * real accounts would give away exactly what the server refuses to.
 */
export function ForgotPasswordForm({
  defaultEmail,
  onBack,
}: {
  defaultEmail?: string;
  onBack: () => void;
}) {
  const t = useT();
  const stage = useAuthStage();

  const form = useForm<ForgotPasswordRequest>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { emailAddress: defaultEmail ?? "" },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutate, isPending, isSuccess, variables, reset } = useApiMutation({
    mutationFn: (data: ForgotPasswordRequest) => authService.forgotPassword(data.emailAddress),
    form,
    resourceName: "Password reset",
  });

  if (isSuccess && variables) {
    return (
      <ForgotPasswordSent emailAddress={variables.emailAddress} onBack={onBack} onRetry={reset} />
    );
  }

  return (
    <>
      <AuthBackButton onClick={onBack}>{t("Back to sign in")}</AuthBackButton>
      <AuthHeading title={t("Reset your password")}>
        {t(
          "Enter the address you sign in with and we'll send you a link to choose a new password.",
        )}
      </AuthHeading>

      <form
        className="flex flex-col gap-4"
        noValidate
        onSubmit={handleSubmit((data) => {
          stage.burst();
          mutate(data);
        })}
      >
        <AuthTextField
          name="emailAddress"
          control={control}
          label={t("Email")}
          type="email"
          placeholder="you@carrier.com"
          autoComplete="username"
          autoFocus
          disabled={isPending}
        />
        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit type="submit" busy={isPending} busyLabel={t("Sending link")}>
          {t("Send reset link")}
        </AuthSubmit>
      </form>
    </>
  );
}

function ForgotPasswordSent({
  emailAddress,
  onBack,
  onRetry,
}: {
  emailAddress: string;
  onBack: () => void;
  onRetry: () => void;
}) {
  const t = useT();
  const rt = useRichT();

  return (
    <AuthSuccess
      mark={<AuthMailMark />}
      title={t("Check your inbox")}
      lead={rt(
        "If <b>{0}</b> has an account, a reset link is on its way. It works once and expires shortly, so use it soon.",
        {
          b: (content) => (
            <b className="text-foreground font-medium [overflow-wrap:anywhere]">{content}</b>
          ),
        },
        emailAddress,
      )}
      actions={
        <>
          <AuthSubmit onClick={onBack} leadingIcon={ArrowLeftIcon} trailingIcon={null}>
            {t("Back to sign in")}
          </AuthSubmit>
          <AuthQuietButton onClick={onRetry}>{t("Use a different address")}</AuthQuietButton>
        </>
      }
    >
      <AuthHint>
        {t(
          "Nothing arrived? Check spam, then try again. Make sure it's the address your administrator set the account up with.",
        )}
      </AuthHint>
    </AuthSuccess>
  );
}
