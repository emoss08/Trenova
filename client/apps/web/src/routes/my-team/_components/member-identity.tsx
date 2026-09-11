import { useT } from "@trenova/shared/i18n/use-t";
import type { TeamMemberRow } from "@/lib/graphql/org-structure";
import { memberHealth } from "@/lib/my-team";
import { Avatar, AvatarBadge, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import type { WorkerHealthMeta } from "@trenova/shared/lib/worker-health";

export function memberHref(workerId: string): string {
  return `/hr/workers?panelEntityId=${workerId}&panelType=edit`;
}

type MemberAvatarProps = {
  member: Pick<TeamMemberRow, "name" | "fleetColor" | "fleetCode">;
  size?: "sm" | "default";
  className?: string;
};

/**
 * Initials in the design system's avatar, with the terminal's own colour as
 * the one badge. The colour is the fleet's, not ours, which is why it is the
 * only raw colour on the row.
 */
export function MemberAvatar({ member, size = "default", className }: MemberAvatarProps) {
  return (
    <Avatar size={size} className={className}>
      <AvatarFallback className="text-xs font-medium">
        {getNameInitials(member.name)}
      </AvatarFallback>
      {member.fleetColor ? (
        <AvatarBadge
          aria-hidden
          title={member.fleetCode || undefined}
          style={{ backgroundColor: member.fleetColor }}
        />
      ) : null}
    </Avatar>
  );
}

type MemberIdentityProps = {
  member: TeamMemberRow;
  size?: "sm" | "default";
  className?: string;
};

export function MemberIdentity({ member, size = "default", className }: MemberIdentityProps) {
  const t = useT();

  const left = member.status !== "Active";
  const secondary = [member.positionTitle, member.fleetCode].filter(Boolean).join(" · ");

  return (
    <div className={cn("flex flex-row min-w-0 items-center gap-2.5", className)}>
      <MemberAvatar member={member} size={size} />
      <div className="flex min-w-0 flex-col leading-tight">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="truncate text-sm font-medium">{member.name}</span>
          {left ? (
            <Badge variant="inactive" className="text-2xs h-4 px-1">
              {t("Left")}
            </Badge>
          ) : null}
        </span>
        {secondary ? (
          <span className="text-muted-foreground truncate text-xs">{secondary}</span>
        ) : null}
      </div>
    </div>
  );
}

type HealthDotProps = {
  category: string;
  meta: WorkerHealthMeta;
};

function HealthDot({ category, meta }: HealthDotProps) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "inline-flex items-center gap-1.5 text-xs whitespace-nowrap",
              meta.good ? "text-muted-foreground" : "text-foreground font-medium",
            )}
          />
        }
      >
        <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", meta.dotClass)} />
        <span className="sr-only">{category}: </span>
        {meta.label}
      </TooltipTrigger>
      <TooltipContent>
        {category} · {meta.label}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * Compliance, training and safety in one glance. Colour lives on the dot only;
 * a state that needs somebody's eye is set in the foreground weight instead.
 */
export function HealthTrio({ member, className }: { member: TeamMemberRow; className?: string }) {
  const health = memberHealth(member);
  return (
    <div className={cn("flex items-center gap-3", className)}>
      <HealthDot category="Compliance" meta={health.compliance} />
      <HealthDot category="Training" meta={health.training} />
      <HealthDot category="Safety" meta={health.safety} />
    </div>
  );
}
