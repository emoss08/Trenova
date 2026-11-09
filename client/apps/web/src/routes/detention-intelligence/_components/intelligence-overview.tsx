import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type {
  DetentionIntelligence,
  DetentionNextStep,
} from "@/lib/graphql/detention-intelligence";
import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import { ArrowRightIcon, Download01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigate } from "react-router";
import { intelligenceCsv } from "./intelligence-export";
import {
  DETENTION_DESK_PATH,
  DETENTION_POLICIES_PATH,
  INDUSTRY_DETAINED_SHARE,
  amount,
  hours,
  keptAfterDriverPay,
  shareOf,
} from "./intelligence-model";
import { Narrative, money, percent, type NarrativeTargets } from "./intelligence-parts";

/** When the figures were read, moving on the minute without redrawing the rest of the page. */
function UpdatedAgo({ at }: { at: number }) {
  const t = useT();
  const now = useNow("minute");
  return (
    <span className="text-muted-foreground text-xs">
      {t("Updated {0}", formatRelativeTime(Math.min(Math.round(at / 1000) - now, 0)))}
    </span>
  );
}

/**
 * Nova's briefing on the window, written on the server from the same figures as the page: the
 * wait, the schedule, the money and what moved, read as one passage. Export and when the figures
 * were read sit beside it.
 */
export function IntelligenceBrief({
  data,
  targets,
  updatedAt,
}: {
  data: DetentionIntelligence;
  targets: NarrativeTargets;
  /** When the read came back, in milliseconds. */
  updatedAt: number;
}) {
  const t = useT();
  const segments = data.summary.flatMap((paragraph, index) =>
    index === 0
      ? paragraph.segments
      : [
          { text: " ", strong: false, tone: null, target: null, targetId: null },
          ...paragraph.segments,
        ],
  );

  return (
    <section
      aria-label={t("Summary")}
      className="grid items-center gap-x-8 gap-y-3 md:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="flex min-w-0 flex-col gap-2">
        <span className="text-muted-foreground flex items-center gap-2 text-xs">
          <NovaBadge size="sm" />
          <span className="text-foreground font-medium">{t("Nova")}</span>
          {t("Detention intelligence")}
        </span>
        <p className="text-muted-foreground text-lg leading-relaxed text-pretty">
          <Narrative segments={segments} targets={targets} />
        </p>
      </div>
      <div className="flex items-center gap-3 md:flex-col md:items-end md:gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={data.facilities.length === 0}
          onClick={() =>
            downloadCsv(intelligenceCsv(data, t), exportFilename("detention-intelligence"))
          }
        >
          <Download01Icon className="size-3.5" />
          {t("Export")}
        </Button>
        <UpdatedAgo at={updatedAt} />
      </div>
    </section>
  );
}

/** The bar beside "Stops detained" with a tick at the industry share, so the figure has a scale. */
function Benchmark({ share }: { share: number }) {
  const t = useT();

  return (
    <span
      className="flex items-center gap-2"
      title={t("ATRI, 2023: drivers reported detention at 39.3% of stops")}
    >
      <span className="bg-muted relative h-1 w-24 rounded-full">
        <span
          className="bg-foreground/70 absolute inset-y-0 left-0 rounded-full"
          style={{ width: `${Math.min(share, 1) * 100}%` }}
        />
        <span
          aria-hidden
          className="bg-muted-foreground absolute -inset-y-1 w-px"
          style={{ left: `${INDUSTRY_DETAINED_SHARE * 100}%` }}
        />
      </span>
      {t("Industry {0}", percent(INDUSTRY_DETAINED_SHARE))}
    </span>
  );
}

