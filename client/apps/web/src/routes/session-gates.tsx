import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import type { OrganizationSetup } from "@trenova/shared/types/user";
import { AuthShell } from "./auth/_components/auth-shell";
import { ChangePasswordForm } from "./auth/_components/change-password-form";
import { OrganizationSetupStep } from "./auth/_components/organization-setup-form";
import { RoleSelection, resolveAuthorizedRoles } from "./auth/_components/role-selection";

/**
 * The safety net for sessions that reach the app still needing roles activated — an SSO
 * callback that lands on "/", or an organization switch made from inside the app, both
 * of which bypass the sign-in flow's own role step. It reuses that step, in the same
 * frame, so the two never drift apart.
 */
export function RoleActivationGate({ manifest }: { manifest: PermissionManifest }) {
  const t = useT();

  const authorizedRoles = resolveAuthorizedRoles(manifest);
  const organizationName = manifest.availableOrgs.find(
    (org) => org.id === manifest.organizationId,
  )?.name;

  return (
    <>
      <Metadata
        title={t("Select roles")}
        description={t("Choose the roles to activate for this session")}
      />
      <AuthShell screenKey="role">
        <RoleSelection
          roles={authorizedRoles}
          organizationName={organizationName}
          stepLabel="Session scope"
        />
      </AuthShell>
    </>
  );
}

/**
 * Stands in front of the app when the account owes a password change — set by an
 * administrator, or on a freshly provisioned account.
 *
 * This is the visible half of a rule the API already enforces: a session with the flag
 * set is refused everything but the change-password endpoint, so there is nothing to
 * skip to. The gate exists so the user is told why, not to be the thing stopping them.
 */
export function PasswordChangeGate() {
  const t = useT();

  return (
    <>
      <Metadata title={t("Change password")} description={t("Choose a new password to continue")} />
      <AuthShell screenKey="change-password">
        <ChangePasswordForm />
      </AuthShell>
    </>
  );
}

/**
 * Stands in front of the app while the session's organization lacks the details the
 * product needs: an organization a new instance's bootstrap made has only its name.
 *
 * The server says whether it is complete on every session check and refuses to save an
 * organization without them, so a page reached around this gate still cannot store one.
 * The gate exists so the first administrator fills them in before anything else asks.
 */
export function OrganizationSetupGate({ setup }: { setup: OrganizationSetup }) {
  const t = useT();

  return (
    <>
      <Metadata
        title={t("Set up your organization")}
        description={t("Add your company's address and identifiers to continue")}
      />
      <AuthShell screenKey="organization-setup">
        <OrganizationSetupStep setup={setup} />
      </AuthShell>
    </>
  );
}
