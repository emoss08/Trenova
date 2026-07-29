import { useDebounce } from "@/hooks/use-debounce";
import { pricingFingerprint } from "@/lib/detention-policy";
import { CalculationReceipt } from "@/routes/detention-desk/_components/calculation-receipt";
import { apiService } from "@/services/api";
import { Card } from "@trenova/shared/components/ui/card";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  CAP_KIND_LABEL,
  deltaToneClass,
  formatDetentionMinutes,
  formatSignedCurrency,
  OCCURRENCE_STATUS_LABEL,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  DetentionPolicy,
  PreviewResult,
  PreviewScenario,
} from "@trenova/shared/types/detention";
import { useQueries } from "@tanstack/react-query";
import { ChevronDownIcon, FlaskConicalIcon, TriangleAlertIcon } from "lucide-react";
import { AnimatePresence, m, useInView } from "motion/react";
import { useMemo, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * Worked scenarios the preview runs on every change. They are chosen to expose
 * the terms operators most often misconfigure: a stop that lands just inside
 * free time, one that runs long enough to reach a second rate tier, and a late
 * arrival that exercises the entitlement rule.
 */
const SCENARIO_BASE = 1_767_225_600;
const PREVIEW_DEBOUNCE_MS = 400;

type ScenarioDefinition = {
  key: string;
  label: string;
  headline: string;
  description: string;
  scenario: PreviewScenario;
};

const SCENARIOS: ScenarioDefinition[] = [
  {
    key: "just-over",
    label: "Just over",
    headline: "Just over free time",
    description: "On-time arrival, departs 2h 10m later",
    scenario: {
      arrivedAt: SCENARIO_BASE,
      departedAt: SCENARIO_BASE + 130 * 60,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
  {
    key: "long-hold",
    label: "Long hold",
    headline: "Long hold",
    description: "On-time arrival, departs 7h later",
    scenario: {
      arrivedAt: SCENARIO_BASE,
      departedAt: SCENARIO_BASE + 7 * 3600,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
  {
    key: "late-arrival",
    label: "Late arrival",
    headline: "Late arrival",
    description: "Arrives 90m after the window, departs 5h later",
    scenario: {
      arrivedAt: SCENARIO_BASE + 150 * 60,
      departedAt: SCENARIO_BASE + 150 * 60 + 5 * 3600,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
];

const BAR_TRANSITION = { duration: 0.45, ease: [0.22, 1, 0.36, 1] } as const;

type Segment = {
  key: string;
  minutes: number;
  className: string;
  legend: string;
};

/**
 * Splits a priced scenario into the bands an operator argues about: the free
 * time the contract gave away, what is actually billed, how much of that came
 * from rounding rather than dwell, and the tail that dwelled but bills nothing.
 */
function dwellSegments(result: PreviewResult, suppressed: boolean): Segment[] {
  const freeUsed = Math.min(result.freeMinutesGranted, result.rawDwellMinutes);
  const uplift = Math.max(0, result.roundedMinutes - result.billableMinutes);
  const billed = Math.min(result.roundedMinutes, result.billableMinutes);
  const unbilled = Math.max(0, result.rawDwellMinutes - freeUsed - result.billableMinutes);

  return [
    {
      key: "free",
      minutes: freeUsed,
      className: "bg-foreground/20",
      legend: "Free time",
    },
    {
      key: "billed",
      minutes: billed,
      className: suppressed
        ? "bg-red-500/60 dark:bg-red-400/60"
        : "bg-emerald-500 dark:bg-emerald-400",
      legend: "Billed",
    },
    {
      key: "rounding",
      minutes: uplift,
      className: suppressed
        ? "bg-red-500/30 dark:bg-red-400/30"
        : "bg-emerald-500/45 dark:bg-emerald-400/45",
      legend: "Rounding",
    },
    {
      key: "unbilled",
      minutes: unbilled,
      className: "bg-muted-foreground/15",
      legend: "Not billed",
    },
  ].filter((segment) => segment.minutes > 0);
}

function DwellBar({ result, suppressed }: { result: PreviewResult; suppressed: boolean }) {
  const segments = dwellSegments(result, suppressed);
  const total = segments.reduce((sum, segment) => sum + segment.minutes, 0);

  if (total === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-1.5 w-full overflow-hidden rounded-full bg-muted">
        {segments.map((segment) => (
          <m.div
            key={segment.key}
            className={cn("h-full first:rounded-l-full last:rounded-r-full", segment.className)}
            initial={{ width: 0 }}
            animate={{ width: `${(segment.minutes / total) * 100}%` }}
            transition={BAR_TRANSITION}
          />
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        {segments.map((segment) => (
          <div key={segment.key} className="flex items-center gap-1.5">
            <span className={cn("size-1.5 shrink-0 rounded-full", segment.className)} />
            <span className="text-2xs text-muted-foreground">
              {segment.legend}
              <span className="ml-1 tabular-nums text-foreground/70">
                {formatDetentionMinutes(segment.minutes)}
              </span>
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function StatCell({
  label,
  value,
  className,
}: {
  label: string;
  value: string;
  className?: string;
}) {
  return (
    <div className="px-4 py-2.5">
      <p className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">{label}</p>
      <p className={cn("mt-0.5 text-sm font-medium tabular-nums", className)}>{value}</p>
    </div>
  );
}

function Signal({ tone, children }: { tone: "neutral" | "warn" | "bad"; children: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-md px-1.5 py-0.5 text-2xs font-medium",
        tone === "neutral" && "bg-muted text-muted-foreground",
        tone === "warn" && "bg-amber-500/15 text-amber-700 dark:text-amber-400",
        tone === "bad" && "bg-red-500/15 text-red-700 dark:text-red-400",
      )}
    >
      {children}
    </span>
  );
}

function ScenarioResult({
  definition,
  result,
}: {
  definition: ScenarioDefinition;
  result: PreviewResult;
}) {
  const [receiptOpen, setReceiptOpen] = useState(false);
  const suppressed = result.suppressedByGate;
  const currency = result.policySnapshot.currency || "USD";

  return (
    <>
      <div className="px-4 py-3.5">
        <div className="flex items-end justify-between gap-3">
          <div className="min-w-0">
            <m.p
              key={`${definition.key}-${result.billableAmount}`}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className={cn(
                "text-3xl leading-none font-semibold tracking-tight tabular-nums",
                suppressed && "text-muted-foreground line-through decoration-1",
              )}
            >
              {formatCurrency(result.billableAmount, currency)}
            </m.p>
            <p className="mt-1.5 text-xs text-muted-foreground tabular-nums">
              {formatDetentionMinutes(result.roundedMinutes)} billed of{" "}
              {formatDetentionMinutes(result.rawDwellMinutes)} dwell
            </p>
          </div>
          <div className="shrink-0 text-right">
            <p className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
              Net margin
            </p>
            <p
              className={cn(
                "mt-0.5 text-sm font-medium tabular-nums",
                deltaToneClass(result.netMargin),
              )}
            >
              {formatSignedCurrency(result.netMargin, currency)}
            </p>
          </div>
        </div>

        <div className="mt-3.5">
          <DwellBar result={result} suppressed={suppressed} />
        </div>

        {(suppressed ||
          result.arrivedLate ||
          result.capApplied !== "None" ||
          result.netMargin < 0 ||
          result.billableAmount === 0) && (
          <div className="mt-3 flex flex-wrap gap-1.5">
            {suppressed && <Signal tone="bad">Suppressed — notice missed</Signal>}
            {result.netMargin < 0 && (
              <Signal tone="warn">
                {`Pays out ${formatCurrency(Math.abs(result.netMargin), currency)} more than it bills`}
              </Signal>
            )}
            {result.arrivedLate && <Signal tone="neutral">Arrived late</Signal>}
            {result.capApplied !== "None" && (
              <Signal tone="neutral">{CAP_KIND_LABEL[result.capApplied]}</Signal>
            )}
            {result.billableAmount === 0 && !suppressed && (
              <Signal tone="neutral">Nothing billable</Signal>
            )}
          </div>
        )}
      </div>

      <div className="grid grid-cols-3 divide-x divide-border border-t border-border">
        <StatCell label="Gross" value={formatCurrency(result.grossAmount, currency)} />
        <StatCell label="Driver pay" value={formatCurrency(result.driverPayAmount, currency)} />
        <StatCell label="Status" value={OCCURRENCE_STATUS_LABEL[result.status]} />
      </div>

      <div className="border-t border-border">
        <button
          type="button"
          aria-expanded={receiptOpen}
          onClick={() => setReceiptOpen((open) => !open)}
          className="flex w-full items-center justify-between gap-2 px-4 py-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          <span>How this charge was calculated</span>
          <ChevronDownIcon
            className={cn("size-3.5 transition-transform", receiptOpen && "rotate-180")}
          />
        </button>
        <AnimatePresence initial={false}>
          {receiptOpen && (
            <m.div
              key="receipt"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.22, ease: "easeOut" }}
              className="overflow-hidden"
            >
              <div className="border-t border-border px-4 py-3">
                <CalculationReceipt trace={result.calculationTrace} currency={currency} />
              </div>
            </m.div>
          )}
        </AnimatePresence>
      </div>
    </>
  );
}

function LiveIndicator({ pricing }: { pricing: boolean }) {
  return (
    <div className="flex shrink-0 items-center gap-1.5">
      <m.span
        className={cn(
          "size-1.5 rounded-full",
          pricing ? "bg-amber-500" : "bg-emerald-500 dark:bg-emerald-400",
        )}
        animate={pricing ? { opacity: [1, 0.3, 1] } : { opacity: 1 }}
        transition={pricing ? { duration: 1.1, repeat: Infinity, ease: "easeInOut" } : undefined}
      />
      <span className="text-2xs text-muted-foreground">{pricing ? "Pricing" : "Live"}</span>
    </div>
  );
}

/**
 * Live preview panel. Edits on the Terms tab re-run the server's production
 * calculator (debounced), so the operator sees the money their terms produce
 * before the policy ever touches a shipment.
 */
export function DetentionPolicyPreview() {
  const { control } = useFormContext<DetentionPolicy>();
  const values = useWatch({ control });

  const rootRef = useRef<HTMLDivElement>(null);
  const isVisible = useInView(rootRef);
  const [activeKey, setActiveKey] = useState(SCENARIOS[0].key);

  const policy = useDebounce(values, PREVIEW_DEBOUNCE_MS) as DetentionPolicy;

  const missing = [
    !policy?.name && "Name",
    !policy?.code && "Code",
    !policy?.accessorialChargeId && "Accessorial charge",
  ].filter((label): label is string => Boolean(label));
  const ready = missing.length === 0;

  const fingerprint = useMemo(() => pricingFingerprint(policy ?? {}), [policy]);

  const results = useQueries({
    // The fingerprint is the pricing-relevant projection of the policy; keying on the whole
    // object would re-price every time a description is typed.
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queries: SCENARIOS.map((definition) => ({
      queryKey: ["detention", "preview", definition.key, definition.scenario, fingerprint],
      queryFn: () => apiService.detentionPolicyService.preview(policy, definition.scenario),
      enabled: ready && isVisible,
      retry: false,
      staleTime: Infinity,
    })),
  });

  const activeIndex = Math.max(
    0,
    SCENARIOS.findIndex((definition) => definition.key === activeKey),
  );
  const definition = SCENARIOS[activeIndex];
  const active = results[activeIndex];

  const items: SegmentedControlItem<string>[] = SCENARIOS.map((scenario, index) => {
    const query = results[index];

    return {
      value: scenario.key,
      label: scenario.label,
      caption: query.data
        ? formatCurrency(query.data.billableAmount, query.data.policySnapshot.currency || "USD")
        : query.isError
          ? "—"
          : "···",
    };
  });

  return (
    <div ref={rootRef} className="flex flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-2xs font-semibold tracking-wide text-muted-foreground uppercase">
            Live preview
          </h3>
          <p className="mt-1 text-xs text-muted-foreground">
            Worked examples priced by the same engine that bills real shipments.
          </p>
        </div>
        {ready && <LiveIndicator pricing={results.some((query) => query.isFetching)} />}
      </div>

      {!ready ? (
        <div className="flex flex-col items-center gap-2.5 rounded-lg border border-dashed border-border py-10 text-center">
          <FlaskConicalIcon className="size-5 text-muted-foreground/60" />
          <p className="max-w-[16rem] text-xs text-muted-foreground">
            Finish these fields on the Terms tab to price the worked examples.
          </p>
          <div className="flex flex-wrap justify-center gap-1.5">
            {missing.map((label) => (
              <span
                key={label}
                className="rounded-md bg-muted px-1.5 py-0.5 text-2xs font-medium text-muted-foreground"
              >
                {label}
              </span>
            ))}
          </div>
        </div>
      ) : (
        <>
          <SegmentedControl
            items={items}
            value={activeKey}
            onValueChange={setActiveKey}
            aria-label="Preview scenario"
            fullWidth
          />

          <Card className="gap-0 overflow-hidden rounded-lg p-0">
            <div className="flex items-baseline justify-between gap-3 border-b border-border px-4 py-2.5">
              <p className="truncate text-xs font-medium">{definition.headline}</p>
              <p className="shrink-0 text-2xs text-muted-foreground">{definition.description}</p>
            </div>

            {active.isError ? (
              <div className="flex items-start gap-2.5 px-4 py-6">
                <TriangleAlertIcon className="mt-px size-4 shrink-0 text-amber-500" />
                <div className="min-w-0">
                  <p className="text-xs font-medium">This scenario could not be priced</p>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {active.error instanceof Error
                      ? active.error.message
                      : "Resolve the validation errors on the Terms tab to see this scenario."}
                  </p>
                </div>
              </div>
            ) : active.data ? (
              <ScenarioResult definition={definition} result={active.data} />
            ) : (
              <div className="flex flex-col gap-3 px-4 py-3.5">
                <Skeleton className="h-8 w-32 rounded-md" />
                <Skeleton className="h-1.5 w-full rounded-full" />
                <Skeleton className="h-3 w-48 rounded-md" />
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
