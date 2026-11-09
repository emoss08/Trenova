import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type {
  FactoringCompanyOverview,
  FactoringPacePoint,
} from "@/lib/graphql/factoring-company-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

const WIDTH = 360;
const HEIGHT = 132;
const LEFT = 28;
const RIGHT = 352;
const TOP = 8;
const BOTTOM = 108;
const MIN_MAX_DAYS = 3;

function PaceDot({
  point,
  cx,
  cy,
  step,
}: {
  point: FactoringPacePoint;
  cx: number;
  cy: number;
  step: number;
}) {
  const hover = usePointerHoverTarget(point, point.submissionId);
  return (
    <g>
      <circle
        cx={cx}
        cy={cy}
        r={4}
        strokeWidth={2}
        className={cn("stroke-card", point.late ? "fill-warning" : "fill-foreground")}
      />
      <rect
        x={cx - step / 2}
        y={TOP}
        width={step}
        height={BOTTOM - TOP}
        fill="transparent"
        tabIndex={0}
        aria-label={point.number}
        className="ui-focus-ring outline-none"
        {...hover}
      />
    </g>
  );
}

/**
 * How long the factor took to fund its last schedules, in business days from sent to funded,
 * against the agreement's funding days. Late schedules wear the warning tone and say so in the
 * legend and the hover, never by colour alone.
 */
export function FactoringPaceChart({ overview }: { overview: FactoringCompanyOverview }) {
  const t = useT();
  const points = overview.pace;
  const agreed = overview.agreement?.fundingDays ?? null;

  if (points.length === 0) {
    return (
      <SectionPanel title={t("Funding pace")}>
        <SectionPanelQuiet>
          {t("No schedule has been funded yet. Each funded schedule shows here.")}
        </SectionPanelQuiet>
      </SectionPanel>
    );
  }

  const maxDays = Math.max(MIN_MAX_DAYS, agreed ?? 0, ...points.map((point) => point.businessDays));
  const y = (days: number) => BOTTOM - (days / maxDays) * (BOTTOM - TOP);
  const step = points.length > 1 ? (RIGHT - LEFT - 12) / (points.length - 1) : 0;
  const x = (index: number) => (points.length > 1 ? LEFT + 6 + index * step : (LEFT + RIGHT) / 2);
  const ticks = Array.from({ length: maxDays + 1 }, (_, days) => days);
  const label = (index: number) => formatUnixDate(points[index].sentAt);

  return (
    <SectionPanel
      title={t("Funding pace")}
      hint={t("Last {0, plural, one {# schedule} other {# schedules}}", points.length)}
    >
      <div className="flex flex-col gap-2 p-3">
        <PointerHoverCard<FactoringPacePoint>
          itemId={(point) => point.submissionId}
          render={(point) => (
            <>
              <HoverCardHeading>{point.number}</HoverCardHeading>
              <HoverCardRow label={t("Sent")} value={formatUnixDate(point.sentAt)} />
              <HoverCardRow label={t("Funded")} value={formatUnixDate(point.fundedAt)} />
              <HoverCardRow
                label={t("Took")}
                value={t(
                  "{0, plural, one {# business day} other {# business days}}",
                  point.businessDays,
                )}
              />
              {point.late ? (
                <HoverCardRow
                  label={t("Agreed")}
                  value={t("{0, plural, one {# day} other {# days}}, late", agreed ?? 0)}
                />
              ) : null}
            </>
          )}
        >
          <svg
            viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
            className="block h-auto w-full overflow-visible"
            role="img"
            aria-label={t("Business days from sent to funded for the last schedules")}
          >
            {ticks.map((days) => (
              <g key={days}>
                <line
                  x1={LEFT}
                  x2={RIGHT}
                  y1={y(days)}
                  y2={y(days)}
                  className="stroke-border-subtle"
                  strokeWidth={1}
                />
                <text
                  x={LEFT - 8}
                  y={y(days) + 3}
                  textAnchor="end"
                  className="fill-foreground-subtle text-2xs tabular-nums"
                >
                  {t("{0}d", days)}
                </text>
              </g>
            ))}
            {agreed !== null ? (
              <line
                x1={LEFT}
                x2={RIGHT}
                y1={y(agreed)}
                y2={y(agreed)}
                strokeDasharray="3 3"
                strokeWidth={1}
                className="stroke-foreground-muted"
              />
            ) : null}
            {points.map((point, index) => (
              <PaceDot
                key={point.submissionId}
                point={point}
                cx={x(index)}
                cy={y(point.businessDays)}
                step={Math.max(step, 12)}
              />
            ))}
            <text x={x(0)} y={126} className="fill-foreground-subtle text-2xs">
              {label(0)}
            </text>
            {points.length > 1 ? (
              <text
                x={x(points.length - 1)}
                y={126}
                textAnchor="end"
                className="fill-foreground-subtle text-2xs"
              >
                {label(points.length - 1)}
              </text>
            ) : null}
          </svg>
        </PointerHoverCard>
        <div className="flex flex-wrap gap-x-3.5 gap-y-1 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <span aria-hidden className="size-2 rounded-full bg-foreground" />
            {t("On time")}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span aria-hidden className="size-2 rounded-full bg-warning" />
            {t("Late")}
          </span>
          {agreed !== null ? (
            <span className="inline-flex items-center gap-1.5">
              <span aria-hidden className="w-3 border-t border-dashed border-foreground-muted" />
              {t("Agreed: {0, plural, one {# business day} other {# business days}}", agreed)}
            </span>
          ) : null}
        </div>
        {overview.paceMedianDays !== null && overview.paceMedianDays !== undefined ? (
          <p className="text-sm text-muted-foreground">
            {overview.paceLate === 0
              ? t(
                  "Median {0, plural, one {# business day} other {# business days}}, every schedule on time.",
                  overview.paceMedianDays,
                )
              : t(
                  "Median {0, plural, one {# business day} other {# business days}}. {1} of the last {2} funded late.",
                  overview.paceMedianDays,
                  overview.paceLate,
                  points.length,
                )}
          </p>
        ) : null}
      </div>
    </SectionPanel>
  );
}
