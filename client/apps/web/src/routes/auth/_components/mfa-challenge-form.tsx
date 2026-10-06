import { useApiMutation } from "@/hooks/use-api-mutation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import { authService } from "@trenova/shared/services/auth";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  verifyMFAChallengeRequestSchema,
  type MFAChallenge,
  type VerifyMFAChallengeRequest,
} from "@trenova/shared/types/mfa";
import type { LoginResponse } from "@trenova/shared/types/user";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { AuthCardBody } from "./auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { StepCrumbs, StepHeading } from "./auth-primitives";

/**
 * The second step of a password sign-in for an account with an authenticator app:
 * the six-digit code, or one of the recovery codes issued when it was set up.
 */
export function MFAChallengeForm({
  challenge,
  stepLabel,
  onAuthenticated,
  onCancel,
}: {
  challenge: MFAChallenge;
  stepLabel: string;
  onAuthenticated: (response: LoginResponse) => Promise<void> | void;
  onCancel: () => void;
}) {
  const t = useT();
  const setUser = useAuthStore((state) => state.setUser);
  const [useRecovery, setUseRecovery] = useState(false);

  const form = useForm<VerifyMFAChallengeRequest>({
    resolver: zodResolver(verifyMFAChallengeRequestSchema),
    defaultValues: {
      challengeToken: challenge.mfaChallengeToken,
      code: "",
      recoveryCode: "",
    },
  });
  const rootError = form.formState.errors.root?.message;

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: authService.verifyMFA,
    form,
    resourceName: "Two-factor sign-in",
    onSuccess: async (data) => {
      setUser(data.user);
      await onAuthenticated(data);
    },
  });

  const toggleRecovery = () => {
    form.reset({
      challengeToken: challenge.mfaChallengeToken,
      code: "",
      recoveryCode: "",
    });
    setUseRecovery((current) => !current);
  };

  return (
    <AuthCardBody>
      <StepCrumbs left={stepLabel} right={t("Two-factor authentication")} />
      <StepHeading title={t("Enter your code")}>
        {useRecovery
          ? t("Enter one of the recovery codes you saved when you set up two-factor authentication.")
          : t("Open your authenticator app and enter the six-digit code for Trenova.")}
      </StepHeading>

      <form
        className="mt-[18px] flex flex-col gap-3.5"
        noValidate
        onSubmit={form.handleSubmit((values) => void mutateAsync(values))}
      >
        {useRecovery ? (
          <AuthTextField
            name="recoveryCode"
            control={form.control}
            label={t("Recovery code")}
            placeholder="xxxxx-xxxxx"
            autoComplete="one-time-code"
            required
            disabled={isPending}
          />
        ) : (
          <AuthTextField
            name="code"
            control={form.control}
            label={t("Authentication code")}
            placeholder="123 456"
            autoComplete="one-time-code"
            required
            disabled={isPending}
          />
        )}
        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit type="submit" isLoading={isPending} loadingText={t("Verifying code")}>
          {t("Verify")}
        </AuthSubmit>
        <div className="flex items-center justify-between gap-3 text-xs">
          <button
            type="button"
            onClick={toggleRecovery}
            className="text-muted-foreground hover:text-foreground cursor-pointer bg-transparent transition-colors duration-150"
          >
            {useRecovery ? t("Use your authenticator app") : t("Use a recovery code")}
          </button>
          <button
            type="button"
            onClick={onCancel}
            className="text-muted-foreground hover:text-foreground cursor-pointer bg-transparent transition-colors duration-150"
          >
            {t("Start over")}
          </button>
        </div>
      </form>
    </AuthCardBody>
  );
}
