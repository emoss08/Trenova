import { credentialHealthMeta, shortDaysUntil } from "../lib/credential";
import { cn } from "../lib/utils";
import type { CredentialHealth } from "../types/worker-credential";
import { Badge } from "./ui/badge";

type CredentialHealthBadgeProps = {
  health: CredentialHealth;
  /** When provided, expiring/expired badges show the day count too. */
  daysUntilExpiry?: number | null;
  className?: string;
};

export function CredentialHealthBadge({
  health,
  daysUntilExpiry,
  className,
}: CredentialHealthBadgeProps) {
  const meta = credentialHealthMeta(health);
  const showDays = daysUntilExpiry != null && (health === "ExpiringSoon" || health === "Expired");
  return (
    <Badge variant={meta.badgeVariant} className={cn("gap-1.5 whitespace-nowrap", className)}>
      <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", meta.dotClass)} />
      {meta.label}
      {showDays ? (
        <span className="tabular-nums opacity-80">· {shortDaysUntil(daysUntilExpiry)}</span>
      ) : null}
    </Badge>
  );
}
