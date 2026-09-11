import { useT } from "@trenova/shared/i18n/use-t";
import { useReducedMotion } from "motion/react";
import { useEffect } from "react";

const HANDOFF_MS = 1600;
const HANDOFF_REDUCED_MS = 200;

/**
 * The last beat before the dashboard: the credential is issued, and the bar is the
 * receipt for the manifest fetch that already completed. It holds long enough to read
 * what the session was scoped to, then hands off.
 */
export function AuthHandoff({
  organizationName,
  roleCount,
  permissionCount,
  onComplete,
}: {
  organizationName?: string;
  roleCount: number;
  /** Undefined when the manifest carried no per-role counts; the clause is dropped. */
  permissionCount?: number;
  onComplete: () => void;
}) {
  const t = useT();

  const prefersReducedMotion = useReducedMotion();

  useEffect(() => {
    const timer = setTimeout(onComplete, prefersReducedMotion ? HANDOFF_REDUCED_MS : HANDOFF_MS);
    return () => clearTimeout(timer);
  }, [onComplete, prefersReducedMotion]);

  return (
    <div className="flex flex-col items-center gap-4 px-6 py-[38px]" role="status">
      <span className="auth-ring-pop border-border relative grid size-11 place-items-center rounded-full border">
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path className="auth-check-draw" d="M5 12.5l4.5 4.5L19 7" />
        </svg>
      </span>
      <div className="text-center">
        <div className="text-[14px] font-[550] tracking-[-0.01em]">
          {organizationName ? `Entering ${organizationName}` : "Entering Trenova"}
        </div>
        <div className="text-subtle-foreground font-table mt-1.5 text-[11.5px]">
          {t("{0} role{1}{2}", roleCount, roleCount === 1 ? "" : "s", permissionCount === undefined
            ? null
            : ` · ${permissionCount.toLocaleString()} permission${permissionCount === 1 ? "" : "s"}`)}
        </div>
      </div>
      <div className="bg-border-2 h-0.5 w-full overflow-hidden rounded-sm">
        <i className="auth-progress bg-foreground block h-full" />
      </div>
    </div>
  );
}
