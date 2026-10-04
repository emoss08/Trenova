import { TurnstileWidget, type TurnstileHandle } from "@/components/turnstile-widget";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { cloudSignupService } from "@/services/cloud-signup";
import type { SignupResendRequest } from "@/types/cloud-signup";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { z } from "zod";
import { AuthErrorText, AuthSubmit, AuthTextField } from "../../auth/_components/auth-field";
import { rateLimitMessage } from "./signup-form";
import { formatCooldown, RESEND_COOLDOWN_SECONDS, RESEND_TURNSTILE_ACTION } from "./signup-sent";

const resendSchema = z.object({
  emailAddress: z
    .string()
    .trim()
    .min(1, { error: "Enter the email you signed up with" })
    .pipe(z.email({ error: "Enter a valid email address" })),
});

type ResendValues = z.input<typeof resendSchema>;

/**
 * Asks for a new verification link when the one in hand is spent, expired or
 * mangled. The answer is the same whether or not the address has a pending signup.
 */
export function ResendVerification({ turnstileSiteKey }: { turnstileSiteKey: string }) {
  const t = useT();
  const turnstileRef = useRef<TurnstileHandle>(null);
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const [cooldownUntil, setCooldownUntil] = useState(0);
  const [sentTo, setSentTo] = useState<string | null>(null);
  const now = useNowSeconds(1000);
  const remaining = Math.max(0, cooldownUntil - now);

  const form = useForm<ResendValues>({
    resolver: zodResolver(resendSchema),
    defaultValues: { emailAddress: "" },
  });
  const { control, handleSubmit, formState, setError } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending } = useMutation({
    mutationFn: (request: SignupResendRequest) => cloudSignupService.resend(request),
    onError: (error) => {
      if (error instanceof ApiRequestError) {
        if (error.isRateLimitError()) {
          setError("root", { message: t(rateLimitMessage(error.retryAfter)) });
          if (error.retryAfter) {
            setCooldownUntil(Math.floor(Date.now() / 1000) + error.retryAfter);
          }
          return;
        }
        const emailError = error.getFieldError("emailAddress");
        if (emailError) {
          setError("emailAddress", { message: emailError.message });
          return;
        }
        setError("root", {
          message: error.getFieldError("turnstileToken")?.message ?? error.normalize().message,
        });
        return;
      }
      setError("root", { message: t("The email could not be sent. Try again.") });
    },
  });

  const onSubmit = async (values: ResendValues) => {
    if (!turnstileToken) {
      setError("root", { message: t("Complete the security check first.") });
      return;
    }
    const token = turnstileToken;
    turnstileRef.current?.reset();

    const emailAddress = values.emailAddress.trim();
    try {
      await mutateAsync({ emailAddress, turnstileToken: token });
    } catch {
      return;
    }
    setSentTo(emailAddress);
    setCooldownUntil(Math.floor(Date.now() / 1000) + RESEND_COOLDOWN_SECONDS);
  };

  return (
    <form
      className="mt-4 flex flex-col gap-3.5"
      noValidate
      onSubmit={(event) => void handleSubmit(onSubmit)(event)}
    >
      <AuthTextField
        name="emailAddress"
        control={control}
        label={t("Email address")}
        type="email"
        required
        placeholder="name@work-email.com"
        autoComplete="email"
        disabled={isPending}
      />
      <TurnstileWidget
        ref={turnstileRef}
        siteKey={turnstileSiteKey}
        action={RESEND_TURNSTILE_ACTION}
        onTokenChange={setTurnstileToken}
      />
      {rootError ? <AuthErrorText>{rootError}</AuthErrorText> : null}
      {sentTo && !rootError ? (
        <span role="status" className="text-success text-xs">
          {t("If {0} has a pending signup, a new link is on its way.", sentTo)}
        </span>
      ) : null}
      <AuthSubmit
        type="submit"
        isLoading={isPending}
        loadingText={t("Sending")}
        disabled={remaining > 0 || !turnstileToken}
      >
        {remaining > 0 ? t("Resend in {0}", formatCooldown(remaining)) : t("Send a new link")}
      </AuthSubmit>
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <Link to="/login" className="hover:text-foreground underline underline-offset-[3px]">
          {t("Sign in instead")}
        </Link>
        <Link to="/signup" className="hover:text-foreground underline underline-offset-[3px]">
          {t("Start over")}
        </Link>
      </div>
    </form>
  );
}
