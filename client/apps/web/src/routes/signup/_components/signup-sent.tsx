import { TurnstileWidget, type TurnstileHandle } from "@/components/turnstile-widget";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { cloudSignupService } from "@/services/cloud-signup";
import type { SignupResendRequest } from "@/types/cloud-signup";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AuthCardBody } from "../../auth/_components/auth-card";
import { AuthErrorText, AuthSubmit } from "../../auth/_components/auth-field";
import { AuthTray, StepCrumbs, StepHeading } from "../../auth/_components/auth-primitives";
import { rateLimitMessage } from "./signup-form";

export const RESEND_COOLDOWN_SECONDS = 60;
export const RESEND_TURNSTILE_ACTION = "signup_resend";

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

export function formatCooldown(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return `${minutes}:${String(rest).padStart(2, "0")}`;
}

/**
 * Shown after the server accepts a signup. It says the same thing whether the address
 * was new, already pending or already an account, because the server answers the same
 * way in every case — a page that said more would tell a stranger which addresses
 * have accounts.
 */
export function SignupSent({
  emailAddress,
  turnstileSiteKey,
  onStartOver,
}: {
  emailAddress: string;
  turnstileSiteKey: string;
  onStartOver: () => void;
}) {
  const t = useT();
  const turnstileRef = useRef<TurnstileHandle>(null);
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const [cooldownUntil, setCooldownUntil] = useState(() => nowSeconds() + RESEND_COOLDOWN_SECONDS);
  const [resent, setResent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const now = useNowSeconds(1000);
  const remaining = Math.max(0, cooldownUntil - now);

  // Errors are shown inline beside the button rather than as toasts: the person is
  // looking at this one control, and a toast would only repeat it further away.
  const { mutateAsync, isPending } = useMutation({
    mutationFn: (request: SignupResendRequest) => cloudSignupService.resend(request),
    onError: (mutationError) => {
      if (mutationError instanceof ApiRequestError) {
        if (mutationError.isRateLimitError()) {
          setError(t(rateLimitMessage(mutationError.retryAfter)));
          if (mutationError.retryAfter) {
            setCooldownUntil(nowSeconds() + mutationError.retryAfter);
          }
          return;
        }
        const turnstileError = mutationError.getFieldError("turnstileToken");
        setError(turnstileError?.message ?? mutationError.normalize().message);
        return;
      }
      setError(t("The email could not be sent. Try again."));
    },
  });

  const resend = async () => {
    if (!turnstileToken || remaining > 0) {
      return;
    }
    const token = turnstileToken;
    turnstileRef.current?.reset();
    setError(null);

    try {
      await mutateAsync({ emailAddress, turnstileToken: token });
    } catch {
      return;
    }
    setResent(true);
    setCooldownUntil(nowSeconds() + RESEND_COOLDOWN_SECONDS);
  };

  return (
    <AuthCardBody>
      <StepCrumbs left="Trenova Cloud" right="Link sent" />
      <StepHeading title={t("Check your inbox")}>
        {t(
          "If {0} can be used for a new account, a verification link is on its way. Open it on this device to finish setting up your workspace.",
          emailAddress,
        )}
      </StepHeading>

      <p className="text-subtle-foreground mt-4 mb-0 text-xs">
        {t(
          "The link works once and expires after a day. Nothing arrived? Check spam, then send it again.",
        )}
      </p>

      <div className="mt-4 flex flex-col gap-3">
        <TurnstileWidget
          ref={turnstileRef}
          siteKey={turnstileSiteKey}
          action={RESEND_TURNSTILE_ACTION}
          onTokenChange={setTurnstileToken}
        />
        {error ? <AuthErrorText>{error}</AuthErrorText> : null}
        {resent && !error ? (
          <span role="status" className="text-success text-xs">
            {t("Sent again. Use the newest link; earlier ones stop working.")}
          </span>
        ) : null}
        <AuthSubmit
          onClick={() => void resend()}
          isLoading={isPending}
          loadingText={t("Sending")}
          disabled={remaining > 0 || !turnstileToken}
        >
          {remaining > 0
            ? t("Resend in {0}", formatCooldown(remaining))
            : t("Resend verification email")}
        </AuthSubmit>
      </div>

      <AuthTray onBack={onStartOver} hints={<span>{t("Wrong address? Start over.")}</span>} />
    </AuthCardBody>
  );
}
