import { Link } from "@trenova/shared/components/link";
import { useT } from "@trenova/shared/i18n/use-t";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { fetchMyPortalProfile } from "@trenova/shared/lib/graphql/driver-portal";
import { dropPushOfAnotherDriver } from "@/lib/push-owner";
import { currentProfile, useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import {
  Bell01Icon,
  ClockIcon,
  Home02Icon,
  ReceiptTextIcon,
  Truck01Icon,
  Wallet02Icon,
} from "@trenova/shared/components/icons";
import { WorkerAvatar } from "@trenova/shared/components/worker-avatar";
import { m } from "motion/react";
import { useEffect } from "react";
import { Outlet, useLocation } from "@tanstack/react-router";
import { DashIdleTimeout } from "./dash-idle-timeout";
import { useDashRealtime } from "./use-dash-realtime";

const tabs = [
  { to: "/dash", label: "Home", icon: Home02Icon, end: true },
  { to: "/dash/loads", label: "Loads", icon: Truck01Icon, end: false },
  { to: "/dash/hos", label: "HOS", icon: ClockIcon, end: false },
  { to: "/dash/pay", label: "Pay", icon: ReceiptTextIcon, end: false },
  { to: "/dash/money", label: "Money", icon: Wallet02Icon, end: false },
] as const;

export function useDashProfile() {
  return useQuery({
    queryKey: ["dash-profile"],
    queryFn: ({ signal }) => fetchMyPortalProfile({ signal }),
    staleTime: 5 * 60 * 1000,
  });
}

function NotificationBell() {
  const { data: unreadCount } = useUnreadNotificationCount("mine");
  const count = unreadCount ?? 0;
  const t = useT();

  return (
    <Link
      to="/dash/notifications"
      activeOptions={{ includeSearch: false }}
      aria-label={count > 0 ? t("Notifications ({0} unread)", count) : t("Notifications")}
      className="relative flex size-8 items-center justify-center rounded-full border border-border bg-muted text-muted-foreground data-[status=active]:border-primary data-[status=active]:text-foreground"
    >
      <Bell01Icon className="size-4" />
      {count > 0 ? (
        <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-badge bg-primary px-1 text-2xs font-semibold text-primary-foreground">
          {count > 99 ? "99+" : count}
        </span>
      ) : null}
    </Link>
  );
}

export function DashLayout() {
  const userId = useAuthStore((state) => currentProfile(state)?.id);

  useEffect(() => {
    if (userId) {
      void dropPushOfAnotherDriver(userId).catch(() => {});
    }
  }, [userId]);

  const t = useT();

  const pathname = useLocation({ select: (location) => location.pathname });
  const { data: profile } = useDashProfile();
  useDashRealtime();

  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <header className="sticky top-0 z-20 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-14 w-full max-w-lg items-center justify-between px-4">
          <div className="flex items-baseline gap-2">
            <Link to="/dash" className="text-lg font-semibold tracking-tight">
              {t("Dash")}
            </Link>
            {profile?.organizationName ? (
              <span className="max-w-40 truncate text-xs text-muted-foreground">
                {profile.organizationName}
              </span>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <Link
              to="/dash/profile"
              activeOptions={{ includeSearch: false }}
              aria-label={t("Profile")}
              className="flex size-8 items-center justify-center rounded-full ui-focus-ring data-[status=active]:ring-2 data-[status=active]:ring-primary"
            >
              <WorkerAvatar
                aria-hidden
                firstName={profile?.firstName}
                lastName={profile?.lastName}
                photo={profile?.photo}
                className="size-8"
                fallbackClassName="text-xs font-semibold"
              />
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-lg flex-1 px-4 pt-4 pb-24">
        <m.div
          key={pathname}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.18, ease: "easeOut" }}
        >
          <Outlet />
        </m.div>
      </main>

      <nav
        aria-label={t("Primary")}
        className="fixed inset-x-0 bottom-0 z-20 border-t border-border bg-background/90 pb-[env(safe-area-inset-bottom)] backdrop-blur-md"
      >
        <div className="mx-auto grid w-full max-w-lg grid-cols-5">
          {tabs.map((tab) => (
            <Link
              key={tab.to}
              to={tab.to}
              activeOptions={{ exact: tab.end, includeSearch: false }}
              className="relative flex flex-col items-center gap-1 py-2.5 text-xs font-medium text-muted-foreground data-[status=active]:text-foreground"
            >
              {({ isActive }) => (
                <>
                  {isActive ? (
                    <m.span
                      layoutId="dash-tab-indicator"
                      className="absolute top-0 h-0.5 w-8 rounded-full bg-foreground"
                    />
                  ) : null}
                  <tab.icon className="size-5" strokeWidth={isActive ? 2.2 : 1.8} />
                  {t(tab.label)}
                </>
              )}
            </Link>
          ))}
        </div>
      </nav>
      <DashIdleTimeout />
    </div>
  );
}
