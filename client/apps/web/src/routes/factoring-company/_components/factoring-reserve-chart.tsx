import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import {
  FACTORING_RESERVE_HISTORY_DAYS,
  type FactoringReserveDay,
} from "@/lib/graphql/factoring-company-page";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";

const WIDTH = 1000;
const HEIGHT = 170;
const LEFT = 56;
const RIGHT = 992;
const TOP = 12;
const BOTTOM = 142;

function niceCeiling(value: number): number {
  if (value <= 0) return 100_00;
  const magnitude = 10 ** Math.floor(Math.log10(value));
  return Math.ceil(value / magnitude) * magnitude;
}

function DayTarget({ day, x, width }: { day: FactoringReserveDay; x: number; width: number }) {
  const hover = usePointerHoverTarget(day, String(day.day));
  return (
    <rect
      x={x - width / 2}
      y={TOP}
      width={width}
      height={BOTTOM - TOP}
      fill="transparent"
      className="hover:fill-surface-hover"
      {...hover}
    />
  );
}

/**
 * The reserve the factor held at the end of each of the last days, walked back from today's
 * balance through each day's movements: how much of your money it is holding, and when that moved.
 */
export function FactoringReserveChart({
  factoringCompanyId,
  currencyCode,
}: {
  factoringCompanyId: string;
  currencyCode: string;
}) {
  const t = useT();
  const { data, isLoading, error, refetch } = useQuery(
    factoringQueries.reserveHistory(factoringCompanyId),
  );
  const title = t("Reserve held, last {0} days", FACTORING_RESERVE_HISTORY_DAYS);

  if (isLoading) {
    return (
      <SectionPanel title={title}>
        <div aria-busy aria-label={t("Loading the reserve history")} className="p-3">
          <span aria-hidden className="ui-shimmer block h-40 rounded-md" />
        </div>
      </SectionPanel>
    );
  }
  if (error || !data) {
    return (
      <SectionPanel title={title}>
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The reserve history could not be loaded.")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      </SectionPanel>
    );
  }
  if (data.every((day) => day.reserveMinor === 0)) {
    return (
      <SectionPanel title={title}>
        <SectionPanelQuiet>{t("The factor has held no reserve in this window.")}</SectionPanelQuiet>
      </SectionPanel>
    );
  }

  const top = niceCeiling(Math.max(...data.map((day) => day.reserveMinor)));
  const bottom = Math.min(0, ...data.map((day) => day.reserveMinor));
  const span = top - bottom || 1;
  const x = (index: number) => LEFT + (index / Math.max(data.length - 1, 1)) * (RIGHT - LEFT);
  const y = (minor: number) => BOTTOM - ((minor - bottom) / span) * (BOTTOM - TOP);
  const line = data
    .map((day, index) => `${index ? "L" : "M"}${x(index)} ${y(day.reserveMinor)}`)
    .join(" ");
  const area = `M${x(0)} ${y(bottom)} ${data
    .map((day, index) => `L${x(index)} ${y(day.reserveMinor)}`)
    .join(" ")} L${x(data.length - 1)} ${y(bottom)} Z`;
  const ticks = [bottom, bottom + span / 2, top];
  const step = (RIGHT - LEFT) / Math.max(data.length - 1, 1);
  const last = data.at(-1);
  const labelled = [0, Math.floor((data.length - 1) / 2), data.length - 1];

  return (
    <SectionPanel title={title} hint={t("Hover a day")}>
      <div className="p-3">
        <PointerHoverCard<FactoringReserveDay>
          itemId={(day) => String(day.day)}
          render={(day) => (
            <>
              <HoverCardHeading>{formatUnixDate(day.day)}</HoverCardHeading>
              <HoverCardRow
                label={t("Reserve held")}
                value={formatMinor(day.reserveMinor, currencyCode)}
                swatchClass="bg-chart-1"
              />
            </>
          )}
        >
          <svg
            viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
            className="block h-auto w-full overflow-visible"
            role="img"
            aria-label={title}
          >
            {ticks.map((tick) => (
              <g key={tick}>
                <line
                  x1={LEFT}
                  x2={RIGHT}
                  y1={y(tick)}
                  y2={y(tick)}
                  strokeWidth={1}
                  className="stroke-border-subtle"
                />
                <text
                  x={LEFT - 8}
                  y={y(tick) + 3}
                  textAnchor="end"
                  className="fill-foreground-subtle text-2xs tabular-nums"
                >
                  {formatMinor(tick, currencyCode)}
                </text>
              </g>
            ))}
            <path d={area} className="fill-chart-1/15" />
            <path
              d={line}
              fill="none"
              strokeWidth={2}
              strokeLinejoin="round"
              className="stroke-chart-1"
            />
            {data.map((day, index) => (
              <DayTarget key={day.day} day={day} x={x(index)} width={step} />
            ))}
            {last ? (
              <circle
                cx={x(data.length - 1)}
                cy={y(last.reserveMinor)}
                r={4}
                strokeWidth={2}
                className="fill-chart-1 stroke-card"
              />
            ) : null}
            {labelled.map((index) => (
              <text
                key={index}
                x={x(index)}
                y={HEIGHT - 6}
                textAnchor={index === 0 ? "start" : index === data.length - 1 ? "end" : "middle"}
                className="fill-foreground-subtle text-2xs"
              >
                {formatUnixDate(data[index].day)}
              </text>
            ))}
          </svg>
        </PointerHoverCard>
      </div>
    </SectionPanel>
  );
}
