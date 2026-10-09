import { CheckIcon, Mail01Icon } from "@trenova/shared/components/icons";
import type { CSSProperties, ReactNode } from "react";
import { AuthHeading } from "./auth-primitives";

/**
 * The layout every finished screen shares — "Welcome back", "You're in", "Check your
 * inbox": a mark, the heading and its lead, whatever the screen adds, then its actions.
 * Each screen passes what differs; this holds no copy of its own.
 */
export function AuthSuccess({
  mark,
  title,
  lead,
  children,
  actions,
}: {
  mark: ReactNode;
  title: ReactNode;
  lead?: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div role="status" className="auth-enter flex flex-col items-start gap-3">
      <div className="mb-2">{mark}</div>
      <div className="-mb-4 w-full">
        <AuthHeading title={title}>{lead}</AuthHeading>
      </div>
      {children}
      {actions ? <div className="mt-1 flex w-full flex-col items-start gap-4">{actions}</div> : null}
    </div>
  );
}

/** An ink disc whose check strokes in. */
export function AuthCheckMark() {
  return (
    <span className="auth-pop auth-draw bg-ink text-ink-foreground grid size-10 place-items-center rounded-full">
      <CheckIcon className="size-[15px]" strokeWidth={2.2} />
    </span>
  );
}

const MAIL_DRAW_STYLE = { "--auth-draw": 100 } as CSSProperties;

/** An envelope on the field fill that strokes in, then drifts while the screen waits. */
export function AuthMailMark() {
  return (
    <span
      style={MAIL_DRAW_STYLE}
      className="auth-float auth-draw bg-auth-field text-foreground border-border-strong grid size-10 place-items-center rounded-full border"
    >
      <Mail01Icon className="size-[17px]" strokeWidth={1.6} />
    </span>
  );
}

/** A dashed aside under a lead: what to try when the expected thing has not happened. */
export function AuthHint({ children }: { children: ReactNode }) {
  return (
    <p className="text-muted-foreground border-border-strong m-0 mb-2 w-full rounded-[10px] border border-dashed px-3 py-2.5 text-sm text-pretty">
      {children}
    </p>
  );
}

/** A thin bar that fills over `durationMs`, for a screen that moves on by itself. */
export function AuthProgress({ durationMs }: { durationMs: number }) {
  return (
    <div className="bg-border my-3 h-0.5 w-full overflow-hidden rounded-sm">
      <i
        className="auth-progress bg-brand block h-full"
        style={{ "--auth-progress-duration": `${durationMs}ms` } as CSSProperties}
      />
    </div>
  );
}
