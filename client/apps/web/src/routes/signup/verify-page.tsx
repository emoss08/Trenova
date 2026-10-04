import { Metadata } from "@/components/metadata";
import { ONBOARDING_QUERY_ROOT } from "@/lib/queries/onboarding";
import { cloudSignupService } from "@/services/cloud-signup";
import { useQueryClient } from "@tanstack/react-query";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { planLimitFromError, SIGNUPS_PAUSED_REASON } from "@trenova/shared/lib/plan-limit";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { AuthCard, AuthCardBody } from "../auth/_components/auth-card";
import { AuthSubmit } from "../auth/_components/auth-field";
import type { CredentialReceipt } from "../auth/_components/auth-panel";
import { AuthMobileBrand, StepCrumbs, StepHeading } from "../auth/_components/auth-primitives";
import { AuthShell } from "../auth/_components/auth-shell";
import { ResendVerification } from "./_components/resend-verification";

export type VerifyOutcome =
  | { status: "verifying" }
  | { status: "verified" }
  | { status: "missing-token" }
  | { status: "invalid-token" }
  | { status: "signups-paused" }
  | { status: "rate-limited"; retryAfter: number | null }
  | { status: "failed"; message: string };

const INVALID_TOKEN_STATUSES = new Set([400, 404, 409, 410, 422]);

/** What a failed verification means to the person holding the link. */
export function verifyFailureOutcome(error: unknown): VerifyOutcome {
  const planLimit = planLimitFromError(error);
  if (planLimit?.kind === "restricted" && planLimit.reason === SIGNUPS_PAUSED_REASON) {
    return { status: "signups-paused" };
  }

  if (error instanceof ApiRequestError) {
    if (error.isRateLimitError()) {
      return { status: "rate-limited", retryAfter: error.retryAfter };
    }
    if (INVALID_TOKEN_STATUSES.has(error.status) || error.isNotFoundError()) {
      return { status: "invalid-token" };
    }
    if (error.status > 0 && error.status < 500) {
      return { status: "failed", message: error.normalize().message };
    }
  }

  return {
    status: "failed",
    message:
      "We could not reach Trenova to verify your email. Check your connection and try again.",
  };
}

/**
 * The page the verification email opens. It spends the token once — the service
 * shares one request between React's development double-mount and a real mount —
 * and on success signs the new owner in exactly as the sign-in form does before
 * handing them to the welcome wizard.
 */
export function SignupVerifyPage() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token")?.trim() ?? "";
  const { config } = usePublicConfig();
  const setUser = useAuthStore((state) => state.setUser);
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const [attempt, setAttempt] = useState(0);
  const [outcome, setOutcome] = useState<VerifyOutcome>(() =>
    token ? { status: "verifying" } : { status: "missing-token" },
  );

  useEffect(() => {
    if (!token) {
      return;
    }

    let active = true;
    cloudSignupService
      .verifyOnce(token)
      .then(async (response) => {
        if (!active) {
          return;
        }
        setUser(response.user);
        queryClient.removeQueries({ queryKey: ONBOARDING_QUERY_ROOT });
        try {
          await fetchManifest();
        } catch {
          // The app shell fetches the manifest again on its own; a failure here only
          // means the first page waits for it.
        }
        if (active) {
          setOutcome({ status: "verified" });
          void navigate("/onboarding", { replace: true });
        }
      })
      .catch((error: unknown) => {
        if (active) {
          setOutcome(verifyFailureOutcome(error));
        }
      });

    return () => {
      active = false;
    };
  }, [token, attempt, fetchManifest, navigate, queryClient, setUser]);

  const retry = useCallback(() => {
    setOutcome({ status: "verifying" });
    setAttempt((current) => current + 1);
  }, []);

  const receipt: CredentialReceipt = {
    issued: outcome.status === "verified",
    rows: [
      { key: "Link", value: token ? "Received" : undefined },
      { key: "Email", value: outcome.status === "verified" ? "Verified" : undefined },
      { key: "Workspace", value: outcome.status === "verified" ? "Provisioned" : undefined },
    ],
  };

  return (
    <>
      <Metadata title={t("Verify email")} description={t("Finish creating your Trenova account")} />
      <AuthShell step={outcome.status === "verified" ? "done" : "login"} receipt={receipt}>
        <AuthMobileBrand />
        <AuthCard stepKey={outcome.status}>
          <VerifyBody
            outcome={outcome}
            turnstileSiteKey={config.turnstileSiteKey}
            onRetry={retry}
          />
        </AuthCard>
      </AuthShell>
    </>
  );
}

function VerifyBody({
  outcome,
  turnstileSiteKey,
  onRetry,
}: {
  outcome: VerifyOutcome;
  turnstileSiteKey: string;
  onRetry: () => void;
}) {
  const t = useT();

  switch (outcome.status) {
    case "verifying":
    case "verified":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Verifying" />
          <StepHeading title={t("Setting up your workspace")}>
            {t("Verifying your email and preparing your organization. This takes a few seconds.")}
          </StepHeading>
          <div className="mt-4">
            <AuthSubmit isLoading loadingText={t("Verifying")}>
              {t("Verifying")}
            </AuthSubmit>
          </div>
        </AuthCardBody>
      );
    case "missing-token":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Link problem" />
          <StepHeading title={t("This link is incomplete")}>
            {t(
              "The verification link is missing its token. Some mail clients wrap long links across lines — copy the whole thing, or ask for a new one.",
            )}
          </StepHeading>
          <ResendVerification turnstileSiteKey={turnstileSiteKey} />
        </AuthCardBody>
      );
    case "invalid-token":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Link problem" />
          <StepHeading title={t("This link has expired")}>
            {t(
              "Verification links work once and expire after a day. If you already used this one, sign in instead; otherwise ask for a new link.",
            )}
          </StepHeading>
          <ResendVerification turnstileSiteKey={turnstileSiteKey} />
        </AuthCardBody>
      );
    case "signups-paused":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Wait list" />
          <StepHeading title={t("You're on the wait list")}>
            {t(
              "Your email is verified, but the free demo is full right now. We open new workspaces as space frees up and will email you when yours is ready — there's nothing else you need to do.",
            )}
          </StepHeading>
          <div className="mt-4">
            <Link
              to="/login"
              className="text-muted-foreground hover:text-foreground text-sm underline underline-offset-[3px]"
            >
              {t("Back to sign in")}
            </Link>
          </div>
        </AuthCardBody>
      );
    case "rate-limited":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Slow down" />
          <StepHeading title={t("Too many attempts")}>
            {outcome.retryAfter
              ? t(
                  "This network has made too many requests. Try again in {0} minutes.",
                  String(Math.max(1, Math.ceil(outcome.retryAfter / 60))),
                )
              : t("This network has made too many requests. Wait a while and try again.")}
          </StepHeading>
          <div className="mt-4">
            <AuthSubmit onClick={onRetry}>{t("Try again")}</AuthSubmit>
          </div>
        </AuthCardBody>
      );
    case "failed":
      return (
        <AuthCardBody>
          <StepCrumbs left="Trenova Cloud" right="Not verified" />
          <StepHeading title={t("We couldn't verify your email")}>{t(outcome.message)}</StepHeading>
          <div className="mt-4">
            <AuthSubmit onClick={onRetry}>{t("Try again")}</AuthSubmit>
          </div>
        </AuthCardBody>
      );
  }
}
