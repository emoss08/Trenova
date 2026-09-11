import { useT } from "@trenova/shared/i18n/use-t";
import logoRainbow from "@/assets/logo.webp";
import { Metadata } from "@/components/metadata";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { PRIVACY_URL, TERMS_URL } from "@trenova/shared/lib/constants";
import { authService } from "@trenova/shared/services/auth";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import type { RoleSummary } from "@trenova/shared/types/role";
import type { TenantLoginMetadata, UserOrganization } from "@trenova/shared/types/organization";
import type { LoginResponse } from "@trenova/shared/types/user";
import type { UseQueryResult } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import { AuthCard, AuthCardBody } from "./auth-card";
import { AuthHandoff } from "./auth-handoff";
import type { CredentialReceipt } from "./auth-panel";
import { AuthShell, type AuthStep } from "./auth-shell";
import { ForgotPasswordForm } from "./forgot-password-form";
import { LoginForm } from "./login-form";
import { OrganizationSelection } from "./organization-selection";
import { RoleSelection, resolveAuthorizedRoles } from "./role-selection";

function stepLabel(index: number, total: number) {
  return `${String(index).padStart(2, "0")} / ${String(total).padStart(2, "0")}`;
}

function manifestOrganizationName(manifest: PermissionManifest): string | undefined {
  return manifest.availableOrgs.find((org) => org.id === manifest.organizationId)?.name;
}

// Undefined is not zero: a manifest rehydrated from localStorage, or one from a server
// that predates the count, carries no permissionCount at all. Summing those as zero
// would report "0 permissions" for a role that has hundreds, so an unknown count
// suppresses the clause instead (see AuthHandoff).
function countPermissions(roles: RoleSummary[]): number | undefined {
  if (roles.some((role) => role.permissionCount === undefined)) {
    return undefined;
  }
  return roles.reduce((total, role) => total + (role.permissionCount ?? 0), 0);
}

const SESSION_ID_BODY_LENGTH = 8;

/**
 * A session id is a PULID — a short prefix and a 26-character ULID — which is far wider
 * than the receipt row it sits in, and the AUTHORIZED stamp lands on top of that row's
 * right edge. Keeping the prefix and the leading body characters stays enough to match
 * a session against a log line without running under the stamp.
 */
export function formatSessionId(sessionId?: string): string | undefined {
  if (!sessionId) {
    return undefined;
  }

  const separator = sessionId.indexOf("_");
  if (separator === -1) {
    return sessionId.slice(0, SESSION_ID_BODY_LENGTH);
  }

  return sessionId.slice(0, separator + 1 + SESSION_ID_BODY_LENGTH);
}

/**
 * Owns the sign-in flow: login → organization → active roles → handoff.
 *
 * The two middle steps are conditional and the flow only learns which apply once the
 * server has answered — how many organizations the user belongs to, and whether the
 * session still needs roles activated — so the step counter is refined as it goes
 * rather than guessed up front.
 */
