import { LifeBuoy01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef } from "react";
import { Link, useLocation } from "react-router";
import { useCurrentSupportSession, useStaffProfile } from "../../hooks/use-support-session";
import { SUPPORT_CONSOLE_PATH } from "../../lib/support-access";
import { SupportSessionBanner, sessionEndedPath } from "./support-session-banner";

const DEFAULT_ELEVATION_MINUTES = 30;

/**
 * What Trenova support sees above the app. Inside a support session it is the session
 * banner; for platform staff in their own organization it is a quiet line linking the
 * support console. Everyone else, and every self-hosted install, sees nothing.
 */
export function SupportAppBanner() {
  const t = useT();
  const location = useLocation();
  const profile = useStaffProfile();
  const isStaff = profile.data?.isStaff ?? false;
  const current = useCurrentSupportSession(isStaff);
  const wasActive = useRef(false);

  const active = current.data?.active ? current.data.session : null;
  const endedReason = current.data?.endedReason ?? "";

  useEffect(() => {
    if (active) {
      wasActive.current = true;
      return;
    }
    if (wasActive.current && current.data) {
      wasActive.current = false;
      window.location.assign(sessionEndedPath(endedReason));
    }
  }, [active, current.data, endedReason]);

  if (!isStaff) {
    return null;
  }

  if (active) {
    return (
      <SupportSessionBanner
        session={active}
        fetchedAt={Math.floor(current.dataUpdatedAt / 1000)}
        elevationMinutes={profile.data?.elevationMinutes ?? DEFAULT_ELEVATION_MINUTES}
      />
    );
  }

  if (location.pathname === SUPPORT_CONSOLE_PATH) {
    return null;
  }

  return (
    <div
      data-testid="staff-strip"
      className="text-muted-foreground bg-card flex min-h-7 shrink-0 items-center gap-2 border-b px-4 py-1 text-xs"
    >
      <LifeBuoy01Icon className="size-3.5 shrink-0" aria-hidden="true" />
      <span>{t("Trenova staff")}</span>
      <span className="flex-1" />
      <Link to={SUPPORT_CONSOLE_PATH} className="text-foreground font-medium underline-offset-[3px] hover:underline">
        {t("Support console")}
      </Link>
    </div>
  );
}
