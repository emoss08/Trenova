import { firstNameOf } from "@/lib/onboarding-copy";
import { useT } from "@trenova/shared/i18n/use-t";
import { useReducedMotion } from "motion/react";
import { useEffect } from "react";
import { AuthCheckMark, AuthProgress, AuthSuccess } from "./auth-success";

const HANDOFF_MS = 1600;
const HANDOFF_REDUCED_MS = 200;

/**
 * The last beat before the dashboard: the session is issued, and the bar is the
 * receipt for the manifest fetch that already completed. It holds long enough to read
 * what the session was scoped to, then hands off.
 */
export function AuthHandoff({
  userName,
  organizationName,
  roleCount,
  permissionCount,
  onComplete,
}: {
  userName?: string;
  organizationName?: string;
  roleCount: number;
  /** Undefined when the manifest carried no per-role counts; the clause is dropped. */
  permissionCount?: number;
  onComplete: () => void;
}) {
  const t = useT();

  const prefersReducedMotion = useReducedMotion();
  const duration = prefersReducedMotion ? HANDOFF_REDUCED_MS : HANDOFF_MS;
  const firstName = firstNameOf(userName);

  useEffect(() => {
    const timer = setTimeout(onComplete, duration);
    return () => clearTimeout(timer);
  }, [onComplete, duration]);

  return (
    <AuthSuccess
      mark={<AuthCheckMark />}
      title={firstName ? t("Welcome back, {0}.", firstName) : t("Welcome back.")}
    >
      <p className="text-muted-foreground m-0 flex flex-wrap gap-x-1.5 font-mono text-sm">
        <span>
          {organizationName ? t("Opening {0}", organizationName) : t("Opening Trenova")}
        </span>
        <span aria-hidden="true">·</span>
        <span>
          {t(
            "{0, plural, one {# role} other {# roles}}{1}",
            roleCount,
            permissionCount === undefined
              ? null
              : ` ${t("· {0} {1, plural, one {permission} other {permissions}}", permissionCount.toLocaleString(), permissionCount)}`,
          )}
        </span>
      </p>
      <AuthProgress durationMs={duration} />
    </AuthSuccess>
  );
}
