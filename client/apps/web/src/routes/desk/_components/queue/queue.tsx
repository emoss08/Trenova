import { CheckIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { QueueLeave } from "./use-queue-cursor";

/**
 * The two-column room every Desk queue is worked in: the queue down the left,
 * the one item in front of the person in the middle, and the bar that acts on
 * it underneath. Decisions and the Watchtower are both drawn in it, so the two
 * read as one way of working rather than two screens that happen to look alike.
 */
export function QueueLayout({
  label,
  aside,
  className,
  children,
}: {
  label: string;
  aside: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("dk-dc2", className)}>
      <aside className="dk-dc2-q" aria-label={label}>
        {aside}
      </aside>
      <main className="dk-dc2-main">{children}</main>
    </div>
  );
}

/**
 * The queue's title, how much of it is behind the person, and the bar that
 * fills as they clear it.
 */
export function QueueHeader({
  title,
  progressLabel,
  done,
  total,
}: {
  title: string;
  progressLabel: string;
  done: number;
  total: number;
}) {
  const share = total > 0 ? Math.min(1, done / total) : 0;

  return (
    <>
      <div className="dk-dc2-qh">
        <b>{title}</b>
        <span>{progressLabel}</span>
      </div>
      <div
        className="dk-dc2-prog"
        role="progressbar"
        aria-label={progressLabel}
        aria-valuemin={0}
        aria-valuemax={total}
        aria-valuenow={Math.min(done, total)}
      >
        <i style={{ transform: `scaleX(${share})` }} />
      </div>
    </>
  );
}

/** The card the queue's current item is read in, and how it leaves. */
export function QueueCard({
  leaving,
  className,
  children,
}: {
  leaving: QueueLeave | null;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("dk-dc2-card", leaving && `dk-out-${leaving}`, className)}>{children}</div>
  );
}

/** What the middle shows once nothing is left to work. */
export function QueueClear({ title, body }: { title: string; body: string }) {
  return (
    <div className="dk-dc2-clear">
      <span className="dk-dc2-cic">
        <CheckIcon className="size-5.5" strokeWidth={2.2} />
      </span>
      <b>{title}</b>
      <span>{body}</span>
    </div>
  );
}

/** A quiet line in the middle while the queue is first read. */
export function QueueReading({ label }: { label: string }) {
  return (
    <div className="dk-dc2-clear" aria-busy="true">
      <span className="dk-dc2-qe">{label}</span>
    </div>
  );
}
