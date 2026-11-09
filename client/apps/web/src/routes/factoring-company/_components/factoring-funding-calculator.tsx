import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { formatBasisPoints } from "@/lib/factoring";
import type { FactoringFundingPreview } from "@/lib/graphql/factoring-company-page";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { InfoCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { Slider } from "@trenova/shared/components/ui/slider";
import { useThrottledValue } from "@trenova/shared/hooks/use-throttled-value";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { currencySymbol, formatMinor, parseAmountToMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type ReactNode } from "react";
import { NumericFormat } from "react-number-format";

const MAX_DAYS = 90;
const DAY_MARKS = [0, 30, 60, 90] as const;
const DEFAULT_FACE = "10000.00";
const DEFAULT_DAYS = 30;
const REFRESH_MS = 120;
const DAYS_PER_YEAR = 365;
const AXIS_MIN_GAP_DAYS = 9;

const CHART = { width: 268, height: 150, left: 30, right: 260, top: 14, bottom: 122 };

export type FundingTerms = {
  advanceRateBp: number;
  reserveRateBp: number;
  minimumFeeMinor: number;
  feeTiers: readonly {
    daysFrom: number;
    daysTo?: number | null;
    rateBp: number;
    flatMinor: number;
  }[];
};

type FeeTier = FundingTerms["feeTiers"][number];

/** The tier whose days hold `days`; past the last boundary, the last tier. */
function tierAt(tiers: readonly FeeTier[], days: number): FeeTier | undefined {
  return (
    tiers.find((tier) => days >= tier.daysFrom && (tier.daysTo == null || days < tier.daysTo)) ??
    tiers.at(-1)
  );
}

/**
 * The days the chart labels: both ends and every tier boundary between them, dropping a boundary
 * that would sit on top of the one before it (the right end always stays).
 */
function axisDays(tiers: readonly FeeTier[]): number[] {
  const boundaries = [
    ...new Set([0, MAX_DAYS, ...tiers.flatMap((tier) => [tier.daysFrom, tier.daysTo ?? MAX_DAYS])]),
  ]
    .filter((d) => d >= 0 && d <= MAX_DAYS)
    .sort((a, b) => a - b);
  const kept: number[] = [];
  for (const d of boundaries) {
    const last = kept.at(-1);
    if (last === undefined || d - last >= AXIS_MIN_GAP_DAYS) kept.push(d);
    else if (d === MAX_DAYS) kept[kept.length - 1] = d;
  }
  return kept;
}

function Swatch({ className }: { className: string }) {
  return <span aria-hidden className={cn("inline-block size-2 shrink-0 rounded-xs", className)} />;
}

function SwatchLabel({ swatch, children }: { swatch: string; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <Swatch className={swatch} />
      {children}
    </span>
  );
}

/** The date an advance requested today lands, counting only weekdays. */
function fundedBy(businessDays: number): number {
  const day = new Date();
  let left = businessDays;
  while (left > 0) {
    day.setDate(day.getDate() + 1);
    const weekday = day.getDay();
    if (weekday !== 0 && weekday !== 6) left -= 1;
  }
  return Math.floor(day.getTime() / 1000);
}

/** The fee as a yearly rate on the money advanced, if the customer pays on the due date. */
function annualizedFeeRate(funding: FactoringFundingPreview): number | null {
  if (funding.advanceMinor <= 0 || funding.days <= 0) return null;
  return (funding.feeMinor / funding.advanceMinor) * (DAYS_PER_YEAR / funding.days);
}

/**
 * The fee tiers as steps by days left, with a marker on the step the slider is over. Which drawn
 * step sits under the thumb is read from the steps themselves, so the marker moves with the thumb;
 * the money still comes from the server.
 */
function FeeSteps({ terms, days }: { terms: FundingTerms; days: number }) {
  const t = useT();
  const tiers = [...terms.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);
  const maxPercent = Math.ceil(Math.max(100, ...tiers.map((tier) => tier.rateBp)) / 100);
  const x = (d: number) =>
    CHART.left + (Math.min(d, MAX_DAYS) / MAX_DAYS) * (CHART.right - CHART.left);
  const y = (bp: number) => CHART.bottom - (bp / (maxPercent * 100)) * (CHART.bottom - CHART.top);
  const path = tiers
    .map((tier, index) => {
      const level = y(tier.rateBp);
      return `${index === 0 ? "M" : "L"}${x(tier.daysFrom)} ${level} L${x(tier.daysTo ?? MAX_DAYS)} ${level}`;
    })
    .join(" ");
  const markerX = x(days);
  const rateBp = tierAt(tiers, days)?.rateBp ?? null;
  const glide = "transition-transform duration-150 ease-out";

  return (
    <svg
      viewBox={`0 0 ${CHART.width} ${CHART.height}`}
      className="block h-auto w-full overflow-visible"
      role="img"
      aria-label={t("Fee rate by days left on an invoice's terms")}
    >
      {Array.from({ length: maxPercent + 1 }, (_, percent) => (
        <g key={percent}>
          <line
            x1={CHART.left}
            x2={CHART.right}
            y1={y(percent * 100)}
            y2={y(percent * 100)}
            strokeWidth={1}
            className="stroke-border-subtle"
          />
          <text
            x={CHART.left - 6}
            y={y(percent * 100) + 3}
            textAnchor="end"
            className="fill-foreground-subtle text-2xs tabular-nums"
          >
            {t("{0}%", percent)}
          </text>
        </g>
      ))}
      {axisDays(tiers).map((d) => (
        <text
          key={d}
          x={x(d)}
          y={CHART.bottom + 16}
          textAnchor={d === 0 ? "start" : d === MAX_DAYS ? "end" : "middle"}
          className="fill-foreground-subtle text-2xs tabular-nums"
        >
          {t("{0}d", d)}
        </text>
      ))}
      <path
        d={path}
        fill="none"
        strokeWidth={2}
        strokeLinejoin="round"
        className="stroke-foreground"
      />
      <g className={glide} style={{ transform: `translateX(${markerX}px)` }}>
        <line
          x1={0}
          x2={0}
          y1={CHART.top}
          y2={CHART.bottom}
          strokeDasharray="2 2"
          strokeWidth={1}
          className="stroke-warning-foreground"
        />
      </g>
      {rateBp !== null ? (
        <g className={glide} style={{ transform: `translate(${markerX}px, ${y(rateBp)}px)` }}>
          <circle cx={0} cy={0} r={4.5} strokeWidth={2} className="fill-warning stroke-card" />
          <text
            x={Math.min(Math.max(markerX, CHART.left + 18), CHART.right - 18) - markerX}
            y={-9}
            textAnchor="middle"
            className="fill-foreground text-2xs font-medium tabular-nums"
          >
            {formatBasisPoints(rateBp)}
          </text>
        </g>
      ) : null}
    </svg>
  );
}

/**
 * What one invoice pays under the agreement: type an amount and the days left on its terms, and
 * the server works out the advance, fee and release exactly as funding will, with what the fee
 * costs a year if the customer pays on the due date. The browser does none of the money math.
 */
export function FactoringFundingCalculator({
  terms,
  fundingDays,
  children,
}: {
  terms: FundingTerms;
  fundingDays: number;
  /** The agreement's full terms, shown under the figures. */
  children?: ReactNode;
}) {
  const t = useT();
  const rt = useRichT();
  const amountId = useId();
  const daysId = useId();
  const [amount, setAmount] = useState(DEFAULT_FACE);
  const [days, setDays] = useState(DEFAULT_DAYS);
  const faceMinor = parseAmountToMinor(amount.replace(/\.$/, "")) ?? 0;
  const tiers = [...terms.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);
  const settledFace = useThrottledValue(faceMinor, REFRESH_MS);
  const settledDays = useThrottledValue(days, REFRESH_MS);
  const preview = useQuery({
    ...factoringQueries.fundingPreview({
      faceMinor: settledFace,
      days: settledDays,
      advanceRateBp: terms.advanceRateBp,
      reserveRateBp: terms.reserveRateBp,
      minimumFeeMinor: terms.minimumFeeMinor,
      feeTiers: terms.feeTiers.map((tier) => ({
        daysFrom: tier.daysFrom,
        daysTo: tier.daysTo ?? null,
        rateBp: tier.rateBp,
        flatMinor: tier.flatMinor,
      })),
    }),
    placeholderData: keepPreviousData,
  });
  const funding = preview.data;
  const waiting = preview.isPlaceholderData || preview.isFetching;
  const share = (minor: number) =>
    funding && funding.faceMinor > 0 ? minor / funding.faceMinor : 0;
  const annual = funding ? annualizedFeeRate(funding) : null;
  const minimumCharged =
    funding !== undefined &&
    terms.minimumFeeMinor > 0 &&
    funding.feeMinor === terms.minimumFeeMinor &&
    funding.faceMinor > terms.minimumFeeMinor;
  const feeTierLine = (preview: FactoringFundingPreview) => {
    const rate = formatBasisPoints(preview.feeRateBp);
    const tier = tierAt(tiers, preview.days);
    if (!tier) return t("At the {0} tier", rate);
    return tier.daysTo == null
      ? t("{0} tier, {1}+ days", rate, tier.daysFrom)
      : t("{0} tier, days {1}–{2}", rate, tier.daysFrom, tier.daysTo);
  };
  const figure = (minor: number | undefined) =>
    minor === undefined ? (
      <span className="ui-shimmer inline-block h-5 w-20 rounded-sm" />
    ) : (
      formatMinor(minor)
    );

  return (
    <div className="grid md:grid-cols-[18.75rem_minmax(0,1fr)]">
      <div className="flex flex-col gap-4 border-b p-4 md:border-r md:border-b-0">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={amountId}>{t("Invoice amount")}</Label>
          <div className="ui-field ui-container-focus-ring flex h-9 items-center gap-1 px-2.5">
            <span aria-hidden className="text-foreground-subtle text-base">
              {currencySymbol()}
            </span>
            <NumericFormat
              id={amountId}
              value={amount}
              valueIsNumericString
              onValueChange={(values) => setAmount(values.value)}
              thousandSeparator
              decimalScale={2}
              fixedDecimalScale
              allowNegative={false}
              inputMode="decimal"
              placeholder="0.00"
              className="placeholder:text-foreground-subtle min-w-0 flex-1 bg-transparent text-lg font-medium tabular-nums outline-none"
            />
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label id={daysId}>{t("Days left on its terms when submitted")}</Label>
          <div className="flex items-start gap-2.5">
            <Slider
              aria-labelledby={daysId}
              min={0}
              max={MAX_DAYS}
              step={1}
              value={days}
              onValueChange={(value) => setDays(value)}
              marks={DAY_MARKS.map((d) => ({ value: d, label: d.toLocaleString() }))}
              className="-mx-2 min-w-0 flex-1"
            />
            <output className="-mt-0.5 w-16 shrink-0 text-right text-lg font-semibold tabular-nums">
              {t("{0, plural, one {# day} other {# days}}", days)}
            </output>
          </div>
          <p className="text-muted-foreground text-xs">
            {t(
              "The days left pick the fee tier. Net 30 submitted the day it's invoiced has 30 left.",
            )}
          </p>
        </div>
        <FeeSteps terms={terms} days={days} />
      </div>
      <div aria-busy={waiting} className="flex min-w-0 flex-col gap-4 p-4">
        {preview.error ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("What this invoice pays could not be worked out.")}</span>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void preview.refetch()}
              >
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : null}
        <div
          aria-hidden
          className="flex h-7 gap-0.5 overflow-hidden rounded-sm text-xs font-medium"
        >
          <span
            className="bg-foreground text-background flex items-center truncate px-2 transition-[flex-grow] duration-200"
            style={{ flexGrow: funding ? share(funding.advanceMinor) : 1 }}
          >
            {funding ? t("Advance {0}", formatMinor(funding.advanceMinor)) : null}
          </span>
          <span
            className="bg-warning min-w-1.5 transition-[flex-grow] duration-200"
            style={{ flexGrow: funding ? share(funding.feeMinor) : 0 }}
          />
          <span
            className="bg-muted text-foreground flex items-center truncate px-2 transition-[flex-grow] duration-200"
            style={{ flexGrow: funding ? share(funding.restOnReleaseMinor) : 0 }}
          >
            {funding ? t("Release {0}", formatMinor(funding.restOnReleaseMinor)) : null}
          </span>
        </div>
        <KpiStrip minItemWidth="9rem" aria-label={t("What the invoice pays")}>
          <KpiStripItem
            label={<SwatchLabel swatch="bg-foreground">{t("Advance")}</SwatchLabel>}
            value={figure(funding?.advanceMinor)}
            sub={t(
              "Paid by {0} ({1, plural, one {# business day} other {# business days}})",
              formatUnixDate(fundedBy(fundingDays)),
              fundingDays,
            )}
          />
          <KpiStripItem
            label={<SwatchLabel swatch="bg-warning">{t("Fee")}</SwatchLabel>}
            value={figure(funding?.feeMinor)}
            sub={
              minimumCharged
                ? t("The minimum fee; the tier's rate comes to less")
                : funding
                  ? feeTierLine(funding)
                  : t("No fee tier")
            }
          />
          <KpiStripItem
            label={
              <SwatchLabel swatch="bg-muted ring-border-strong ring-1 ring-inset">
                {t("Paid on release")}
              </SwatchLabel>
            }
            value={figure(funding?.restOnReleaseMinor)}
            sub={t("When the customer pays the factor")}
          />
        </KpiStrip>
        <p className="text-muted-foreground bg-sunken flex items-start gap-2 rounded-md border px-3 py-2.5 text-sm">
          <InfoCircleIcon className="mt-0.5 size-3.5 shrink-0" />
          <span>
            {faceMinor === 0
              ? t("Enter an amount to see what it pays.")
              : !funding
                ? t("Working out what it pays...")
                : annual === null
                  ? t("With no days left, the fee is {0} flat.", formatMinor(funding.feeMinor))
                  : rt(
                      "If the customer pays on the due date, the fee costs about <b>{0} a year</b> on the money advanced. Paid sooner, the factor keeps the same fee.",
                      {
                        b: (chunks) => (
                          <strong className="text-foreground font-medium tabular-nums">
                            {chunks}
                          </strong>
                        ),
                      },
                      t(
                        "{0}%",
                        (annual * 100).toLocaleString(undefined, { maximumFractionDigits: 1 }),
                      ),
                    )}
          </span>
        </p>
        {children}
      </div>
    </div>
  );
}
