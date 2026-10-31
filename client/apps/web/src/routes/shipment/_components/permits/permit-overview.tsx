import { InfoPopover } from "@/components/info-popover";
import { Car01Icon, CheckIcon, MoonStarIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import {
  dimensionMeter,
  formatMeasurement,
  type DimensionMeterTone,
  type DimensionRow,
  type EscortSummaryEntry,
  type RestrictionSummaryEntry,
} from "@trenova/shared/lib/permit";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { PermitAssessment } from "@trenova/shared/types/permit";
import type { ReactNode } from "react";

/** A section of the tab: a quiet heading over its content. */
export function PermitSection({
  title,
  meta,
  children,
}: {
  title: string;
  meta?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section aria-label={title} className="flex flex-col gap-2.5">
      <div className="flex items-baseline justify-between gap-2">
        <h3 className="text-foreground-muted text-xs font-medium">{title}</h3>
        {meta}
      </div>
      {children}
    </section>
  );
}

/**
 * Where the shipment's permits stand, said first: how many are still to file (or that they are
 * all in place, or that the load is legal on this route), how far through the checklist it is,
 * and when the load can leave at the earliest, with the lead time and fees behind that date.
 */
export function PermitSummary({
  assessment,
  open,
  resolved,
  pickupTooSoon,
}: {
  assessment: PermitAssessment;
  open: number;
  resolved: number;
  pickupTooSoon: boolean;
}) {
  const t = useT();
  const total = open + resolved;
  const fee = Number(assessment.totalEstimatedFee ?? 0);

  const unchecked = total === 0 && !assessment.routeResolved;
  const title =
    open > 0
      ? t("{0, plural, one {# permit to file} other {# permits to file}}", open)
      : total > 0
        ? t("Permits in place")
        : unchecked
          ? t("Route not checked")
          : t("Legal on this route");
  const done = open === 0 && !unchecked;

  const facts: { key: string; node: ReactNode }[] = [];
  if (assessment.maxLeadTimeDays > 0) {
    facts.push({
      key: "pickup",
      node: (
        <span className={cn(pickupTooSoon && "text-warning-foreground font-medium")}>
          {t("Earliest pickup {0}", formatUnixDateTimeShort(assessment.earliestPickup))}
        </span>
      ),
    });
    facts.push({
      key: "lead",
      node: t(
        "{0, plural, one {# day of lead time} other {# days of lead time}}",
        assessment.maxLeadTimeDays,
      ),
    });
  }
  if (fee > 0) {
    facts.push({
      key: "fee",
      node: (
        <span className="flex items-center gap-1">
          {assessment.feeIsBaseOnly
            ? t("From {0} in fees", formatCurrency(fee))
            : t("About {0} in fees", formatCurrency(fee))}
          {assessment.feeIsBaseOnly && (
            <InfoPopover title={t("Base fees only")}>
              {t(
                "Per-mile charges are left out because the miles in each state are not known for this route, so the real cost will be higher where a state charges by distance.",
              )}
            </InfoPopover>
          )}
        </span>
      ),
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <h2
          key={title}
          className={cn(
            "animate-rise flex items-center gap-1.5 text-base font-semibold",
            done && "text-success",
            unchecked && "text-foreground-muted",
          )}
        >
          {done && <CheckIcon aria-hidden className="animate-confirm size-4" strokeWidth={2.5} />}
          {title}
        </h2>
        {total > 0 && (
          <span className="text-foreground-muted shrink-0 text-xs tabular-nums">
            {t("{0} of {1} done", resolved, total)}
          </span>
        )}
      </div>
      {total > 0 && (
        <div
          role="progressbar"
          aria-label={t("Permits done")}
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={resolved}
          className="bg-muted h-1.5 w-full overflow-hidden rounded-full"
        >
          <div
            className={cn(
              "h-full rounded-full transition-[width] duration-500 ease-settle",
              done ? "bg-success" : "bg-info",
            )}
            style={{ width: `${(resolved / total) * 100}%` }}
          />
        </div>
      )}
      {facts.length > 0 && (
        <p className="text-foreground-muted flex flex-wrap items-center gap-x-1.5 text-xs">
          {facts.map((fact, index) => (
            <span key={fact.key} className="flex items-center gap-x-1.5">
              {index > 0 && <span aria-hidden>·</span>}
              {fact.node}
            </span>
          ))}
        </p>
      )}
      {pickupTooSoon && (
        <p className="text-warning-foreground text-xs">
          {t("The booked pickup is sooner than the permits can be issued.")}
        </p>
      )}
    </div>
  );
}

const METER_FILL: Record<DimensionMeterTone, string> = {
  ok: "bg-success",
  near: "bg-warning",
  over: "bg-danger",
};

function SizeMeter({ row }: { row: DimensionRow }) {
  const t = useT();
  const meter = dimensionMeter(row);

  return (
    <li
      className="grid grid-cols-[6.5rem_minmax(0,1fr)_auto] items-center gap-x-3"
      data-dimension={row.key}
      data-exceeded={row.exceeded || undefined}
    >
      <span className="flex flex-col">
        <span className="text-foreground-muted text-2xs">{t(row.label)}</span>
        <span className={cn("text-xs font-medium tabular-nums", row.exceeded && "text-danger")}>
          {formatMeasurement(row.actual, row.unit)}
        </span>
      </span>
      <span className="bg-muted h-1.5 overflow-hidden rounded-full">
        {meter && (
          <span
            className={cn(
              "animate-fill block h-full origin-left rounded-full",
              METER_FILL[meter.tone],
            )}
            style={{ width: `${meter.percentOfLimit}%` }}
          />
        )}
      </span>
      <span className="flex flex-col items-end tabular-nums">
        {row.limit === null ? (
          <span className="text-foreground-subtle text-2xs">{t("No limit found")}</span>
        ) : (
          <>
            <span className="text-foreground-muted text-2xs">
              {row.governingStateCode
                ? t("{0} limit {1}", row.governingStateCode, formatMeasurement(row.limit, row.unit))
                : t("Limit {0}", formatMeasurement(row.limit, row.unit))}
            </span>
            {row.headroom !== null && (
              <span
                className={cn(
                  "text-xs font-medium",
                  row.exceeded ? "text-danger" : "text-foreground-muted font-normal",
                )}
              >
                {row.exceeded
                  ? t("{0} over", formatMeasurement(Math.abs(row.headroom), row.unit))
                  : t("{0} left", formatMeasurement(row.headroom, row.unit))}
              </span>
            )}
          </>
        )}
      </span>
    </li>
  );
}

/**
 * The load's size against the tightest limit on the route, one meter per dimension: how full it
 * runs, which state sets the limit, and how much is left or how far over it is.
 */
export function SizeMeters({ rows }: { rows: DimensionRow[] }) {
  const t = useT();
  const over = rows.filter((row) => row.exceeded).length;
  return (
    <PermitSection
      title={t("Size against the tightest limit")}
      meta={
        <span
          className={cn(
            "text-2xs tabular-nums",
            over > 0 ? "text-danger font-medium" : "text-foreground-muted",
          )}
        >
          {over > 0 ? t("{0, plural, one {# over} other {# over}}", over) : t("All within limits")}
        </span>
      }
    >
      <ul className="flex flex-col gap-3">
        {rows.map((row) => (
          <SizeMeter key={row.key} row={row} />
        ))}
      </ul>
    </PermitSection>
  );
}

/**
 * What the trip needs beyond the paper: the escort vehicles, each once for the trip with the
 * states that ask for it, and the movement restrictions the permitting states publish.
 */
export function TripNeeds({
  escorts,
  restrictions,
}: {
  escorts: EscortSummaryEntry[];
  restrictions: RestrictionSummaryEntry[];
}) {
  const t = useT();
  if (escorts.length === 0 && restrictions.length === 0) return null;

  return (
    <PermitSection title={t("What the trip needs")}>
      <ul className="flex flex-col gap-2">
        {escorts.map((escort) => (
          <li key={escort.role} className="flex items-start gap-2.5 text-xs">
            <Car01Icon aria-hidden className="text-foreground-muted mt-0.5 size-3.5 shrink-0" />
            <span>
              <span className="font-medium">{escort.label}</span>
              {escort.stateCodes.length > 0 && (
                <span className="text-foreground-muted">
                  {" · "}
                  {t("required by {0}", escort.stateCodes.join(", "))}
                </span>
              )}
            </span>
          </li>
        ))}
        {restrictions.map((restriction) => (
          <li key={restriction.kind} className="flex items-start gap-2.5 text-xs">
            <MoonStarIcon aria-hidden className="text-foreground-muted mt-0.5 size-3.5 shrink-0" />
            <span>
              <span className="font-medium">{restriction.label}</span>
              <span className="text-foreground-muted">
                {" · "}
                {restriction.stateCodes.join(", ")}
              </span>
            </span>
          </li>
        ))}
      </ul>
      {restrictions.length > 0 && (
        <p className="text-foreground-subtle text-2xs">
          {t(
            "Restrictions are published by the permitting states; Trenova does not yet check them against your appointment times.",
          )}
        </p>
      )}
    </PermitSection>
  );
}
