import { BillView } from "@/components/tariff-bill/bill-view";
import { refusalText } from "@/components/tariff-bill/refusal-text";
import { TARIFF_REGION_LABELS, unreachedRegions } from "@/lib/tariff-coverage";
import { lineAt, lineOf } from "@/lib/tariff-source-lines";
import {
  AlertCircleIcon,
  Beaker02Icon,
  CheckCircleIcon,
  ChevronRightIcon,
  FilePlus02Icon,
  PlusIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShownValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type {
  TariffCounterexample,
  TariffCoverage,
  TariffRange,
  TariffTestResult,
} from "@trenova/shared/types/tariff";
import { useState, type ReactNode } from "react";
import { TickedText } from "./ticked-text";
import type { TariffTestCells, TariffTestsState } from "./use-tariff-tests";

export function StatusIcon({ status }: { status: TariffTestResult["status"] }) {
  if (status === "passed") {
    return <CheckCircleIcon aria-hidden className="text-success size-3.5 shrink-0" />;
  }
  if (status === "failed") {
    return <XCircleIcon aria-hidden className="text-danger size-3.5 shrink-0" />;
  }
  return <AlertCircleIcon aria-hidden className="text-warning size-3.5 shrink-0" />;
}

function RangeButton({
  source,
  range,
  onReveal,
  children,
  className,
}: {
  source: string;
  range: TariffRange;
  onReveal: (range: TariffRange) => void;
  children: React.ReactNode;
  className?: string;
}) {
  const t = useT();
  return (
    <button
      type="button"
      onClick={() => onReveal(range)}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover flex min-w-0 items-start gap-2 rounded-sm text-left",
        className,
      )}
    >
      {children}
      <span className="text-foreground-subtle ml-auto shrink-0 font-mono text-2xs tabular-nums">
        {t("Ln {0}", lineOf(source, range.from))}
      </span>
    </button>
  );
}

/**
 * A failed property's simplest failing case: each varied value, the out-of-order pair of a
 * monotone expectation ("999 lb → $310.00, 1,000 lb → $250.00"), the refusal when the case
 * refused to rate, and "Save as test", which writes the case into the file as a `test`.
 */
