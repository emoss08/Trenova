import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { unlessRefused } from "@/components/lading-editor/lading-service";
import { Beaker02Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  ladingTest,
  type LadingContext,
  type LadingLocation,
  type LadingProjectInput,
  type LadingTestReport,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";

/**
 * The project's tests and properties, run in the browser on the working copy, with the readiness
 * a version is certified against and the elements the samples carry that no map reads.
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

  const runTests = () => {
    const report = unlessRefused(() => ladingTest(edi, { project: working, context }), null);
    setRun({ report, of: working });
    onCoverage(report?.coverage ?? []);
  };

  const report = run?.report;
  const stale = run !== undefined && run.of !== working;

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex items-center gap-3">
        <Button type="button" size="sm" onClick={runTests}>
          <Beaker02Icon className="size-3.5" />
          {run ? t("Run the tests again") : t("Run the tests")}
        </Button>
        {stale ? (
          <span className="text-muted-foreground text-xs">
            {t("The files changed since these tests ran")}
          </span>
        ) : null}
      </div>
      {run === undefined ? (
        <p className="text-muted-foreground text-sm">
          {t(
            "Tests and properties run here, on the files as they are now, without saving. Certifying runs them again on the server.",
          )}
        </p>
      ) : report === null || report === undefined ? (
        <p className="text-danger text-sm">
          {t("The tests could not run. Fix the project's errors and try again.")}
        </p>
      ) : (
        <>
          <dl className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-muted-foreground text-xs">{t("Tests")}</dt>
              <dd className="tabular-nums">{report.readiness.tests}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground text-xs">{t("Failing")}</dt>
              <dd className={cn("tabular-nums", report.readiness.failing > 0 && "text-danger")}>
                {report.readiness.failing}
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground text-xs">{t("Errors")}</dt>
              <dd className={cn("tabular-nums", report.readiness.errors > 0 && "text-danger")}>
                {report.readiness.errors}
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground text-xs">{t("Flow clauses without a test")}</dt>
              <dd
                className={cn(
                  "tabular-nums",
                  report.readiness.uncoveredFlowClauses > 0 && "text-warning",
                )}
              >
                {report.readiness.uncoveredFlowClauses}
              </dd>
            </div>
          </dl>
          <p className={cn("text-sm", report.readiness.ready ? "text-success" : "text-warning")}>
            {report.readiness.ready
              ? t("Ready to certify")
              : t(
                  "Not ready to certify: fix the errors and failing tests, and test every flow clause",
                )}
          </p>
          {report.tests.length === 0 ? (
            <p className="text-muted-foreground text-sm">
              {t("The project has no tests. Add a test block to a map or flow file.")}
            </p>
          ) : (
            <ul className="flex flex-col divide-y rounded-md border">
              {report.tests.map((test) => (
                <li key={`${test.path}:${test.name}`} className="flex flex-col gap-1 px-3 py-2">
                  <button
                    type="button"
                    className="ui-focus-ring flex items-baseline gap-3 text-left text-sm"
                    onClick={() => onOpen({ path: test.path, range: test.range })}
                  >
                    <span
                      className={cn(
                        "w-12 shrink-0 text-xs",
                        test.passed ? "text-success" : "text-danger",
                      )}
                    >
                      {test.passed ? t("Passed") : t("Failed")}
                    </span>
                    <span className="min-w-0 flex-1">{test.name}</span>
                    <span className="text-muted-foreground shrink-0 text-xs">
                      {test.kind === "property"
                        ? t("{0, plural, one {# case} other {# cases}}", test.cases)
                        : test.path}
                    </span>
                  </button>
                  {test.failures.map((failure, index) => (
                    <button
                      // oxlint-disable-next-line react/no-array-index-key -- a test can fail twice at one range
                      key={`${failure.range.from}:${index}`}
                      type="button"
                      className="ui-focus-ring text-danger ml-15 text-left text-xs"
                      onClick={() => onOpen({ path: test.path, range: failure.range })}
                    >
                      {translateMessage(failure.message, t)}
                    </button>
                  ))}
                  {test.counterexample ? (
                    <p className="text-muted-foreground ml-15 font-mono text-xs">
                      {test.counterexample
                        .map((binding) => t("{0} = {1}", binding.name, binding.value))
                        .join(", ")}
                    </p>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
          {report.unread.length > 0 ? (
            <div className="flex flex-col gap-1">
              <p className="text-sm font-medium">{t("Sent by the partner, read by no map")}</p>
              <ul className="flex flex-col gap-1 text-xs">
                {report.unread.map((item) => (
                  <li
                    key={`${item.map}:${item.element}:${item.qualifier ?? ""}`}
                    className="flex gap-3"
                  >
                    <span className="w-32 shrink-0 font-mono">
                      {item.qualifier ? t("{0} ({1})", item.element, item.qualifier) : item.element}
                    </span>
                    <span className="text-muted-foreground min-w-0 flex-1">
                      {t(
                        "In {0} of {1} samples for {2}, such as {3}",
                        item.samples,
                        item.total,
                        item.map,
                        item.example,
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
