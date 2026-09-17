import type { CarrierIntelReviewState } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, type ReactNode } from "react";
import { FreshnessIndicator, type FreshnessIndicatorProps } from "./freshness-indicator";
import { ReviewStateLabel } from "./review-state-label";
import { RiskLabel } from "./status-dot";

export type IntelSnapshotHeaderProps = {
  label: string;
  title?: ReactNode;
  riskLevel: string | null | undefined;
  reviewState?: CarrierIntelReviewState | null;
  reviewedAt?: number | null;
  blockingCount?: number;
  openEventCount?: number;
  freshness?: FreshnessIndicatorProps | null;
  meta?: ReactNode;
  note?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export function IntelSnapshotHeader({
  label,
  title,
  riskLevel,
  reviewState,
  reviewedAt,
  blockingCount = 0,
  openEventCount,
  freshness,
  meta,
  note,
  actions,
  children,
  className,
}: IntelSnapshotHeaderProps) {
  const t = useT();

  const items: { id: string; node: ReactNode }[] = [
    { id: "risk", node: <RiskLabel level={riskLevel} /> },
    reviewState && reviewState !== "None"
      ? { id: "review", node: <ReviewStateLabel state={reviewState} reviewedAt={reviewedAt} /> }
      : null,
    blockingCount > 0
      ? {
          id: "blockers",
          node: (
            <span className="text-foreground tabular-nums">
              {t("{0, plural, one {# blocker} other {# blockers}}", blockingCount)}
            </span>
          ),
        }
      : null,
    openEventCount !== undefined && openEventCount > 0
      ? {
          id: "events",
          node: (
            <span className="tabular-nums">
              {t("{0, plural, one {# open change} other {# open changes}}", openEventCount)}
            </span>
          ),
        }
      : null,
    freshness ? { id: "freshness", node: <FreshnessIndicator {...freshness} /> } : null,
    meta ? { id: "meta", node: <span className="truncate">{meta}</span> } : null,
  ].filter((item): item is { id: string; node: ReactNode } => item !== null);

  return (
    <section aria-label={label} className={cn("flex flex-col gap-3 border-b pb-4", className)}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          {title ? <h3 className="truncate text-sm font-medium">{title}</h3> : null}
          <div className="text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs">
            {items.map((item, index) => (
              <Fragment key={item.id}>
                {index > 0 ? <span aria-hidden>·</span> : null}
                {item.node}
              </Fragment>
            ))}
          </div>
          {note ? <p className="text-muted-foreground text-xs">{note}</p> : null}
        </div>
        {actions ? <div className="flex shrink-0 items-center gap-1.5">{actions}</div> : null}
      </div>
      {children}
    </section>
  );
}
