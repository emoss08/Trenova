import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Ic, type IcName } from "../kit/ic";

export type CalloutTone = "i" | "w" | "d" | "k";

const CALLOUT_ICON: Record<CalloutTone, IcName> = {
  i: "sparkle",
  w: "warn",
  d: "alert",
  k: "check",
};

type CalloutProps = {
  tone?: CalloutTone;
  icon?: IcName;
  /** A button at the end of the callout. */
  action?: ReactNode;
  children: ReactNode;
};

const CALLOUT_TONE: Record<CalloutTone, { box: string; icon: string }> = {
  i: { box: "bg-brand-subtle", icon: "text-brand-subtle-foreground" },
  w: { box: "bg-warning-subtle", icon: "text-warning-subtle-foreground" },
  d: { box: "bg-danger-subtle", icon: "text-danger-subtle-foreground" },
  k: { box: "bg-success-subtle", icon: "text-success-subtle-foreground" },
};

/** A note inside a section that says what a change will do. */
export function Callout({ tone = "i", icon, action, children }: CalloutProps) {
  return (
    <div
      className={cn(
        "text-foreground mt-3 flex items-start gap-2.5 rounded-lg px-3 py-2.5 text-base leading-normal",
        CALLOUT_TONE[tone].box,
      )}
      role={tone === "w" || tone === "d" ? "alert" : "status"}
    >
      <span className={cn("mt-0.5 shrink-0", CALLOUT_TONE[tone].icon)}>
        <Ic n={icon ?? CALLOUT_ICON[tone]} s={13} />
      </span>
      <div className="min-w-0 flex-1">{children}</div>
      {action && <div className="shrink-0 self-center">{action}</div>}
    </div>
  );
}
