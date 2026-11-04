import { recordPath } from "@/config/record-links";
import type {
  DetentionFacility,
  DetentionIntelligence,
} from "@/lib/graphql/detention-intelligence";
import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import { AlertTriangleIcon, Download01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigate } from "react-router";
import {
  ARRIVAL_BANDS,
  ARRIVAL_BAND_LABELS,
  CAUSES,
  CAUSE_LABELS,
  CAUSE_LEVER,
  DETENTION_DESK_PATH,
  DETENTION_POLICIES_PATH,
  INDUSTRY_DETAINED_SHARE,
  NOTICE_ON_TIME_TARGET,
  WEEKDAYS,
  WEEKDAY_LABELS,
  amount,
  arrivalPeak,
  causeMinutes,
  foldArrivals,
  hours,
  shareOf,
} from "./intelligence-model";
import { toCsv } from "./intelligence-export";
import { money, percent } from "./intelligence-parts";
import { WeeklyHoursChart } from "./trend-section";
import { useFacilityIntelligence } from "./use-detention-intelligence";
import { ArrivalGrid } from "./when-section";

/**
 * Inset from the window on three sides and 540px wide, as the design draws it. The sheet's own
 * width is set on its side variant, so the override has to be too, or the narrower default wins.
 */
const SHEET_PLACEMENT =
  "flex flex-col gap-0 overflow-hidden p-0 data-[side=right]:top-2 data-[side=right]:right-2 data-[side=right]:bottom-2 data-[side=right]:w-[calc(100%-1rem)] data-[side=right]:sm:max-w-135";

/** The detention desk narrowed to one facility's occurrences, through its own search. */
function deskSearchPath(locationName: string): string {
  return `${DETENTION_DESK_PATH}?${new URLSearchParams({ q: locationName }).toString()}`;
}

function ScoreFigure({
  label,
  value,
  sub,
  tone,
  subTone,
}: {
  label: string;
  value: string;
  sub: string;
  tone?: "danger";
  /** Colours the line under the figure: worse is danger, better is good. */
  subTone?: "danger" | "good";
}) {
  return (
    <div className="min-w-0 px-3 py-2.5">
      <p className="text-muted-foreground text-xs">{label}</p>
      <p
        className={cn(
          "mt-0.5 font-mono text-lg font-semibold tabular-nums",
          tone === "danger" && "text-danger-foreground",
        )}
      >
        {value}
      </p>
      <p
        className={cn(
          "text-2xs leading-tight",
          subTone === "danger"
            ? "text-danger-foreground"
            : subTone === "good"
              ? "text-success-foreground"
              : "text-muted-foreground",
        )}
      >
        {sub}
      </p>
    </div>
  );
}

function MoneyLine({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3 border-b py-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono tabular-nums">{children}</span>
    </div>
  );
}

/**
 * One facility's detention, laid out for the conversation with the shipper: how long trucks
 * wait, whose wait it is, when it happens and what it costs. It exports, because a scorecard is
 * what a carrier brings to a quarterly review.
 */
