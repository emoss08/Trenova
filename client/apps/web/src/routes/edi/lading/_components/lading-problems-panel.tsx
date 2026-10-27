import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import {
  ChevronRightIcon,
  CornerDownRightIcon,
  FileCode01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import type {
  LadingLocation,
  LadingProjectInput,
  LadingSeverity,
} from "@trenova/shared/edi/lading";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import {
  FilterChips,
  PanelSheet,
  PanelToolbar,
  SeverityIcon,
} from "./lading-panel-kit";
import { ProblemsSketch } from "./lading-panel-sketches";
import { byFile, lineOf, type Problem } from "./studio-model";

type Filter = "all" | LadingSeverity;

/**
 * Every finding in the project, under the file it is in: errors first, each with its code, its
 * line, the places it relates to and how many quick fixes it has. A chip keeps one severity.
 */
export function LadingProblemsPanel({
  problems,
  working,
  onOpen,
}: {
  problems: readonly Problem[];
  working: LadingProjectInput;
  onOpen: (location: LadingLocation) => void;
}) {
  const t = useT();
  const [filter, setFilter] = useState<Filter>("all");
  const [closed, setClosed] = useState<ReadonlySet<string>>(() => new Set());

  if (problems.length === 0) {
    return (
      <PanelSheet
        sketch={<ProblemsSketch clean />}
        title={t("No problems")}
        description={t(
          "Every file checks clean. Findings show here as you type, under the file they are in.",
        )}
      />
    );
  }

  const count = (severity: LadingSeverity) =>
    problems.filter((problem) => problem.severity === severity).length;
  const kept = filter === "all" ? problems : problems.filter((p) => p.severity === filter);
  const files = byFile(kept);
  const sourceOf = (path: string) => working.files.find((file) => file.path === path)?.source ?? "";

  return (
    <div className="flex flex-col">
      <PanelToolbar
        end={
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {In # file} other {In # files}}", files.length)}
          </span>
        }
      >
        <FilterChips<Filter>
          value={filter}
          label={t("Severity")}
          onChange={setFilter}
          items={[
            { value: "all", label: t("All"), count: problems.length },
            { value: "error", label: t("Errors"), count: count("error"), tone: "danger" },
            { value: "warning", label: t("Warnings"), count: count("warning"), tone: "warning" },
            { value: "info", label: t("Notes"), count: count("info"), tone: "info" },
          ]}
        />
      </PanelToolbar>
      {files.length === 0 ? (
        <p className="text-muted-foreground px-3 py-6 text-center text-sm">
          {t("Nothing at this severity")}
        </p>
      ) : (
        <ul className="flex flex-col py-1">
          {files.map(({ path, entries: inFile }) => {
            const open = !closed.has(path);
            const source = sourceOf(path);
            const errors = inFile.filter((problem) => problem.severity === "error").length;
            const warnings = inFile.filter((problem) => problem.severity === "warning").length;
            return (
              <li key={path}>
                <button
                  type="button"
                  aria-expanded={open}
                  onClick={() =>
                    setClosed((current) => {
                      const next = new Set(current);
                      if (open) next.add(path);
                      else next.delete(path);
                      return next;
                    })
                  }
                  className="ui-focus-ring hover:bg-surface-hover flex w-full items-center gap-1.5 px-2 py-1 text-left"
                >
                  <ChevronRightIcon
                    className={cn(
                      "text-muted-foreground size-3.5 shrink-0 transition-transform",
                      open && "rotate-90",
                    )}
                  />
                  <FileCode01Icon className="text-muted-foreground size-3.5 shrink-0" />
                  <span className="min-w-0 truncate font-mono text-xs">{path}</span>
                  {errors > 0 ? (
                    <Badge variant="danger">
                      {t("{0, plural, one {# error} other {# errors}}", errors)}
                    </Badge>
                  ) : null}
                  {warnings > 0 ? (
                    <Badge variant="warning">
                      {t("{0, plural, one {# warning} other {# warnings}}", warnings)}
                    </Badge>
                  ) : null}
                </button>
                {open ? (
                  <ul className="flex flex-col">
                    {inFile.map((problem, index) => (
                      <li
                        // oxlint-disable-next-line react/no-array-index-key -- two findings can share a file, code and range
                        key={`${problem.range.from}:${problem.code}:${index}`}
                      >
                        <button
                          type="button"
                          onClick={() => onOpen({ path, range: problem.range })}
                          className="ui-focus-ring hover:bg-surface-hover flex w-full items-start gap-2 py-1 pr-3 pl-9 text-left text-sm"
                        >
                          <SeverityIcon severity={problem.severity} className="mt-0.5" />
                          <span className="min-w-0 flex-1">
                            {translateMessage(problem.message, t)}
                          </span>
                          {problem.fixes.length > 0 ? (
                            <Badge variant="neutral" appearance="outline">
                              {t(
                                "{0, plural, one {# quick fix} other {# quick fixes}}",
                                problem.fixes.length,
                              )}
                            </Badge>
                          ) : null}
                          <span className="text-muted-foreground shrink-0 font-mono text-xs">
                            {problem.code}
                          </span>
                          <span className="text-muted-foreground w-14 shrink-0 text-right text-xs tabular-nums">
                            {t("Ln {0}", lineOf(source, problem.range.from))}
                          </span>
                        </button>
                        {problem.related.map((related, at) => (
                          <button
                            // oxlint-disable-next-line react/no-array-index-key -- related places can repeat
                            key={at}
                            type="button"
                            onClick={() => onOpen({ path, range: related.range })}
                            className="ui-focus-ring hover:bg-surface-hover text-muted-foreground flex w-full items-start gap-2 py-0.5 pr-3 pl-14 text-left text-xs"
                          >
                            <CornerDownRightIcon className="mt-0.5 size-3 shrink-0" />
                            <span className="min-w-0 flex-1">
                              {translateMessage(related.message, t)}
                            </span>
                            <span className="w-14 shrink-0 text-right tabular-nums">
                              {t("Ln {0}", lineOf(source, related.range.from))}
                            </span>
                          </button>
                        ))}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
