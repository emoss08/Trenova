import { BillView } from "@/components/tariff-bill/bill-view";
import {
  AlertCircleIcon,
  Beaker02Icon,
  CheckCircleIcon,
  ChevronRightIcon,
  PlusIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffRange, TariffTestResult } from "@trenova/shared/types/tariff";
import { useState } from "react";
import type { TariffTestsState } from "./use-tariff-tests";

/** The 1-based line a position falls on. */
export function lineOf(source: string, offset: number): number {
  let line = 1;
  const end = Math.min(offset, source.length);
  for (let index = 0; index < end; index += 1) if (source.charCodeAt(index) === 10) line += 1;
  return line;
}

function StatusIcon({ status }: { status: TariffTestResult["status"] }) {
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

function TestRow({
  test,
  source,
  onReveal,
  currency,
}: {
  test: TariffTestResult;
  source: string;
  onReveal: (range: TariffRange) => void;
  currency?: string | null;
}) {
  const t = useT();
  const [open, setOpen] = useState(test.status !== "passed");
  const expandable = test.failures.length > 0 || test.error !== null || test.bill !== null;

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
          <span className="truncate text-sm">{test.name}</span>
        </RangeButton>
      </div>
      {open && (
        <div className="flex flex-col gap-2 pl-9">
          {test.error && (
            <p className="text-warning-foreground text-xs">
              {translateMessage(test.error.message, t)}
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
                  {translateMessage(failure.message, t)}
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
                        = {value.display}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))}
          {test.bill && <BillView bill={test.bill} currency={currency} onSelectRange={onReveal} />}
        </div>
      )}
    </li>
  );
}

export type TestsPanelProps = {
  state: TariffTestsState;
  source: string;
  onReveal: (range: TariffRange) => void;
  /** Adds a `test` block to the source; offered in the empty state. */
  onAddTest?: () => void;
  currency?: string | null;
  className?: string;
};

/**
 * The file's `test` blocks as the browser ran them, on every change: each test's verdict, the
 * expectations that failed with the values involved, and the bill the test rated, every row
 * linked to the source it is about.
 */
export function TestsPanel({
  state,
  source,
  onReveal,
  onAddTest,
  currency,
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
            "A test sets a few facts with given and checks the bill with expect. Tests run on every change and must pass before review.",
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

  const { run, passed, failed, stale } = state;
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
      {run.error ? (
        <p className="text-warning-foreground px-3 py-3 text-xs">
          {t("The tests could not run: {0}", translateMessage(run.error.message, t))}
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
            />
          ))}
        </ul>
      )}
    </div>
  );
}
