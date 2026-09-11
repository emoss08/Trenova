import { useT } from "@trenova/shared/i18n/use-t";
import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@trenova/shared/services/auth";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { TenantLoginMetadata } from "@trenova/shared/types/organization";
import {
  loginRequestSchema,
  type LoginRequest,
  type LoginResponse,
} from "@trenova/shared/types/user";
import { cn } from "@trenova/shared/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { BuildingIcon, TruckIcon } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useSearchParams } from "react-router";
import { AuthCardBody } from "./auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { StepCrumbs, StepHeading } from "./auth-primitives";

export type AuthAudience = "office" | "driver";

const AUDIENCE_OPTIONS = [
  { value: "office", label: "Office", icon: BuildingIcon },
  { value: "driver", label: "Driver", icon: TruckIcon },
] as const;

export function LoginForm({
  organizationSlug,
  tenantMetadata,
  stepLabel,
  onAuthenticated,
  onForgotPassword,
}: {
  organizationSlug?: string;
  tenantMetadata?: TenantLoginMetadata;
  stepLabel: string;
  onAuthenticated: (response: LoginResponse) => Promise<void> | void;
  // Handed whatever is already in the email box, so recovery does not start by asking
  // for an address the user just typed.
  onForgotPassword: (emailAddress: string) => void;
}) {
  const t = useT();

  const [searchParams, setSearchParams] = useSearchParams();
  const ssoError = searchParams.get("sso_error");
  const setUser = useAuthStore((state) => state.setUser);
  const [audience, setAudience] = useState<AuthAudience>("office");

  // The audience toggle only makes sense on the generic sign-in page: a tenant login
  // page is already scoped to one organization's office users.
  const showAudienceToggle = !tenantMetadata;
  const isDriverAudience = showAudienceToggle && audience === "driver";

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
    resolver: zodResolver(loginRequestSchema),
    defaultValues: {
      emailAddress: "",
      password: "",
      organizationSlug,
    },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: authService.login,
    form,
    resourceName: "Login",
    onSuccess: async (data) => {
      setUser(data.user);
      await onAuthenticated(data);
    },
  });

  const onSubmit = (data: LoginRequest) => {
    void mutateAsync(data);
  };

  return (
    <AuthCardBody>
      <StepCrumbs left={stepLabel} right="Secure sign-in" />
      <StepHeading
        title={
          isDriverAudience ? "Driver sign-in" : (tenantMetadata?.organizationName ?? "Welcome back")
        }
      >
        {isDriverAudience ? (
          "Dash is where drivers see loads and pay."
        ) : tenantMetadata ? (
          `Sign in to ${tenantMetadata.organizationName}`
        ) : (
          <>
            {t("Don't have an account yet?")}{" "}
            <a
              href="#"
              className="text-foreground decoration-foreground/35 hover:decoration-foreground underline underline-offset-[3px]"
              onClick={(event) => event.preventDefault()}
            >
              {t("Create an account")}
            </a>
          </>
        )}
      </StepHeading>

      {showAudienceToggle ? (
        <AudienceToggle value={audience} onChange={setAudience} />
      ) : (
        <div className="h-[18px]" />
      )}

      {isDriverAudience ? (
        <div className="flex flex-col gap-3.5">
          <p className="text-muted-foreground m-0 text-[12.5px]">
            {t("Loads, settlement statements and pay — built for the phone.")}
          </p>
          <a
            href="/dash/login"
            className="bg-foreground text-background border-foreground flex h-10 w-full items-center justify-center gap-2 rounded-[9px] border text-[13px] font-[550] transition-[opacity,transform] duration-150 hover:opacity-90 active:scale-[0.988]"
          >
            {t("Continue to Dash")}
            <ArrowRight />
          </a>
          <p className="text-subtle-foreground m-0 text-[11.5px]">
            {t("First time here? Use the invitation link your carrier sent you.")}
          </p>
        </div>
      ) : (
        <form className="flex flex-col gap-3.5" noValidate onSubmit={handleSubmit(onSubmit)}>
          {ssoError && (
            <button
              type="button"
              className="text-auth-danger cursor-pointer text-left text-[11.5px]"
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
                    className="border-border hover:border-input flex h-[38px] items-center justify-center gap-[9px] rounded-[9px] border bg-transparent text-[12.5px] font-medium whitespace-nowrap transition-colors duration-150 hover:bg-[color-mix(in_oklch,var(--foreground)_5%,transparent)]"
                  >
                    <ProviderLogo provider={provider.provider} />
                    {t("Continue with {0}", provider.name)}
                  </a>
                ))}
              </div>
              {passwordEnabled && (
                <div className="text-subtle-foreground flex items-center gap-2.5 text-[11px] before:bg-border-2 after:bg-border-2 before:h-px before:flex-1 after:h-px after:flex-1 before:content-[''] after:content-['']">
                  or
                </div>
              )}
            </>
          )}

          {passwordEnabled && (
            <>
              <AuthTextField
                name="emailAddress"
                control={control}
                label={t("Email address")}
                type="email"
                required
                placeholder="name@work-email.com"
                autoComplete="username"
                disabled={isPending}
              />
              <AuthTextField
                name="password"
                control={control}
                label={t("Password")}
                type="password"
                required
                revealable
                placeholder="••••••••"
                autoComplete="current-password"
                disabled={isPending}
                trailing={
                  <button
                    type="button"
                    onClick={() => onForgotPassword(form.getValues("emailAddress"))}
                    className="text-muted-foreground hover:text-foreground cursor-pointer bg-transparent text-[11.5px] transition-colors duration-150"
                  >
                    {t("Forgot?")}
                  </button>
                }
              />
              {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
              <AuthSubmit type="submit" isLoading={isPending} loadingText={t("Verifying credentials")}>
                {t("Sign in")}
              </AuthSubmit>
            </>
          )}
        </form>
      )}
    </AuthCardBody>
  );
}

