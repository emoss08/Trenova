import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import { AuthShell } from "./auth/_components/auth-shell";
import { ChangePasswordForm } from "./auth/_components/change-password-form";
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