export function IntelligenceFigures({ data }: { data: DetentionIntelligence }) {
  const t = useT();
  const { totals, previous, recovery } = data;
  const delta =
    previous.pastFreeMinutes > 0
      ? Math.round(
          ((totals.pastFreeMinutes - previous.pastFreeMinutes) / previous.pastFreeMinutes) * 100,
        )
      : null;
  const detainedShare = shareOf(totals.detainedCount, totals.stopCount);
  const accrued = amount(recovery.accrued);
  const net = keptAfterDriverPay(data);

  return (
    <KpiStrip bare aria-label={t("Detention figures")}>
      <KpiStripItem
        size="lg"
        label={t("Hours past free time")}
        value={hours(totals.pastFreeMinutes).toLocaleString()}
        delta={delta}
        deltaLabel="%"
        deltaTone={delta !== null && delta > 0 ? "danger" : "success"}
        sub={t(
          "{0} min per detained stop",
          Math.round(shareOf(totals.pastFreeMinutes, totals.detainedCount)),
        )}
      />
      <KpiStripItem
        size="lg"
        label={t("Stops detained")}
        value={percent(detainedShare)}
        sub={<Benchmark share={detainedShare} />}
      />
      <KpiStripItem
        size="lg"
        label={t("The dock's share")}
        value={percent(shareOf(totals.dockMinutes, totals.pastFreeMinutes))}
        sub={t("{0} h with the truck on time", hours(totals.dockMinutes).toLocaleString())}
      />
      <KpiStripItem
        size="lg"
        label={t("Recovered")}
        value={percent(shareOf(amount(recovery.billed), accrued))}
        sub={t("{0} billed of {1}", money(amount(recovery.billed)), money(accrued))}
      />
      <KpiStripItem
        size="lg"
        label={t("Kept after driver pay")}
        value={
          net === null ? (
            <WithheldValue group="Margins" />
          ) : (
            <span className={net < 0 ? "text-danger-foreground" : undefined}>{money(net)}</span>
          )
        }
        sub={
          net === null
            ? t("Needs access to margins")
            : t("{0} of billed", percent(shareOf(net, amount(recovery.billed))))
        }
      />
    </KpiStrip>
  );
}

const STEP_LEVERS = defineLabels({
  ShipperReview: "Shipper",
  FreeTime: "Policy",
  Notice: "Billing",
  CustomerTerms: "Customer",
  LateArrivals: "Dispatch",
  EarlyArrivals: "Dispatch",
});

const STEP_ACTIONS = defineLabels({
  ShipperReview: "Open scorecard",
  FreeTime: "Backtest a policy",
  Notice: "Open the detention desk",
  CustomerTerms: "See customers",
  LateArrivals: "Open scorecard",
  EarlyArrivals: "Open scorecard",
});

function StepImpact({ step }: { step: DetentionNextStep }) {
  const t = useT();
  const value =
    step.impactMinutes !== null && step.impactMinutes !== undefined
      ? t("{0} h", hours(step.impactMinutes).toLocaleString())
      : step.impactAmount
        ? money(Math.abs(amount(step.impactAmount)))
        : null;

  return (
    <div className="min-w-24 text-right">
      {value ? <p className="font-mono text-sm font-semibold tabular-nums">{value}</p> : null}
      <p className="text-muted-foreground text-2xs">{step.impactCaption}</p>
    </div>
  );
}

/**
 * At most four moves the figures point to, ranked on the server by what each would recover.
 * Each opens the screen that acts on it.
 */
export function NextSteps({
  steps,
  targets,
}: {
  steps: readonly DetentionNextStep[];
  targets: NarrativeTargets;
}) {
  const t = useT();
  const navigate = useNavigate();
  if (steps.length === 0) {
    return null;
  }

  const act = (step: DetentionNextStep) => {
    switch (step.kind) {
      case "FreeTime":
        void navigate(DETENTION_POLICIES_PATH);
        return;
      case "Notice":
        void navigate(DETENTION_DESK_PATH);
        return;
      case "CustomerTerms":
        targets.onSection("customers");
        return;
      default:
        if (step.locationId) targets.onFacility(step.locationId);
    }
  };

  return (
    <section aria-label={t("What to do next")} className="bg-card rounded-lg border">
      <header className="flex items-baseline gap-3 border-b px-4 py-2.5">
        <h2 className="text-sm font-semibold">{t("What to do next")}</h2>
        <span className="text-muted-foreground text-xs">
          {t("Ranked by the hours or dollars each would recover")}
        </span>
      </header>
      <ol className="divide-y">
        {steps.map((step, index) => (
          <li
            key={`${step.kind}-${step.locationId ?? step.customerId ?? index}`}
            className="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3"
          >
            <span className="bg-muted text-muted-foreground grid size-6 shrink-0 place-items-center rounded-md font-mono text-xs font-semibold">
              {index + 1}
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-sm">
                <span className="bg-muted text-muted-foreground text-2xs mr-2 rounded-badge px-1.5 py-0.5 font-medium">
                  {STEP_LEVERS[step.kind]}
                </span>
                <Narrative segments={step.title} targets={targets} />
              </p>
              <p className="text-muted-foreground mt-0.5 text-xs">{step.detail}</p>
            </div>
            <StepImpact step={step} />
            <Button
              type="button"
              size="sm"
              variant="outline"
              className={cn("shrink-0")}
              onClick={() => act(step)}
            >
              {STEP_ACTIONS[step.kind]}
              <ArrowRightIcon className="size-3" />
            </Button>
          </li>
        ))}
      </ol>
    </section>
  );
}
