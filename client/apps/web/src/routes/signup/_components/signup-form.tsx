import { TurnstileWidget, type TurnstileHandle } from "@/components/turnstile-widget";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useLegalUrls } from "@/hooks/use-legal-urls";
import { cloudSignupService } from "@/services/cloud-signup";
import {
  SIGNUP_FORM_DEFAULTS,
  signupFormSchema,
  type SignupFormValues,
  type SignupRequest,
} from "@/types/cloud-signup";
import { zodResolver } from "@hookform/resolvers/zod";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useId, useRef, useState } from "react";
import { Controller, useForm, useWatch, type Control } from "react-hook-form";
import { Link } from "react-router";
import { AuthCardBody } from "../../auth/_components/auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "../../auth/_components/auth-field";
import { StepCrumbs, StepHeading } from "../../auth/_components/auth-primitives";
import { PasswordRequirements } from "./password-requirements";

export const SIGNUP_TURNSTILE_ACTION = "signup";

export function rateLimitMessage(retryAfter: number | null): string {
  if (retryAfter === null || retryAfter <= 0) {
    return "Too many signup attempts from this network. Wait a while and try again.";
  }
  const minutes = Math.ceil(retryAfter / 60);
  return minutes <= 1
    ? "Too many signup attempts from this network. Try again in a minute."
    : `Too many signup attempts from this network. Try again in ${minutes} minutes.`;
}

export function SignupForm({
  turnstileSiteKey,
  onSubmitted,
}: {
  turnstileSiteKey: string;
  onSubmitted: (submission: { emailAddress: string; companyName: string }) => void;
}) {
  const t = useT();
  const { termsUrl, privacyUrl } = useLegalUrls();
  const termsId = useId();
  const turnstileRef = useRef<TurnstileHandle>(null);
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);

  const form = useForm<SignupFormValues>({
    resolver: zodResolver(signupFormSchema),
    defaultValues: SIGNUP_FORM_DEFAULTS,
    mode: "onTouched",
  });
  const { control, handleSubmit, formState, setError } = form;
  const rootError = formState.errors.root?.message;
  const [password, emailAddress] = useWatch({ control, name: ["password", "emailAddress"] });

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: (request: SignupRequest) => cloudSignupService.signup(request),
    form,
    resourceName: "Signup",
    onError: (error) => {
      if (error instanceof ApiRequestError && error.isRateLimitError()) {
        setError("root", { type: "server", message: t(rateLimitMessage(error.retryAfter)) });
      }
    },
  });

  const onSubmit = async (values: SignupFormValues) => {
    if (!turnstileToken) {
      setError("root", {
        type: "validation",
        message: t("Complete the security check before creating your account."),
      });
      return;
    }

    const token = turnstileToken;
    // A Turnstile token is good for one verification. Whatever the server answers, this
    // one is spent, so a fresh challenge starts now rather than after the next failure.
    turnstileRef.current?.reset();

    try {
      await mutateAsync({
        name: values.name.trim(),
        emailAddress: values.emailAddress.trim(),
        companyName: values.companyName.trim(),
        password: values.password,
        acceptTerms: values.acceptTerms,
        website: values.website,
        turnstileToken: token,
      });
    } catch {
      return;
    }

    onSubmitted({
      emailAddress: values.emailAddress.trim(),
      companyName: values.companyName.trim(),
    });
  };

  return (
    <AuthCardBody>
      <StepCrumbs left="Trenova Cloud" right="Free demo" />
      <StepHeading title={t("Create your account")}>
        {t("Already have one?")}{" "}
        <Link
          to="/login"
          className="text-foreground decoration-foreground/35 hover:decoration-foreground underline underline-offset-[3px]"
        >
          {t("Sign in")}
        </Link>
      </StepHeading>

      <form
        className="mt-4 flex flex-col gap-3.5"
        noValidate
        onSubmit={(event) => void handleSubmit(onSubmit)(event)}
      >
        <AuthTextField
          name="name"
          control={control}
          label={t("Your name")}
          required
          placeholder={t("Jordan Rivera")}
          autoComplete="name"
          disabled={isPending}
        />
        <AuthTextField
          name="emailAddress"
          control={control}
          label={t("Work email")}
          type="email"
          required
          placeholder="name@work-email.com"
          autoComplete="email"
          disabled={isPending}
        />
        <AuthTextField
          name="companyName"
          control={control}
          label={t("Company name")}
          required
          placeholder={t("Rivera Freight LLC")}
          autoComplete="organization"
          disabled={isPending}
        />
        <div className="flex flex-col gap-2">
          <AuthTextField
            name="password"
            control={control}
            label={t("Password")}
            type="password"
            required
            revealable
            placeholder={t("At least 12 characters")}
            autoComplete="new-password"
            disabled={isPending}
          />
          <PasswordRequirements password={password ?? ""} emailAddress={emailAddress ?? ""} />
        </div>

        <HoneypotField control={control} />

        <Controller
          name="acceptTerms"
          control={control}
          render={({ field, fieldState }) => (
            <div className="flex flex-col gap-1.5">
              <div className="flex items-start gap-2.5">
                <Checkbox
                  id={termsId}
                  checked={field.value}
                  onCheckedChange={(checked) => field.onChange(checked === true)}
                  onBlur={field.onBlur}
                  disabled={isPending}
                  aria-invalid={fieldState.invalid}
                  aria-describedby={fieldState.error ? `${termsId}-error` : undefined}
                  className="mt-0.5"
                />
                <label htmlFor={termsId} className="text-muted-foreground text-xs leading-[1.5]">
                  {t("I agree to the")}{" "}
                  <a
                    href={termsUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-foreground underline underline-offset-[3px]"
                  >
                    {t("Terms of Service")}
                  </a>{" "}
                  {t("and")}{" "}
                  <a
                    href={privacyUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-foreground underline underline-offset-[3px]"
                  >
                    {t("Privacy policy")}
                  </a>
                  .
                </label>
              </div>
              {fieldState.error?.message ? (
                <AuthErrorText id={`${termsId}-error`}>{fieldState.error.message}</AuthErrorText>
              ) : null}
            </div>
          )}
        />

        <TurnstileWidget
          ref={turnstileRef}
          siteKey={turnstileSiteKey}
          action={SIGNUP_TURNSTILE_ACTION}
          onTokenChange={setTurnstileToken}
        />

        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit
          type="submit"
          isLoading={isPending}
          loadingText={t("Creating your account")}
          disabled={turnstileSiteKey === ""}
        >
          {t("Create account")}
        </AuthSubmit>
      </form>
    </AuthCardBody>
  );
}

/**
 * A field no person sees or reaches: off-screen, hidden from assistive technology,
 * out of the tab order and exempt from autofill. A bot that fills every input fills
 * this one too, and the server quietly drops that request.
 */
function HoneypotField({ control }: { control: Control<SignupFormValues> }) {
  return (
    <Controller
      name="website"
      control={control}
      render={({ field }) => (
        <div
          aria-hidden="true"
          className="pointer-events-none absolute -left-[10000px] h-px w-px overflow-hidden opacity-0"
        >
          <label>
            Website
            <input
              {...field}
              type="text"
              tabIndex={-1}
              autoComplete="off"
              value={field.value ?? ""}
            />
          </label>
        </div>
      )}
    />
  );
}