/**
 * Segmented control rather than underline tabs: the knob slides between two equal
 * columns, measured from the active button so the track stays correct at any width.
 */
function AudienceToggle({
  value,
  onChange,
}: {
  value: AuthAudience;
  onChange: (audience: AuthAudience) => void;
}) {
  const t = useT();

  const trackRef = useRef<HTMLDivElement>(null);
  const [knob, setKnob] = useState({ x: 0, width: 0 });

  // offsetLeft is measured from the track's border edge, while the absolutely positioned
  // knob is placed against its padding edge. Subtracting clientLeft — the border width —
  // reconciles the two; a fixed offset instead leaves the knob a border-width off centre,
  // which reads as uneven padding on the inactive side.
  useLayoutEffect(() => {
    const track = trackRef.current;
    const active = track?.querySelector<HTMLButtonElement>(`[data-audience="${value}"]`);
    if (!track || !active) {
      return;
    }

    const measure = () =>
      setKnob({ x: active.offsetLeft - track.clientLeft, width: active.offsetWidth });

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(track);
    return () => observer.disconnect();
  }, [value]);

  return (
    <div
      ref={trackRef}
      role="tablist"
      className="bg-field border-border-2 relative mt-4 mb-[18px] grid grid-cols-2 gap-0.5 rounded-[9px] border p-[3px]"
    >
      <span
        aria-hidden="true"
        className="bg-popover border-border absolute top-[3px] bottom-[3px] left-0 rounded-md border transition-[transform,width] duration-[320ms] ease-[cubic-bezier(0.2,0.8,0.2,1)]"
        style={{ transform: `translateX(${knob.x}px)`, width: knob.width }}
      />
      {AUDIENCE_OPTIONS.map((option) => (
        <button
          key={option.value}
          type="button"
          role="tab"
          data-audience={option.value}
          aria-selected={value === option.value}
          onClick={() => onChange(option.value)}
          className={cn(
            "relative z-1 flex cursor-pointer items-center justify-center gap-[7px] rounded-md border-0 bg-transparent px-3 py-[7px] text-[12.5px] leading-[1.5] font-medium transition-colors duration-[180ms]",
            value === option.value ? "text-foreground" : "text-subtle-foreground",
          )}
        >
          <option.icon className="size-[15px]" />
          {t(option.label)}
        </button>
      ))}
    </div>
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
    <span className="border-border-2 text-subtle-foreground font-table flex size-3.5 items-center justify-center rounded-sm border text-[8px] font-semibold">
      S
    </span>
  );
}

function ArrowRight() {
  return (
    <svg
      viewBox="0 0 24 24"
      width="13"
      height="13"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  );
}
