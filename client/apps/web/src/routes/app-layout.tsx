import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import { SidebarLayout } from "@/components/navigation";
import { usePermissionPolling } from "@/hooks/use-permission-polling";
import { useRealtimeConnection } from "@/hooks/use-realtime-connection";
import { useUserDatePreferenceKey } from "@trenova/shared/hooks/use-user-date-preferences";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import { Outlet } from "react-router";
import { AuthCard } from "./auth/_components/auth-card";
import type { CredentialReceipt } from "./auth/_components/auth-panel";
import { AuthShell } from "./auth/_components/auth-shell";
import { ChangePasswordForm } from "./auth/_components/change-password-form";
import { RoleSelection, resolveAuthorizedRoles } from "./auth/_components/role-selection";

/**
 * The safety net for sessions that reach the app still needing roles activated — an SSO
 * callback that lands on "/", or an organization switch made from inside the app, both
 * of which bypass the sign-in flow's own role step. It reuses that step, in the same
 * frame, so the two never drift apart.
 *
 * There is no session id to show here: the browser only ever sees one at login, so the
 * receipt carries three rows instead of four rather than a row that can never fill.
 */
function RoleActivationGate({ manifest }: { manifest: PermissionManifest }) {
  const t = useT();

  const user = useAuthStore((state) => state.user);
  const authorizedRoles = resolveAuthorizedRoles(manifest);
  const organizationName = manifest.availableOrgs.find(
    (org) => org.id === manifest.organizationId,
  )?.name;

  const receipt: CredentialReceipt = {
    issued: false,
    rows: [
      { key: "Identity", value: user?.emailAddress },
      { key: "Workspace", value: organizationName },
      { key: "Roles", value: undefined },
    ],
  };

  return (
    <>
      <Metadata title={t("Select roles")} description={t("Choose the roles to activate for this session")} />
      <AuthShell step="role" receipt={receipt}>
        <AuthCard stepKey="role">
          <RoleSelection
            roles={authorizedRoles}
            organizationName={organizationName}
            stepLabel="Session scope"
          />
        </AuthCard>
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
function PasswordChangeGate() {
  const t = useT();

  const user = useAuthStore((state) => state.user);

  const receipt: CredentialReceipt = {
    issued: false,
    rows: [
      { key: "Identity", value: user?.emailAddress },
      { key: "Password", value: undefined },
    ],
  };

  return (
    <>
      <Metadata title={t("Change password")} description={t("Choose a new password to continue")} />
      <AuthShell step="login" receipt={receipt}>
        <AuthCard stepKey="change-password">
          <ChangePasswordForm />
        </AuthCard>
      </AuthShell>
    </>
  );
}

export function AppLayout() {
  usePermissionPolling();
  useRealtimeConnection();
  const manifest = usePermissionStore((state) => state.manifest);
  const user = useAuthStore((state) => state.user);
  // Timestamps are formatted from module-level preferences, so a zone or clock
  // change has to remount the page to repaint what is already on screen.
  const datePreferenceKey = useUserDatePreferenceKey();

  // Ordered ahead of the role gate: a session that cannot call anything but
  // change-password cannot activate a role either, so asking for one first would
  // strand the user against an API that refuses the request.
  if (user?.mustChangePassword) {
    return <PasswordChangeGate />;
  }

  if (manifest?.requiresRoleActivation) {
    return <RoleActivationGate key={manifest.organizationId} manifest={manifest} />;
  }

  return (
    <SidebarLayout>
      <Outlet key={datePreferenceKey} />
    </SidebarLayout>
  );
}
