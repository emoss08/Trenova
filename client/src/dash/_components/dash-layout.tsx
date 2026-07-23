import { useUnreadNotificationCount } from "@/hooks/use-notifications";
import { fetchMyPortalProfile } from "@/lib/graphql/driver-portal";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { BellIcon, HouseIcon, ReceiptTextIcon, TruckIcon, WalletIcon } from "lucide-react";
import { m } from "motion/react";
import { NavLink, Outlet, useLocation, Link } from "react-router";
import { useDashRealtime } from "./use-dash-realtime";

const tabs = [
  { to: "/dash", label: "Home", icon: HouseIcon, end: true },
  { to: "/dash/loads", label: "Loads", icon: TruckIcon, end: false },
  { to: "/dash/pay", label: "Pay", icon: ReceiptTextIcon, end: false },
  { to: "/dash/money", label: "Money", icon: WalletIcon, end: false },
] as const;

export function useDashProfile() {
  return useQuery({
    queryKey: ["dash-profile"],
    queryFn: fetchMyPortalProfile,
    staleTime: 5 * 60 * 1000,
  });
}

function NotificationBell() {
  const { data: unreadCount } = useUnreadNotificationCount();
  const count = unreadCount ?? 0;

  return (
    <NavLink
      to="/dash/notifications"
      aria-label={count > 0 ? `Notifications (${count} unread)` : "Notifications"}
      className={({ isActive }) =>
        cn(
          "relative flex size-8 items-center justify-center rounded-full border border-border bg-muted text-muted-foreground",
          isActive && "border-primary text-foreground",
        )
      }
    >
      <BellIcon className="size-4" />
      {count > 0 ? (
        <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-2xs font-semibold text-primary-foreground">
          {count > 99 ? "99+" : count}
        </span>
      ) : null}
    </NavLink>
  );
}

export function DashLayout() {
  const location = useLocation();
  const { data: profile } = useDashProfile();
  useDashRealtime();

  const initials = profile
    ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}`.toUpperCase()
    : "";

  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <header className="sticky top-0 z-20 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-14 w-full max-w-lg items-center justify-between px-4">
          <div className="flex items-baseline gap-2">
            <Link to="/dash" className="text-lg font-semibold tracking-tight">
              Dash
            </Link>
            {profile?.organizationName ? (
              <span className="max-w-40 truncate text-xs text-muted-foreground">
                {profile.organizationName}
              </span>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <NavLink
              to="/dash/profile"
              aria-label="Profile"
              className={({ isActive }) =>
                cn(
                  "flex size-8 items-center justify-center rounded-full border border-border bg-muted text-xs font-semibold text-muted-foreground",
                  isActive && "border-primary text-foreground",
                )
              }
            >
              {initials || "•"}
            </NavLink>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-lg flex-1 px-4 pt-4 pb-24">
        <m.div
          key={location.pathname}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.18, ease: "easeOut" }}
        >
          <Outlet />
        </m.div>
      </main>

      <nav
        aria-label="Primary"
        className="fixed inset-x-0 bottom-0 z-20 border-t border-border bg-background/90 pb-[env(safe-area-inset-bottom)] backdrop-blur-md"
      >
        <div className="mx-auto grid w-full max-w-lg grid-cols-4">
          {tabs.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              end={tab.end}
              className={({ isActive }) =>
                cn(
                  "relative flex flex-col items-center gap-1 py-2.5 text-[11px] font-medium text-muted-foreground",
                  isActive && "text-foreground",
                )
              }
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
                  {tab.label}
                </>
              )}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
