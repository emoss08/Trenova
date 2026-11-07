import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import type { DispatchBoardSummary } from "@/lib/graphql/dispatch-console";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import type { ReactNode } from "react";
import type { UrgencyBucket } from "./dispatch-vocabulary";

// A bar across the workspace, as the design draws it, rather than a card in a page.
const STRIP_CLASS = "rounded-none border-x-0 border-t-0";

// The strip's tone draws the severity dot, so a label carries only its words (and Nova's
// badge where Nova is named).
function Label({ children }: { children: ReactNode }) {
  return <span className="inline-flex items-center gap-1.5">{children}</span>;
}

/**
 * The figures a dispatcher is measured on, each one a way into the queue: the groups it
 * counts, Nova's waiting proposals, the exceptions tab and the open drivers.
 */
export function SummaryStrip({
  summary,
  proposalsWaiting,
  exceptions,
  isPending,
  isError,
  urgencyFocus,
  onUrgencyFocus,
  onShowExceptions,
  onShowOpenDrivers,
  showsDrivers,
}: {
  summary: DispatchBoardSummary | undefined;
  /** Proposals waiting on the moves the board holds. */
  proposalsWaiting: number;
  /** Open dispatch exceptions; null until they have been read. */
  exceptions: { count: number; more: boolean } | null;
  isPending: boolean;
  /** The board could not be read, so there are no figures to show. */
  isError: boolean;
  urgencyFocus: UrgencyBucket | null;
  onUrgencyFocus: (bucket: UrgencyBucket | null) => void;
  onShowExceptions: () => void;
  onShowOpenDrivers: () => void;
  showsDrivers: boolean;
}) {
  const t = useT();

  if (isError && !summary) {
    // A failed read is never drawn as figures still loading: each one says it is unknown.
    const unknown = [
      t("Needs a driver"),
      t("Late"),
      t("Due now"),
      t("Proposals waiting"),
      t("Exceptions"),
      t("Open drivers"),
    ];
    return (
      <KpiStrip aria-label={t("Dispatch figures")} className={STRIP_CLASS}>
        {unknown.map((label) => (
          <KpiStripItem
            key={label}
            label={<Label>{label}</Label>}
            value={<span className="text-muted-foreground">—</span>}
            sub={t("not updated")}
          />
        ))}
      </KpiStrip>
    );
  }

  if (isPending || !summary) {
    return <KpiStripSkeleton count={6} className={STRIP_CLASS} />;
  }

  const toggleFocus = (bucket: UrgencyBucket) =>
    onUrgencyFocus(urgencyFocus === bucket ? null : bucket);

  return (
    <KpiStrip aria-label={t("Dispatch figures")} className={STRIP_CLASS}>
      <KpiStripItem
        label={<Label>{t("Needs a driver")}</Label>}
        hint={t("Moves in the window with no driver. Shows every group.")}
        tone={summary.uncoveredMoves > 0 ? "warning" : undefined}
        onClick={() => onUrgencyFocus(null)}
        value={<span className="tabular-nums">{summary.uncoveredMoves.toLocaleString()}</span>}
        sub={t("{0} already covered", summary.coveredMoves.toLocaleString())}
      />
      <KpiStripItem
        label={<Label>{t("Late")}</Label>}
        hint={t("Uncovered, and the window closed without an arrival. Shows only the Late group.")}
        tone={summary.lateMoves > 0 ? "danger" : undefined}
        active={urgencyFocus === "Late"}
        onClick={
          summary.lateMoves > 0 || urgencyFocus === "Late" ? () => toggleFocus("Late") : undefined
        }
        value={<span className="tabular-nums">{summary.lateMoves.toLocaleString()}</span>}
        sub={t("window closed")}
      />
      <KpiStripItem
        label={<Label>{t("Due now")}</Label>}
        hint={t(
          "Uncovered, with the window open or opening within 4 hours. Shows only the Due now group.",
        )}
        tone={summary.atRiskMoves > 0 ? "warning" : undefined}
        active={urgencyFocus === "Now"}
        onClick={
          summary.atRiskMoves > 0 || urgencyFocus === "Now" ? () => toggleFocus("Now") : undefined
        }
        value={<span className="tabular-nums">{summary.atRiskMoves.toLocaleString()}</span>}
        sub={t("open or opens within 4 hours")}
      />
      <KpiStripItem
        label={
          <Label>
            <NovaBadge size="xs" />
            {t("Proposals waiting")}
          </Label>
        }
        hint={t("Moves where Nova picked a driver and is waiting on a decision.")}
        value={<span className="tabular-nums">{proposalsWaiting.toLocaleString()}</span>}
        sub={t("Nova picked a driver")}
      />
      <KpiStripItem
        label={<Label>{t("Exceptions")}</Label>}
        hint={t(
          "Covered moves that slipped: late, a missed stop, no check call, or out of carriers.",
        )}
        tone={exceptions && exceptions.count > 0 ? "danger" : undefined}
        onClick={onShowExceptions}
        value={
          <span className="tabular-nums">
            {exceptions === null
              ? "—"
              : exceptions.more
                ? t("{0}+", exceptions.count)
                : exceptions.count.toLocaleString()}
          </span>
        }
        sub={t("covered, off plan")}
      />
      <KpiStripItem
        label={<Label>{t("Open drivers")}</Label>}
        hint={t("Available drivers holding no work.")}
        tone={summary.unseatedDrivers > 0 ? "info" : undefined}
        onClick={showsDrivers && summary.unseatedDrivers > 0 ? onShowOpenDrivers : undefined}
        value={<span className="tabular-nums">{summary.unseatedDrivers.toLocaleString()}</span>}
        sub={t("of {0} available", summary.availableDrivers.toLocaleString())}
      />
    </KpiStrip>
  );
}
