import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Ic, type IcName } from "../kit/ic";

type FProps = {
  label: ReactNode;
  hint?: ReactNode;
  /** The field's error, shown in place of the hint. */
  error?: string;
  htmlFor?: string;
  /** Beside the label, such as a count. */
  aside?: ReactNode;
  className?: string;
  children: ReactNode;
};

/**
 * One field of an editor: the label and its hint in a fixed column, the control beside
 * it. Every editor lays its fields out with this, so they line up from one sheet to the
 * next.
 */
export function F({ label, hint, error, htmlFor, aside, className, children }: FProps) {
  return (
    <div className={cn("f", error && "err", className)}>
      <div className="f-l">
        <label htmlFor={htmlFor}>{label}</label>
        {aside}
      </div>
      {children}
      {(error || hint) && <p className="f-h">{error || hint}</p>}
    </div>
  );
}

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

/** A note inside a section that says what a change will do. */
export function Callout({ tone = "i", icon, action, children }: CalloutProps) {
  return (
    <div className={cn("cal", tone)} role={tone === "w" || tone === "d" ? "alert" : "status"}>
      <Ic n={icon ?? CALLOUT_ICON[tone]} s={13} />
      <div>{children}</div>
      {action}
    </div>
  );
}
