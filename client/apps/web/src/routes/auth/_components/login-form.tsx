import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { emailSchema } from "@/lib/auth-validation";
import { edition } from "@/lib/edition";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { authService } from "@trenova/shared/services/auth";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { isMFAChallenge, type MFAChallenge } from "@trenova/shared/types/mfa";
import type { TenantLoginMetadata } from "@trenova/shared/types/organization";
import {
  loginRequestSchema,
  type LoginRequest,
  type LoginResponse,
} from "@trenova/shared/types/user";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useSearchParams } from "react-router";
import { AuthErrorText, AuthPasswordField, AuthSubmit, AuthTextField } from "./auth-field";
import { AuthHeading, AuthQuietButton } from "./auth-primitives";
import { MFAChallengeForm } from "./mfa-challenge-form";
import { useAuthStage } from "./stage/auth-stage-context";

const LoginPrompt = edition.slots.LoginPrompt;

const loginFormSchema = loginRequestSchema.extend({
  emailAddress: emailSchema(() => translate("Enter the work email you were invited with.")),
});

export function LoginForm({
  organizationSlug,
  tenantMetadata,
  stepLabel,
  onAuthenticated,
  onForgotPassword,
}: {
  organizationSlug?: string;
  tenantMetadata?: TenantLoginMetadata;
  /** The flow's step counter, shown by the second-factor step if one follows. */
  stepLabel: string;
  onAuthenticated: (response: LoginResponse) => Promise<void> | void;
  // Handed whatever is already in the email box, so recovery does not start by asking
  // for an address the user just typed.
  onForgotPassword: (emailAddress: string) => void;
}) {
  const t = useT();
  const stage = useAuthStage();

  const [searchParams, setSearchParams] = useSearchParams();
  const ssoError = searchParams.get("sso_error");
  const setUser = useAuthStore((state) => state.setUser);

  const providerQuery = useQuery({
    queryKey: ["auth-providers", organizationSlug],
    queryFn: async () => authService.listProviders(organizationSlug ?? ""),
    enabled: Boolean(organizationSlug),
  });
  const providers = providerQuery.data ?? [];
  const hasAnySso = providers.length > 0;
  const passwordEnabled = tenantMetadata?.passwordEnabled ?? true;
  const returnTo = typeof window !== "undefined" ? `${window.location.origin}/` : "/";

  const form = useForm<LoginRequest>({
    resolver: zodResolver(loginFormSchema),
    defaultValues: {
      emailAddress: "",
      password: "",
      organizationSlug,
    },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const [challenge, setChallenge] = useState<MFAChallenge | null>(null);

  // Back on the sign-in form means no one is signed in, whichever step led here.
  useEffect(() => {
    stage.setDone(false);
  }, [stage]);

  const authenticated = async (response: LoginResponse) => {
    stage.setDone(true);
    try {
      await onAuthenticated(response);
    } catch (error) {
      stage.setDone(false);
      throw error;
    }
  };

  const { mutate, isPending } = useApiMutation({
    mutationFn: authService.login,
    form,
    resourceName: "Login",
    onSuccess: async (data) => {
      if (isMFAChallenge(data)) {
        setChallenge(data);
        return;
      }
      setUser(data.user);
      await authenticated(data);
    },
  });

  const onSubmit = (data: LoginRequest) => {
    stage.burst();
    mutate(data);
  };

  if (challenge) {
    return (
      <MFAChallengeForm
        challenge={challenge}
        stepLabel={stepLabel}
        onAuthenticated={authenticated}
        onCancel={() => {
          setChallenge(null);
          form.setValue("password", "");
        }}
      />
    );
  }

  return (
    <>
      <AuthHeading title={tenantMetadata?.organizationName ?? t("Sign in")}>
        {tenantMetadata ? (
          t("Sign in to {0}", tenantMetadata.organizationName)
        ) : (
          <LoginPrompt fallback={t("Sign in with the account your organization set up for you.")} />
        )}
      </AuthHeading>

      <form className="flex flex-col gap-4" noValidate onSubmit={handleSubmit(onSubmit)}>
        {ssoError && (
          <button
            type="button"
            className="ui-focus-ring text-danger cursor-pointer rounded-sm text-left text-base"
            onClick={() =>
              setSearchParams((params) => {
                params.delete("sso_error");
                return params;
              })
            }
          >
            {ssoError}
          </button>
        )}

        {hasAnySso && (
          <>
            <div className="flex flex-col gap-2">
              {providers.map((provider) => (
                <a
                  key={provider.id}
                  href={authService.getSSOStartUrl(provider.id, organizationSlug ?? "", returnTo)}
                  className="ui-focus-ring bg-auth-field border-border-strong text-auth-body hover:border-foreground/35 flex h-11 items-center justify-center gap-[9px] rounded-[10px] border font-medium whitespace-nowrap transition-colors"
                >
                  <ProviderLogo provider={provider.provider} />
                  {t("Continue with {0}", provider.name)}
                </a>
              ))}
            </div>
            {passwordEnabled && (
              <div className="text-muted-foreground before:bg-border after:bg-border flex items-center gap-3 text-sm before:h-px before:flex-1 before:content-[''] after:h-px after:flex-1 after:content-['']">
                {t("or")}
              </div>
            )}
          </>
        )}

        {passwordEnabled && (
          <>
            <AuthTextField
              name="emailAddress"
              control={control}
              label={t("Email")}
              type="email"
              placeholder="you@carrier.com"
              autoComplete="username"
              disabled={isPending}
            />
            <AuthPasswordField
              name="password"
              control={control}
              label={t("Password")}
              placeholder="••••••••"
              autoComplete="current-password"
              disabled={isPending}
              trailing={
                <AuthQuietButton onClick={() => onForgotPassword(form.getValues("emailAddress"))}>
                  {t("Forgot?")}
                </AuthQuietButton>
              }
            />
            {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
            <AuthSubmit type="submit" busy={isPending} busyLabel={t("Verifying")}>
              {t("Sign in")}
            </AuthSubmit>
          </>
        )}
      </form>
    </>
  );
}

function ProviderLogo({ provider }: { provider: string }) {
  if (provider === "AzureAD") {
    return <EntraLogo className="size-3.5" />;
  }
  if (provider === "Okta") {
    return <OktaLogo className="h-3.5 w-auto" />;
  }
  return (
    <span className="border-border-2 text-subtle-foreground font-table text-3xs flex size-3.5 items-center justify-center rounded-sm border font-semibold">
      S
    </span>
  );
}
