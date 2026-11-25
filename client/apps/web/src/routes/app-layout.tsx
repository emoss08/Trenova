import { RouteErrorBoundary, RouteErrorEmbedding } from "@trenova/shared/components/error-boundary";
import { SidebarLayout } from "@/components/navigation";
import { SessionIdleTimeout } from "@/components/session-idle-timeout";
import { SupportSessionBanner } from "@/components/session-context/support-session-banner";
import { lazyComponent } from "@trenova/shared/lib/lazy-component";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { usePermissionPolling } from "@/hooks/use-permission-polling";
import { useRealtimeConnection } from "@/hooks/use-realtime-connection";
import { useUserDatePreferenceKey } from "@trenova/shared/hooks/use-user-date-preferences";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Suspense, type ReactNode } from "react";
import { Outlet, type ErrorComponentProps } from "@tanstack/react-router";

// Only a session that owes a password change, a role choice or its organization's details
// meets a gate, so the sign-in frame and forms they draw are read when one is shown, not by
// every session.
const loadSessionGates = () => import("./session-gates");
const PasswordChangeGate = lazyComponent(loadSessionGates, (module) => module.PasswordChangeGate);
const RoleActivationGate = lazyComponent(loadSessionGates, (module) => module.RoleActivationGate);
const OrganizationSetupGate = lazyComponent(
  loadSessionGates,
  (module) => module.OrganizationSetupGate,
);

/**
 * Everything a signed-in session owes before a page may render: the polling
 * and realtime connection it keeps open, the idle timeout that signs a quiet
 * session out, the two gates that can stand in front of it, and the remount
 * key that repaints formatted timestamps when the clock preference moves.
 *
 * It takes the outlet as an argument rather than rendering it directly
 * because the app has two frames now. Inside the sidebar, a page is
 * surrounded by chrome; at the Desk there is no chrome at all. Both owe the
 * session exactly the same things, and a session check that lived in only
 * one of them would be a session check the other could be used to skip.
 */
export function AppSession({ children }: { children: (outlet: ReactNode) => ReactNode }) {
  return (
    <>
      <SessionContent>{children}</SessionContent>
      <SessionIdleTimeout />
      <SupportSessionBanner />
    </>
  );
}

// The idle timeout stands beside the gates rather than behind them: a session left on
// the password-change or role screen is as unattended as one left on a page.
function SessionContent({ children }: { children: (outlet: ReactNode) => ReactNode }) {
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
    return (
      <Suspense fallback={<LoadingSkeleton />}>
        <PasswordChangeGate.Component />
      </Suspense>
    );
  }

  if (manifest?.requiresRoleActivation) {
    return (
      <Suspense fallback={<LoadingSkeleton />}>
        <RoleActivationGate.Component key={manifest.organizationId} manifest={manifest} />
      </Suspense>
    );
  }

  // After the role gate: whether this person may fill the organization in depends on the
  // roles the session has active.
  const organizationSetup = user?.organizationSetup;
  if (organizationSetup && !organizationSetup.complete) {
    return (
      <Suspense fallback={<LoadingSkeleton />}>
        <OrganizationSetupGate.Component
          key={user?.currentOrganizationId}
          setup={organizationSetup}
        />
      </Suspense>
    );
  }

  return children(<Outlet key={datePreferenceKey} />);
}

/**
 * The welcome wizard's frame: the session's gates and connections, and nothing else.
 * It wears no sidebar, because nothing behind it is useful until the organization has
 * a profile.
 */
export function OnboardingLayout() {
  return <AppSession>{(outlet) => outlet}</AppSession>;
}

export function AppLayout() {
  return (
    <AppSession>
      {(outlet) => (
        <SidebarLayout>
          <RouteErrorEmbedding value>{outlet}</RouteErrorEmbedding>
        </SidebarLayout>
      )}
    </AppSession>
  );
}

// AppErrorLayout is AppLayout's error component: what the session gate threw, or the layout
// itself. It draws the same frame around the failure rather than replacing the whole window.
// It skips the session's polling and gates: the failure may be in them, and if the frame
// itself throws the root's error component takes over.
export function AppErrorLayout(props: ErrorComponentProps) {
  return (
    <SidebarLayout>
      <RouteErrorBoundary {...props} embedded />
    </SidebarLayout>
  );
}
