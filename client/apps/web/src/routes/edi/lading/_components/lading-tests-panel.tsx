import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { unlessRefused } from "@/components/lading-editor/lading-service";
import {
  Beaker02Icon,
  CheckVerified02Icon,
  FileCode01Icon,
  RefreshCw01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  ladingTest,
  type LadingContext,
  type LadingLocation,
  type LadingProjectInput,
  type LadingReadiness,
  type LadingTestReport,
  type LadingTestResult,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import {
  FilterChips,
  PanelEmpty,
  PanelHeading,
  PanelToolbar,
  ProportionBar,
  StatusIcon,
} from "./lading-panel-kit";
import { byFile } from "./studio-model";

type Filter = "all" | "failed" | "passed";

/**
 * The project's tests and properties, run in the browser on the working copy: each under its
 * file with its failures and a property's counterexample, the readiness a version is certified
 * against, and the elements the samples carry that no map reads.
 */
export function LadingTestsPanel({
  working,
  context,
  onOpen,
  onCoverage,
}: {
  working: LadingProjectInput;
  context: LadingContext;
  onOpen: (location: LadingLocation) => void;
  /** Told what the last test run left uncovered, for the editor's gutter. */
  onCoverage: (coverage: LadingTestReport["coverage"]) => void;
}) {
  const t = useT();
  const edi = useEdi();
  const [run, setRun] = useState<{ report: LadingTestReport | null; of: LadingProjectInput }>();
  const [filter, setFilter] = useState<Filter>("all");

  const runTests = () => {
    const report = unlessRefused(() => ladingTest(edi, { project: working, context }), null);
    setRun({ report, of: working });
    onCoverage(report?.coverage ?? []);
  };

  if (run === undefined) {
    return (
      <PanelEmpty
        icon={Beaker02Icon}
        title={t("Run the project's tests")}
        description={t(
          "Tests and properties run here, on the files as they are now, without saving. Certifying runs them again on the server.",
        )}
        action={
          <Button type="button" size="sm" onClick={runTests}>
            <Beaker02Icon className="size-3.5" />
            {t("Run the tests")}
          </Button>
        }
      />
    );
  }

  const report = run.report;
  if (report === null) {
    return (
      <PanelEmpty
        icon={XCircleIcon}
        tone="danger"
        title={t("The tests could not run")}
        description={t("Fix the project's errors and try again.")}
        action={
          <Button type="button" size="sm" variant="outline" onClick={runTests}>
            <RefreshCw01Icon className="size-3.5" />
            {t("Run the tests again")}
          </Button>
        }
      />
    );
  }

  const stale = run.of !== working;
  const passed = report.tests.filter((test) => test.passed).length;
  const failed = report.tests.length - passed;
  const kept = report.tests.filter((test) =>
    filter === "all" ? true : filter === "passed" ? test.passed : !test.passed,
  );

  return (
    <div className="flex flex-col">
      <PanelToolbar
        end={
          <FilterChips<Filter>
            value={filter}
            label={t("Show")}
            onChange={setFilter}
            items={[
              { value: "all", label: t("All"), count: report.tests.length },
              { value: "failed", label: t("Failed"), count: failed, tone: "danger" },
              { value: "passed", label: t("Passed"), count: passed, tone: "success" },
            ]}
          />
        }
      >
        <Button type="button" size="sm" variant="outline" onClick={runTests}>
          <RefreshCw01Icon className="size-3.5" />
          {t("Run the tests again")}
        </Button>
        {stale ? (
          <span className="text-warning-foreground flex items-center gap-1.5 text-xs">
            <span aria-hidden className="bg-warning size-1.5 rounded-full" />
            {t("The files changed since these tests ran")}
          </span>
        ) : null}
      </PanelToolbar>
      <Readiness readiness={report.readiness} />
      <div className="flex flex-col gap-4 p-3">
        {report.tests.length === 0 ? (
          <PanelEmpty
            compact
            icon={Beaker02Icon}
            title={t("The project has no tests")}
            description={t("Add a test block to a map or flow file.")}
          />
        ) : (
          <>
            <div className="flex flex-col gap-1.5">
              <div className="flex items-baseline gap-3 text-xs">
                <span className="text-success tabular-nums">
                  {t("{0} passed", passed)}
                </span>
                <span className={cn("tabular-nums", failed > 0 ? "text-danger" : "text-muted-foreground")}>
                  {t("{0} failed", failed)}
                </span>
                <span className="text-muted-foreground tabular-nums">
                  {t("{0, plural, one {# test} other {# tests}}", report.tests.length)}
                </span>
              </div>
              <ProportionBar
                label={t("{0} of {1} tests passed", passed, report.tests.length)}
                parts={[
                  { value: passed, tone: "success" },
                  { value: failed, tone: "danger" },
                ]}
              />
            </div>
            {kept.length === 0 ? (
              <p className="text-muted-foreground py-4 text-center text-sm">
                {filter === "failed" ? t("No test failed") : t("No test passed")}
              </p>
            ) : (
              <TestList tests={kept} onOpen={onOpen} />
            )}
          </>
        )}
        <Unread unread={report.unread} />
      </div>
    </div>
  );
}

function TestList({
  tests,
  onOpen,
}: {
  tests: readonly LadingTestResult[];
  onOpen: (location: LadingLocation) => void;
}) {
  const t = useT();
  return (
    <div className="flex flex-col gap-3">
      {byFile(tests).map(({ path, entries }) => (
        <section key={path} className="flex flex-col">
          <h4 className="text-muted-foreground flex items-center gap-1.5 pb-1 font-mono text-xs">
            <FileCode01Icon className="size-3.5" />
            {path}
          </h4>
          <ul className="flex flex-col divide-y rounded-md border">
            {entries.map((test) => (
              <li key={test.name} className="flex flex-col">
                <button
                  type="button"
                  onClick={() => onOpen({ path: test.path, range: test.range })}
                  className="ui-focus-ring hover:bg-surface-hover flex items-center gap-2 px-3 py-1.5 text-left text-sm"
                >
                  <StatusIcon status={test.passed ? "passed" : "failed"} />
                  <span className="sr-only">{test.passed ? t("Passed") : t("Failed")}</span>
                  <span className="min-w-0 flex-1 truncate">{test.name}</span>
                  {test.kind === "property" ? (
                    <Badge variant="accent-violet">
                      {t("{0, plural, one {Property, # case} other {Property, # cases}}", test.cases)}
                    </Badge>
                  ) : null}
                </button>
                {test.failures.length > 0 || test.counterexample ? (
                  <div className="border-danger-border mb-2 ml-5.5 flex flex-col gap-1 border-l-2 pr-3 pl-3">
                    {test.failures.map((failure, index) => (
                      <button
                        // oxlint-disable-next-line react/no-array-index-key -- a test can fail twice at one range
                        key={`${failure.range.from}:${index}`}
                        type="button"
                        className="ui-focus-ring text-danger-foreground text-left text-xs hover:underline"
                        onClick={() => onOpen({ path: test.path, range: failure.range })}
                      >
                        {translateMessage(failure.message, t)}
                      </button>
                    ))}
                    {test.counterexample ? (
                      <div className="flex flex-wrap items-center gap-1.5 text-xs">
                        <span className="text-muted-foreground">{t("Fails for")}</span>
                        {test.counterexample.map((binding) => (
                          <code
                            key={binding.name}
                            className="bg-muted rounded px-1.5 py-0.5 font-mono"
                          >
                            {t("{0} = {1}", binding.name, binding.value)}
                          </code>
                        ))}
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

/**
 * What certifying asks of the version, as one strip under the toolbar: the verdict, then each
 * check with its count.
 */
function Readiness({ readiness }: { readiness: LadingReadiness }) {
  const t = useT();
  const checks = [
    {
      label: t("No errors"),
      ok: readiness.errors === 0,
      detail: t("{0, plural, one {# error} other {# errors}}", readiness.errors),
    },
    {
      label: t("Every test passes"),
      ok: readiness.failing === 0,
      detail: t("{0} of {1} failing", readiness.failing, readiness.tests),
    },
    {
      label: t("Every flow clause tested"),
      ok: readiness.uncoveredFlowClauses === 0,
      detail: t("{0} without a test", readiness.uncoveredFlowClauses),
    },
  ];
  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 border-b px-3 py-2 text-xs">
      <span
        className={cn(
          "flex items-center gap-1.5 font-medium",
          readiness.ready ? "text-success-foreground" : "text-warning-foreground",
        )}
      >
        <CheckVerified02Icon className="size-4 shrink-0" />
        {readiness.ready ? t("Ready to certify") : t("Not ready to certify")}
      </span>
      <ul className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
        {checks.map((check) => (
          <li key={check.label} className="flex items-center gap-1.5">
            <StatusIcon status={check.ok ? "passed" : "failed"} />
            <span>{check.label}</span>
            <span
              className={cn("tabular-nums", check.ok ? "text-muted-foreground" : "text-danger")}
            >
              {check.detail}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** The elements the samples carry that no map reads, with how often and an example. */
function Unread({ unread }: { unread: LadingTestReport["unread"] }) {
  const t = useT();
  if (unread.length === 0) return null;
  return (
    <section className="flex flex-col gap-2">
      <PanelHeading
        title={t("Sent by the partner, read by no map")}
        count={unread.length}
        help={t("Elements the partner's samples carry that no map reads")}
      />
      <div className="overflow-hidden rounded-md border text-xs">
        <div className="bg-muted text-muted-foreground grid grid-cols-[8rem_minmax(0,1fr)_9rem_minmax(0,1fr)] gap-3 border-b px-3 py-1.5 font-medium">
          <span>{t("Element")}</span>
          <span>{t("Map")}</span>
          <span>{t("Seen in")}</span>
          <span>{t("Example")}</span>
        </div>
        <ul className="divide-y">
          {unread.map((item) => (
            <li
              key={`${item.map}:${item.element}:${item.qualifier ?? ""}`}
              className="grid grid-cols-[8rem_minmax(0,1fr)_9rem_minmax(0,1fr)] items-center gap-3 px-3 py-1.5"
            >
              <span className="font-mono">
                {item.qualifier ? t("{0} ({1})", item.element, item.qualifier) : item.element}
              </span>
              <span className="truncate font-mono">{item.map}</span>
              <span className="flex items-center gap-2">
                <ProportionBar
                  className="w-12"
                  label={t("{0} of {1} samples", item.samples, item.total)}
                  total={item.total}
                  parts={[{ value: item.samples, tone: "info" }]}
                />
                <span className="tabular-nums">{t("{0} of {1}", item.samples, item.total)}</span>
              </span>
              <span className="text-muted-foreground truncate font-mono" title={item.example}>
                {item.example}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