export function AuthForm({
  tenantQuery,
  organizationSlug,
}: {
  tenantQuery?: UseQueryResult<TenantLoginMetadata>;
  organizationSlug?: string;
}) {
  const t = useT();

  const navigate = useNavigate();
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const clearPermissions = usePermissionStore((state) => state.clearPermissions);
  const tenantMetadata = tenantQuery?.data;

  const [step, setStep] = useState<AuthStep>("login");
  const [emailAddress, setEmailAddress] = useState("");
  const [sessionId, setSessionId] = useState<string>();
  const [organizations, setOrganizations] = useState<UserOrganization[]>([]);
  const [organizationName, setOrganizationName] = useState<string>();
  const [authorizedRoles, setAuthorizedRoles] = useState<RoleSummary[]>([]);
  const [activeRoles, setActiveRoles] = useState<RoleSummary[]>([]);
  const [orgStepUsed, setOrgStepUsed] = useState(false);
  // A tenant login page is already scoped to one organization, so that step can never
  // appear; everywhere else three is the honest upper bound until the server answers.
  const [totalSteps, setTotalSteps] = useState(organizationSlug ? 2 : 3);

  const resetToLogin = useCallback(async () => {
    // Stepping back past the organization choice means abandoning the session that was
    // just issued — leaving it alive would put a signed-in user in front of a sign-in
    // form. The reset runs whether or not the server acknowledges the logout.
    try {
      await authService.logout();
    } catch (error) {
      handleMutationError({ error, resourceName: "Session" });
    }
    clearPermissions();
    setSessionId(undefined);
    setOrganizations([]);
    setOrganizationName(undefined);
    setAuthorizedRoles([]);
    setActiveRoles([]);
    setOrgStepUsed(false);
    setTotalSteps(organizationSlug ? 2 : 3);
    setStep("login");
  }, [clearPermissions, organizationSlug]);

  /**
   * Runs once the session is pointed at its final organization. The manifest is the
   * one source that knows both which organization the session landed in and whether
   * roles still need activating, so the branch is taken from it rather than inferred.
   */
  const enterWorkspace = useCallback(
    async (fallbackName: string | undefined, usedOrgStep: boolean) => {
      const manifest = await fetchManifest();
      setOrganizationName(manifestOrganizationName(manifest) ?? fallbackName);

      if (manifest.requiresRoleActivation) {
        setAuthorizedRoles(resolveAuthorizedRoles(manifest));
        setTotalSteps(usedOrgStep ? 3 : 2);
        setStep("role");
        return;
      }

      setActiveRoles(manifest.activeRoles);
      setStep("done");
    },
    [fetchManifest],
  );

  const handleAuthenticated = useCallback(
    async (response: LoginResponse) => {
      setEmailAddress(response.user.emailAddress);
      setSessionId(response.sessionId);

      if (organizationSlug) {
        await enterWorkspace(tenantMetadata?.organizationName, false);
        return;
      }

      const availableOrganizations = await apiService.userService.getUserOrganizations();
      setOrganizations(availableOrganizations);

      if (availableOrganizations.length > 1) {
        // The manifest for the organization the user happens to be pointed at is not
        // the one they are about to choose; dropping it keeps a stale scope from
        // briefly answering permission checks.
        clearPermissions();
        setOrgStepUsed(true);
        setTotalSteps(3);
        setStep("org");
        return;
      }

      await enterWorkspace(
        availableOrganizations[0]?.name ?? tenantMetadata?.organizationName,
        false,
      );
    },
    [clearPermissions, enterWorkspace, organizationSlug, tenantMetadata?.organizationName],
  );

  const handleOrganizationSelected = useCallback(
    async (organization: UserOrganization) => {
      setOrganizationName(organization.name);
      await enterWorkspace(organization.name, true);
    },
    [enterWorkspace],
  );

  const handleRolesActivated = useCallback(async (activated: RoleSummary[]) => {
    setActiveRoles(activated);
    setStep("done");
  }, []);

  const handleForgotPassword = useCallback((address: string) => {
    // Carry the address across so recovery starts pre-filled with what was typed.
    setEmailAddress(address);
    setStep("forgot");
  }, []);

  const handleHandoff = useCallback(() => {
    void navigate("/", { replace: true });
  }, [navigate]);

  const receipt: CredentialReceipt = {
    issued: step === "done",
    rows: [
      {
        key: "Identity",
        // Typing an address into the recovery form is not authentication, so the row
        // stays pending on both pre-session steps.
        value: step === "login" || step === "forgot" ? undefined : emailAddress,
      },
      {
        key: "Workspace",
        value: step === "role" || step === "done" ? organizationName : undefined,
      },
      {
        key: "Roles",
        value: step === "done" ? activeRoles.map((role) => role.name).join(", ") : undefined,
      },
      { key: "Session", value: step === "done" ? formatSessionId(sessionId) : undefined },
    ],
  };

  return (
    <>
      <Metadata title={t("Sign In")} description={t("Sign in to your Trenova account")} />
      <AuthShell step={step} receipt={receipt}>
        <div className="mb-1 flex items-center justify-center gap-2.5 min-[900px]:hidden">
          <img src={logoRainbow} alt="" className="size-6 object-contain" />
          <span className="text-[14px] font-semibold tracking-[-0.02em]">{t("Trenova")}</span>
        </div>

        <AuthCard stepKey={step}>
          {tenantQuery?.isLoading ? (
            <AuthCardBody>
              <p className="text-muted-foreground m-0 text-[12.5px]">
                {t("Loading organization sign-in…")}
              </p>
            </AuthCardBody>
          ) : tenantQuery?.isError ? (
            <AuthCardBody>
              <p className="text-auth-danger m-0 text-[12.5px]">
                {t("We couldn't load this tenant login page.")}
              </p>
            </AuthCardBody>
          ) : step === "forgot" ? (
            <ForgotPasswordForm defaultEmail={emailAddress} onBack={() => setStep("login")} />
          ) : step === "login" ? (
            <LoginForm
              organizationSlug={organizationSlug}
              tenantMetadata={tenantMetadata}
              stepLabel={stepLabel(1, totalSteps)}
              onAuthenticated={handleAuthenticated}
              onForgotPassword={handleForgotPassword}
            />
          ) : step === "org" ? (
            <OrganizationSelection
              organizations={organizations}
              stepLabel={stepLabel(2, totalSteps)}
              onBack={() => void resetToLogin()}
              onSelected={handleOrganizationSelected}
            />
          ) : step === "role" ? (
            <RoleSelection
              roles={authorizedRoles}
              organizationName={organizationName}
              stepLabel={stepLabel(orgStepUsed ? 3 : 2, totalSteps)}
              onBack={() => (orgStepUsed ? setStep("org") : void resetToLogin())}
              onActivated={handleRolesActivated}
            />
          ) : (
            <AuthHandoff
              organizationName={organizationName}
              roleCount={activeRoles.length}
              permissionCount={countPermissions(activeRoles)}
              onComplete={handleHandoff}
            />
          )}
        </AuthCard>

        <p className="text-subtle-foreground m-0 text-center text-[11.5px] text-balance">
          {t("By continuing you agree to our")}{" "}
          <a
            href={TERMS_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground hover:text-foreground underline underline-offset-[3px]"
          >
            {t("Terms of Service")}
          </a>{" "}
          and{" "}
          <a
            href={PRIVACY_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground hover:text-foreground underline underline-offset-[3px]"
          >
            {t("Privacy Policy")}
          </a>
          .
        </p>
      </AuthShell>
    </>
  );
}