export function Counterexample({
  counterexample,
  currency,
  onSaveAsTest,
}: {
  counterexample: TariffCounterexample;
  currency?: string | null;
  onSaveAsTest?: (block: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const code = {
    c: (content: ReactNode) => <code className="text-foreground font-mono">{content}</code>,
  };
  const shown = (value: {
    display: string;
    value?: TariffCounterexample["values"][number]["value"];
  }) => formatShownValue(value, t, { currency });
  const { monotone, refusal } = counterexample;
  return (
    <section
      aria-label={t("Counterexample")}
      className="border-danger-border bg-card flex flex-col gap-1.5 rounded-md border px-2 py-1.5"
    >
      <div className="flex items-center gap-2">
        <span className="text-foreground-subtle text-xs font-medium">{t("Counterexample")}</span>
        <span className="flex-1" />
        {onSaveAsTest && (
          <Button
            type="button"
            variant="outline"
            size="xs"
            className="gap-1.5"
            onClick={() => onSaveAsTest(counterexample.test)}
          >
            <FilePlus02Icon className="size-3" />
            {t("Save as test")}
          </Button>
        )}
      </div>
      {counterexample.values.length > 0 && (
        <ul aria-label={t("Varied values")} className="flex flex-wrap gap-1">
          {counterexample.values.map((value) => (
            <li
              key={value.target}
              className="rounded-badge bg-sunken text-foreground border px-1.5 font-mono text-2xs tabular-nums"
            >
              <span className="text-foreground-subtle">{value.target}</span> = {shown(value)}
            </li>
          ))}
        </ul>
      )}
      {monotone && (
        <p className="flex flex-col gap-0.5 text-xs">
          <span className="text-foreground-muted">
            {monotone.direction === "nondecreasing"
              ? rt(
                  "<c>{0}</c> falls as <c>{1}</c> grows:",
                  code,
                  monotone.expression,
                  monotone.target,
                )
              : rt(
                  "<c>{0}</c> rises as <c>{1}</c> grows:",
                  code,
                  monotone.expression,
                  monotone.target,
                )}
          </span>
          <span className="text-danger-foreground font-mono tabular-nums">
            {t(
              "{0} → {1}, {2} → {3}",
              shown(monotone.before.at),
              shown(monotone.before.result),
              shown(monotone.after.at),
              shown(monotone.after.result),
            )}
          </span>
        </p>
      )}
      {refusal && (
        <p className="text-danger-foreground text-xs">
          {t("The case refused to rate: {0}", refusalText(refusal, t))}
        </p>
      )}
    </section>
  );
}

/** How many cases a property rated, and how many its `assume` skipped. */
function caseCount(test: TariffTestResult, t: ReturnType<typeof useT>): string {
  return test.skipped > 0
    ? t("{0, plural, one {# case} other {# cases}}, {1} skipped", test.cases, test.skipped)
    : t("{0, plural, one {# case} other {# cases}}", test.cases);
}

function TestRow({
  test,
  source,
  onReveal,
  currency,
  failureAside,
  onSaveAsTest,
  cells,
}: {
  test: TariffTestResult;
  source: string;
  onReveal: (range: TariffRange) => void;
  currency?: string | null;
  failureAside?: (test: TariffTestResult) => ReactNode;
  onSaveAsTest?: (block: string) => void;
  cells: TariffTestCells;
}) {
  const t = useT();
  const [open, setOpen] = useState(test.status !== "passed");
  const waiting = test.status === "needs";
  const property = test.kind === "property";
  const expandable =
    waiting ||
    test.failures.length > 0 ||
    test.error !== null ||
    test.bill !== null ||
    test.counterexample !== null;

  return (
    <li data-status={test.status} className="flex flex-col gap-1.5 px-3 py-2">
      <div className="flex min-w-0 items-center gap-1.5">
        {expandable && (
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-expanded={open}
            aria-label={open ? t("Hide details") : t("Show details")}
            onClick={() => setOpen((current) => !current)}
          >
            <ChevronRightIcon className={cn("size-3 transition-transform", open && "rotate-90")} />
          </Button>
        )}
        <StatusIcon status={test.status} />
        <RangeButton
          source={source}
          range={test.range}
          onReveal={onReveal}
          className="flex-1 px-1 py-0.5"
        >
          <span className="flex min-w-0 items-center gap-1.5">
            <span className="truncate text-sm">{test.name}</span>
            {property && (
              <Badge variant="neutral" appearance="outline" className="shrink-0">
                {t("Property")}
              </Badge>
            )}
          </span>
        </RangeButton>
      </div>
      {property && test.status !== "needs" && (
        <p className="text-foreground-subtle pl-9 text-2xs tabular-nums">{caseCount(test, t)}</p>
      )}
      {open && (
        <div className="flex flex-col gap-2 pl-9">
          {waiting && (
            <p className="text-foreground-muted text-xs">
              {cells === "loading"
                ? t("Loading the rate table cells this test reads…")
                : t("The rate table cells this test reads could not be loaded")}
            </p>
          )}
          {test.error && (
            <p className="text-warning-foreground text-xs">
              <TickedText text={translateMessage(test.error.message, t)} />
            </p>
          )}
          {test.failures.map((failure, index) => (
            <div
              // oxlint-disable-next-line react/no-array-index-key -- a test's failures have no identity but their order
              key={index}
              className="border-danger-border bg-danger-subtle/40 flex flex-col gap-1 rounded-md border px-2 py-1.5"
            >
              <RangeButton source={source} range={failure.range} onReveal={onReveal}>
                <span className="text-danger-foreground text-xs">
                  <TickedText text={translateMessage(failure.message, t)} />
                </span>
              </RangeButton>
              {failure.values.length > 0 && (
                <ul className="flex flex-wrap gap-1">
                  {failure.values.map((value, valueIndex) => (
                    <li
                      // oxlint-disable-next-line react/no-array-index-key -- shown values follow the expectation's order
                      key={valueIndex}
                    >
                      <button
                        type="button"
                        onClick={() => onReveal(value.range)}
                        className="ui-focus-ring rounded-badge bg-card text-foreground border px-1.5 font-mono text-2xs tabular-nums"
                      >
                        <span className="text-foreground-subtle">
                          {source.slice(value.range.from, value.range.to)}
                        </span>{" "}
                        = {formatShownValue(value, t)}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))}
          {test.counterexample && (
            <Counterexample
              counterexample={test.counterexample}
              currency={currency}
              onSaveAsTest={onSaveAsTest}
            />
          )}
          {test.status !== "passed" && !waiting && failureAside?.(test)}
          {test.bill && <BillView bill={test.bill} currency={currency} onSelectRange={onReveal} />}
        </div>
      )}
    </li>
  );
}

/**
 * How much of the file the tests reach: "Tests reach 18 of 21 regions", and the regions no test
 * or property case ran, each linked to its source.
 */
export function CoverageSummary({
  coverage,
  source,
  onReveal,
}: {
  coverage: TariffCoverage;
  source: string;
  onReveal: (range: TariffRange) => void;
}) {
  const t = useT();
  const unreached = unreachedRegions(coverage);
  const [open, setOpen] = useState(false);
  const complete = coverage.reached === coverage.total;
  return (
    <section aria-label={t("Coverage")} className="flex flex-col gap-1 border-b px-3 py-2 text-xs">
      <div className="flex items-center gap-2">
        <span
          aria-hidden
          className={cn(
            "size-1.5 shrink-0 rounded-full",
            complete ? "bg-success" : "bg-editor-coverage-gap",
          )}
        />
        <span className={cn(complete ? "text-foreground-muted" : "text-foreground")}>
          {t("Tests reach {0} of {1} regions", coverage.reached, coverage.total)}
        </span>
        <span className="flex-1" />
        {unreached.length > 0 && (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            aria-expanded={open}
            onClick={() => setOpen((current) => !current)}
          >
            {open
              ? t("Hide them")
              : t(
                  "{0, plural, one {Show # unreached region} other {Show # unreached regions}}",
                  unreached.length,
                )}
          </Button>
        )}
      </div>
      {open && (
        <ul aria-label={t("Regions no test reaches")} className="flex flex-col">
          {unreached.map((region) => (
            <li key={`${region.kind}:${region.range.from}`}>
              <RangeButton
                source={source}
                range={region.range}
                onReveal={onReveal}
                className="w-full px-1 py-0.5"
              >
                <span className="text-foreground shrink-0">
                  {TARIFF_REGION_LABELS[region.kind]}
                </span>
                <code className="text-foreground-subtle min-w-0 truncate font-mono text-2xs">
                  {lineAt(source, region.range.from)}
                </code>
              </RangeButton>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export type TestsPanelProps = {
  state: TariffTestsState;
  source: string;
  onReveal: (range: TariffRange) => void;
  /** Adds a `test` block to the source; offered in the empty state. */
  onAddTest?: () => void;
  /** Writes a failed property's case into the source as a `test`, as one undoable edit. */
  onSaveAsTest?: (block: string) => void;
  currency?: string | null;
  /** Shown under a test that fails or errors, after what it saw: a way to ask why. */
  failureAside?: (test: TariffTestResult) => ReactNode;
  className?: string;
};

/**
 * The file's `test` and `property` blocks as the browser ran them, on every change: each one's
 * verdict, a property's cases and its counterexample (with "Save as test"), the expectations
 * that failed with the values involved, the bill the test rated, and how many of the file's
 * regions the tests reach, every row linked to the source it is about.
 */
export function TestsPanel({
  state,
  source,
  onReveal,
  onAddTest,
  onSaveAsTest,
  currency,
  failureAside,
  className,
}: TestsPanelProps) {
  const t = useT();

  if (state.status === "blocked") {
    return (
      <div className={cn("flex flex-col items-start gap-1 px-4 py-6 text-sm", className)}>
        <p className="font-medium">{t("The tests wait for a clean check")}</p>
        <p className="text-foreground-muted text-xs">
          {t(
            "{0, plural, one {# error stops} other {# errors stop}} the file from running. Fix them and the tests run again.",
            state.errors,
          )}
        </p>
      </div>
    );
  }

  if (state.status === "empty") {
    return (
      <div className={cn("flex flex-col items-start gap-2 px-4 py-6", className)}>
        <Beaker02Icon className="text-foreground-subtle size-5" />
        <p className="text-sm font-medium">{t("No tests yet")}</p>
        <p className="text-foreground-muted text-xs">
          {t(
            "A test sets a few facts with given and checks the bill with expect. A property varies a fact over a range and checks a rule holds for every case. Both run on every change and must pass before review.",
          )}
        </p>
        {onAddTest && (
          <Button type="button" variant="outline" size="xs" className="gap-1.5" onClick={onAddTest}>
            <PlusIcon className="size-3" />
            {t("Add a test")}
          </Button>
        )}
      </div>
    );
  }

  const { run, passed, failed, stale, cells, retryCells } = state;
  return (
    <div className={cn("flex flex-col", className)} aria-busy={stale || undefined}>
      <div className="flex items-center gap-2 border-b px-3 py-2 text-xs">
        <Badge variant={failed > 0 ? "danger" : "success"}>
          {t("{0} of {1} passing", passed, passed + failed)}
        </Badge>
        {stale && <span className="text-foreground-subtle">{t("Running…")}</span>}
        <span className="flex-1" />
        {onAddTest && (
          <Button type="button" variant="ghost" size="xs" className="gap-1.5" onClick={onAddTest}>
            <PlusIcon className="size-3" />
            {t("Add a test")}
          </Button>
        )}
      </div>
      {(cells === "failed" || cells === "unresolved") && (
        <div className="flex items-center gap-2 border-b px-3 py-2 text-xs">
          <p className="text-warning-foreground flex-1">
            {cells === "failed"
              ? t("The rate table cells these tests read could not be loaded.")
              : t("These tests read more rate table cells than can be loaded at once.")}
          </p>
          {cells === "failed" && (
            <Button type="button" variant="outline" size="xs" onClick={retryCells}>
              {t("Retry")}
            </Button>
          )}
        </div>
      )}
      {run.coverage && !run.error && (
        <CoverageSummary coverage={run.coverage} source={source} onReveal={onReveal} />
      )}
      {run.error ? (
        <p className="text-warning-foreground px-3 py-3 text-xs">
          <TickedText
            text={t("The tests could not run: {0}", translateMessage(run.error.message, t))}
          />
        </p>
      ) : (
        <ul className={cn("divide-border divide-y", stale && "opacity-60")}>
          {run.tests.map((test) => (
            <TestRow
              key={`${test.name}:${test.range.from}`}
              test={test}
              source={source}
              onReveal={onReveal}
              currency={currency}
              failureAside={failureAside}
              onSaveAsTest={onSaveAsTest}
              cells={cells}
            />
          ))}
        </ul>
      )}
    </div>
  );
}
