import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { queryClient } from "@/lib/query-client";
import { apiService } from "@/services/api";
import { authService } from "@trenova/shared/services/auth";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import type { RoleSummary } from "@trenova/shared/types/role";
import type { TenantLoginMetadata, UserOrganization } from "@trenova/shared/types/organization";
import type { LoginResponse } from "@trenova/shared/types/user";
import type { UseQueryResult } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import { AuthHandoff } from "./auth-handoff";
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
  const [userName, setUserName] = useState<string>();
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
    setUserName(undefined);
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
      setUserName(response.user.name);

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
    // Drop anything cached by an earlier session in this tab, including one that
    // expired without signing out, before the app loads for the new one.
    queryClient.clear();
    void navigate("/", { replace: true });
  }, [navigate]);

  return (
    <>
      <Metadata title={t("Sign In")} description={t("Sign in to your Trenova account")} />
      <AuthShell screenKey={step}>
        {tenantQuery?.isLoading ? (
          <p className="text-auth-body text-muted-foreground m-0">
            {t("Loading organization sign-in…")}
          </p>
        ) : tenantQuery?.isError ? (
          <p role="alert" className="text-auth-body text-danger m-0">
            {t("We couldn't load this tenant login page.")}
          </p>
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
            userName={userName}
            organizationName={organizationName}
            roleCount={activeRoles.length}
            permissionCount={countPermissions(activeRoles)}
            onComplete={handleHandoff}
          />
        )}
      </AuthShell>
    </>
  );
}
