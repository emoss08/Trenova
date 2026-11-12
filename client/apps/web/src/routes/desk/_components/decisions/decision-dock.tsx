import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { QueueKey } from "../queue/queue-bar";
import { MAX_BATCH } from "./batch-review-model";

export type DecisionDockSummary = {
  count: number;
  /** What the ticked changes are, in a word: one kind's name, or that they are several kinds. */
  what: string;
  approvable: number;
  flagged: number;
  blocked: number;
  reading: boolean;
};

/**
 * Deciding the ticked changes together, from the foot of the list where they
 * were ticked. It is not there until a change is ticked; then it rises, and
 * sinks when the last is let go, keeping what it said while it leaves.
 */
export function DecisionDock({
  summary,
  canDecide,
  busy,
  onClear,
  onDecline,
  onReview,
}: {
  summary: DecisionDockSummary;
  canDecide: boolean;
  busy: boolean;
  onClear: () => void;
  onDecline: () => void;
  onReview: () => void;
}) {
  const t = useT();
  const open = summary.count > 0;
  const key = JSON.stringify(summary);
  const [kept, setKept] = useState({ key, summary });
  if (open && kept.key !== key) {
    setKept({ key, summary });
  }
  const shown = open ? summary : kept.summary;
  if (shown.count === 0) {
    return null;
  }

  return (
    <div
      role="region"
      aria-label={t("Selected changes")}
      inert={!open}
      className={cn(
        // The footing under the dock is the list's own ground, so no row shows past its edge.
        "absolute inset-x-0 bottom-0 z-10 bg-linear-to-t from-dsk-canvas from-70% to-transparent px-2.5 pt-5 pb-2.5 transition-[translate,opacity] duration-400 ease-(--dk-settle) motion-reduce:transition-none starting:translate-y-full starting:opacity-0",
        open ? "translate-y-0 opacity-100" : "pointer-events-none translate-y-full opacity-0",
      )}
    >
      <div className="flex flex-col gap-2.25 rounded-xl bg-dsk-raised p-2.5 shadow-(--dsk-lift-hi)">
        <div className="flex items-center gap-2 px-0.5 text-sm">
          <b className="font-semibold text-dsk-fg tabular-nums">
            {t("{0, plural, one {# selected} other {# selected}}", shown.count)}
          </b>
          <span className="min-w-0 truncate text-dsk-subtle">{shown.what}</span>
          <Button
            variant="bare"
            size="bare"
            className="ml-auto gap-1.5 rounded-md px-1.5 py-0.75 text-xs text-dsk-subtle hover:bg-dsk-hover hover:text-dsk-fg"
            onClick={onClear}
          >
            {t("Clear")}
            <QueueKey>Esc</QueueKey>
          </Button>
        </div>
        {shown.reading ? (
          <DockNote tone="quiet">{t("Reading every change…")}</DockNote>
        ) : shown.blocked > 0 ? (
          <DockNote tone="danger">
            {t(
              "{0, plural, one {# can't be approved as it stands and is left out of an approval.} other {# can't be approved as they stand and are left out of an approval.}}",
              shown.blocked,
            )}
          </DockNote>
        ) : shown.flagged > 0 ? (
          <DockNote tone="warning">
            {t(
              "{0, plural, one {# of these has something to look at first.} other {# of these have something to look at first.}}",
              shown.flagged,
            )}
          </DockNote>
        ) : shown.count >= MAX_BATCH ? (
          <DockNote tone="quiet">{t("Up to {0} can be decided together.", MAX_BATCH)}</DockNote>
        ) : null}
        {canDecide ? (
          <div className="grid grid-cols-[1fr_1.5fr] gap-1.5">
            <Button variant="secondary" disabled={busy} onClick={onDecline}>
              {t("Decline {0}", shown.count)}
            </Button>
            <Button disabled={busy || shown.reading || shown.approvable === 0} onClick={onReview}>
              {t("Review and approve {0}", shown.approvable)}
            </Button>
          </div>
        ) : (
          <DockNote tone="quiet">{t("You can read these, but not decide them.")}</DockNote>
        )}
      </div>
    </div>
  );
}

function DockNote({ tone, children }: { tone: "quiet" | "warning" | "danger"; children: string }) {
  return (
    <p
      className={cn(
        "flex items-start gap-1.75 px-0.5 text-xs",
        tone === "danger"
          ? "text-danger-foreground"
          : tone === "warning"
            ? "text-warning-foreground"
            : "text-dsk-subtle",
      )}
    >
      {tone !== "quiet" && (
        <span
          aria-hidden
          className={cn(
            "mt-1.25 size-1.5 shrink-0 rounded-full",
            tone === "danger" ? "bg-danger" : "bg-warning",
          )}
        />
      )}
      {children}
    </p>
  );
}