export function FacilityScorecardSheet({
  page,
  facility,
  days,
  onClose,
}: {
  /** The page's read, whose window the scorecard narrows to one facility. */
  page: DetentionIntelligence;
  facility: DetentionFacility | null;
  days: number;
  onClose: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const navigate = useNavigate();
  const scoped = useFacilityIntelligence(page, facility?.locationId ?? null);
  const data = facility ? scoped.data : undefined;
  const grid = data ? foldArrivals(data.arrivals) : null;
  const peak = grid ? arrivalPeak(grid) : null;
  const place = facility
    ? [facility.locationCity, facility.locationState].filter(Boolean).join(", ")
    : "";
  const change =
    facility?.previousPastFreeMinutes === null || facility?.previousPastFreeMinutes === undefined
      ? null
      : hours(facility.pastFreeMinutes) - hours(facility.previousPastFreeMinutes);

  const detained = facility ? shareOf(facility.detainedCount, facility.stopCount) : 0;
  const lead = facility
    ? [...CAUSES].sort((a, b) => causeMinutes(facility, b.key) - causeMinutes(facility, a.key))[0]
    : undefined;
  const overFree = facility ? facility.medianDwellMinutes > facility.medianFreeMinutes : false;
  const net =
    facility?.netMargin === null || facility?.netMargin === undefined
      ? null
      : amount(facility.netMargin);
  const notice =
    facility && facility.noticeDueCount > 0
      ? shareOf(facility.noticeOnTimeCount, facility.noticeDueCount)
      : null;

  const exportScorecard = () => {
    if (!facility) return;
    const rows: [string, string | number][] = [
      [t("Facility"), facility.locationName],
      [t("Window"), t("Last {0} days", days)],
      [t("Stops"), facility.stopCount],
      [t("Stops detained"), percent(detained)],
      [t("Hours past free time"), hours(facility.pastFreeMinutes)],
      ...CAUSES.map((cause): [string, number] => [
        CAUSE_LABELS[cause.key],
        hours(causeMinutes(facility, cause.key)),
      ]),
      [t("Median stop"), formatDetentionMinutes(Math.round(facility.medianDwellMinutes))],
      [t("Slowest one in ten"), formatDetentionMinutes(Math.round(facility.p90DwellMinutes))],
      [t("Free time granted"), formatDetentionMinutes(Math.round(facility.medianFreeMinutes))],
      [t("Billed"), money(amount(facility.billedAmount))],
      [t("Forgiven"), money(amount(facility.waivedAmount))],
    ];
    const csv = toCsv([[t("Measure"), t("Value")], ...rows]);
    downloadCsv(csv, exportFilename(`detention-scorecard-${facility.locationName}`));
  };

  return (
    <Sheet open={facility !== null} onOpenChange={(open) => (open ? undefined : onClose())}>
      <SheetContent side="right" className={SHEET_PLACEMENT}>
        {facility ? (
          <>
            <SheetHeader className="border-b p-4">
              <p className="text-muted-foreground text-xs">
                {t("Facility scorecard · last {0} days", days)}
              </p>
              <SheetTitle>{facility.locationName}</SheetTitle>
              <SheetDescription>
                {place
                  ? t(
                      "{0} · {1, plural, one {# stop} other {# stops}} · {2, plural, one {# detained} other {# detained}}",
                      place,
                      facility.stopCount,
                      facility.detainedCount,
                    )
                  : t(
                      "{0, plural, one {# stop} other {# stops}} · {1, plural, one {# detained} other {# detained}}",
                      facility.stopCount,
                      facility.detainedCount,
                    )}
              </SheetDescription>
              <div className="mt-2 grid grid-cols-3 divide-x rounded-md border">
                <ScoreFigure
                  label={t("Past free time")}
                  value={t("{0} h", hours(facility.pastFreeMinutes))}
                  sub={
                    change === null
                      ? t("Not in the window before")
                      : t(
                          "{0} h vs prior",
                          change > 0
                            ? `+${change.toLocaleString()}`
                            : change < 0
                              ? `−${Math.abs(change).toLocaleString()}`
                              : "0",
                        )
                  }
                  subTone={
                    change === null || change === 0 ? undefined : change > 0 ? "danger" : "good"
                  }
                />
                <ScoreFigure
                  label={t("Stops detained")}
                  value={percent(detained)}
                  sub={t("Industry {0}", percent(INDUSTRY_DETAINED_SHARE))}
                />
                <ScoreFigure
                  label={t("Median stop")}
                  value={formatDetentionMinutes(Math.round(facility.medianDwellMinutes))}
                  sub={t(
                    "{0} free time",
                    formatDetentionMinutes(Math.round(facility.medianFreeMinutes)),
                  )}
                  tone={overFree ? "danger" : undefined}
                />
              </div>
            </SheetHeader>

            <div className="flex-1 overflow-y-auto p-4">
              {overFree ? (
                <Alert size="sm" variant="destructive" className="mb-4">
                  <AlertTriangleIcon />
                  <AlertDescription>
                    {t(
                      "The typical stop here takes {0} longer than the free time granted, so detention is built into the schedule.",
                      formatDetentionMinutes(
                        Math.round(facility.medianDwellMinutes - facility.medianFreeMinutes),
                      ),
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}

              <h3 className="text-sm font-semibold">{t("Who caused the wait")}</h3>
              {lead ? (
                <p className="text-muted-foreground mt-1 text-sm">
                  {t(
                    "{0} of the hours: {1}. {2}",
                    percent(shareOf(causeMinutes(facility, lead.key), facility.pastFreeMinutes)),
                    CAUSE_LABELS[lead.key],
                    CAUSE_LEVER[lead.key],
                  )}
                </p>
              ) : null}
              <div className="mt-2 flex h-3 gap-0.5 overflow-hidden rounded-sm">
                {CAUSES.map((cause) =>
                  causeMinutes(facility, cause.key) > 0 ? (
                    <span
                      key={cause.key}
                      className={cn("block h-full", cause.bgClass)}
                      style={{ flexGrow: causeMinutes(facility, cause.key) }}
                    />
                  ) : null,
                )}
              </div>
              <div className="text-muted-foreground mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
                {CAUSES.map((cause) => (
                  <span key={cause.key} className="inline-flex items-center gap-1.5">
                    <span className={cn("size-2 rounded-sm", cause.bgClass)} />
                    {t(
                      "{0} · {1} h",
                      CAUSE_LABELS[cause.key],
                      hours(causeMinutes(facility, cause.key)),
                    )}
                  </span>
                ))}
              </div>

              <h3 className="mt-6 text-sm font-semibold">{t("When it happens")}</h3>
              {peak ? (
                <p className="text-muted-foreground mt-1 text-sm">
                  {rt(
                    "The worst arrival window is <b>{0} {1}</b>, detained {2} of the time.",
                    { b: (chunk) => <span className="text-foreground font-medium">{chunk}</span> },
                    WEEKDAY_LABELS[WEEKDAYS[peak.weekday]],
                    ARRIVAL_BAND_LABELS[ARRIVAL_BANDS[peak.band].key],
                    percent(peak.share),
                  )}
                </p>
              ) : null}
              <div className="mt-2" aria-busy={scoped.isLoading}>
                {grid ? (
                  <ArrivalGrid grid={grid} compact />
                ) : (
                  <div className="ui-shimmer h-40 rounded-md" />
                )}
              </div>

              <h3 className="mt-6 text-sm font-semibold">{t("Weekly hours past free time")}</h3>
              <div className="mt-2">
                {data ? (
                  <WeeklyHoursChart
                    weeks={data.weeks}
                    baseline={null}
                    changes={data.policyChanges}
                    window={data}
                    height={120}
                  />
                ) : (
                  <div className="ui-shimmer h-28 rounded-md" />
                )}
              </div>

              <h3 className="mt-6 text-sm font-semibold">{t("Money")}</h3>
              <div className="mt-1 grid gap-x-6 sm:grid-cols-2">
                <MoneyLine label={t("Billed")}>{money(amount(facility.billedAmount))}</MoneyLine>
                <MoneyLine label={t("Forgiven")}>
                  <span className="text-warning-foreground">
                    {money(amount(facility.waivedAmount))}
                  </span>
                </MoneyLine>
                <MoneyLine label={t("Driver pay")}>
                  {facility.driverPayAmount === null ? (
                    <WithheldValue group="Margins" />
                  ) : (
                    money(amount(facility.driverPayAmount))
                  )}
                </MoneyLine>
                <MoneyLine label={t("Net")}>
                  {net === null ? (
                    <WithheldValue group="Margins" />
                  ) : (
                    <span
                      className={net < 0 ? "text-danger-foreground" : "text-success-foreground"}
                    >
                      {net > 0 ? t("+{0}", money(net)) : money(net)}
                    </span>
                  )}
                </MoneyLine>
                <MoneyLine label={t("Notice on time")}>
                  {notice === null ? (
                    "—"
                  ) : (
                    <span
                      className={
                        notice < NOTICE_ON_TIME_TARGET ? "text-danger-foreground" : undefined
                      }
                    >
                      {percent(notice)}
                    </span>
                  )}
                </MoneyLine>
                <MoneyLine label={t("Free time granted")}>
                  {formatDetentionMinutes(Math.round(facility.medianFreeMinutes))}
                </MoneyLine>
              </div>
            </div>

            {/* As the design lays it out: the scorecard's own action first, then where to act on
                it, wrapping on a phone. */}
            <SheetFooter className="flex-wrap justify-start bg-transparent px-4.5">
              <Button type="button" size="sm" onClick={exportScorecard}>
                <Download01Icon className="size-3.5" />
                {t("Export scorecard")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void navigate(DETENTION_POLICIES_PATH)}
              >
                {t("Backtest a policy")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void navigate(deskSearchPath(facility.locationName))}
              >
                {t("See occurrences")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void navigate(recordPath("location", facility.locationId))}
              >
                {t("Open location")}
              </Button>
            </SheetFooter>
          </>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
