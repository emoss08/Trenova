import { useT } from "@trenova/shared/i18n/use-t";
import { JsonViewer } from "@/components/elements/json-viewer";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronRightIcon, CircleAlertIcon, CodeIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { argumentRows } from "./proposal-state";
import { describeToolCall, parseToolResult, type ParsedToolResult } from "./tool-presentation";

export type ToolActivityStatus = "running" | "done" | "failed" | "proposed";

export type ToolStep = {
  id: string;
  name: string;
  arguments: Record<string, unknown> | null | undefined;
  status: ToolActivityStatus;
  /** The saved or streamed result. Empty while running. */
  content: string;
};

/**
 * What the assistant did before answering, as a short timeline.
 *
 * "Which records did it read before saying that" is the first question anyone
 * asks of an answer. Each step names the record in plain words; the raw
 * arguments and result stay one click away, never hidden and never dumped
 * into the conversation.
 */
export function ToolTimeline({ steps, live = false }: { steps: ToolStep[]; live?: boolean }) {
  const t = useT();
  const running = steps.filter((step) => step.status === "running").length;
  const failed = steps.filter((step) => step.status === "failed").length;
  const [open, setOpen] = useState(live);

  const summary =
    running > 0
      ? t("{0, plural, one {Looking up # record…} other {Looking up # records…}}", steps.length)
      : failed > 0
        ? t("{0, plural, one {# lookup failed} other {# lookups failed}}", failed)
        : t("{0, plural, one {Looked up # record} other {Looked up # records}}", steps.length);

  const expanded = open || failed > 0;

  return (
    <Collapsible open={expanded} onOpenChange={setOpen} className="min-w-0">
      {/* One receipt line for the whole lookup, opened on request. The line
          reads as a fact about the answer, not as a log of the machine. */}
      <CollapsibleTrigger className="text-muted-foreground hover:text-foreground ui-focus-ring flex w-fit max-w-full items-center gap-1.5 rounded-control py-0.5 text-left text-xs transition-colors">
        {running > 0 ? (
          <Spinner className="size-3 shrink-0" />
        ) : failed > 0 ? (
          <CircleAlertIcon className="text-destructive size-3 shrink-0" />
        ) : (
          <ChevronRightIcon
            className={cn(
              "size-3 shrink-0 transition-transform duration-150",
              expanded && "rotate-90",
            )}
          />
        )}
        <span className="min-w-0 truncate">{summary}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ol className="mt-1.5 flex flex-col gap-1.5 pl-4.5">
          {steps.map((step) => (
            <ToolStepRow key={step.id} step={step} live={live} />
          ))}
        </ol>
      </CollapsibleContent>
    </Collapsible>
  );
}

function ToolStepRow({ step, live }: { step: ToolStep; live: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const description = describeToolCall(step.name, step.arguments);
  const failed = step.status === "failed";
  const expanded = open || failed;

  return (
    <m.li
      initial={live ? { opacity: 0, y: 2 } : false}
      animate={{ opacity: 1, y: 0 }}
      className="min-w-0"
    >
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={expanded}
        aria-label={t("Show details for {0}", description.title)}
        className="flex w-full items-center gap-1.5 text-left text-xs"
      >
        {failed && <CircleAlertIcon className="text-destructive size-3 shrink-0" />}
        <span className="min-w-0 flex-1 truncate">
          <span className={cn("font-medium", failed ? "text-destructive" : "text-foreground")}>
            {description.title}
          </span>
          {description.subject !== "" && (
            <span className="bg-muted text-muted-foreground ml-1.5 rounded-md px-1 py-px font-mono text-2xs">
              {description.subject}
            </span>
          )}
        </span>
        <StatusLabel status={step.status} />
      </button>
      {expanded && <ToolStepDetails step={step} />}
    </m.li>
  );
}

function ToolStepDetails({ step }: { step: ToolStep }) {
  const t = useT();
  const [raw, setRaw] = useState(false);
  const rows = argumentRows(step.arguments ?? null);
  const result = useMemo(
    () => (step.status === "running" || step.content === "" ? null : parseToolResult(step.content)),
    [step.status, step.content],
  );

  return (
    <div className="mt-1.5 flex flex-col gap-2 text-xs">
      {rows.length > 0 && (
        <section className="flex flex-col gap-1">
          <h4 className="text-muted-foreground text-xs font-medium">{t("Asked for")}</h4>
          <div className="flex flex-wrap gap-1">
            {rows.map((row) => (
              <span
                key={row.key}
                className="bg-muted/40 border-border/70 inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0.5"
              >
                <span className="text-muted-foreground font-mono">{row.key}</span>
                <span className="truncate">{row.value}</span>
              </span>
            ))}
          </div>
        </section>
      )}

      <section className="flex flex-col gap-1">
        <div className="flex items-center justify-between gap-2">
          <h4 className="text-muted-foreground text-xs font-medium">
            {step.status === "proposed" ? t("Outcome") : t("Returned")}
          </h4>
          {result?.kind === "json" && (
            <Button
              size="xs"
              variant="ghost"
              className="text-muted-foreground h-5 px-1.5 text-2xs"
              onClick={() => setRaw((value) => !value)}
            >
              <CodeIcon className="size-3" />
              {raw ? t("Show summary") : t("Show raw")}
            </Button>
          )}
        </div>
        <ToolResultBody status={step.status} result={result} raw={raw} />
      </section>
    </div>
  );
}

function ToolResultBody({
  status,
  result,
  raw,
}: {
  status: ToolActivityStatus;
  result: ParsedToolResult | null;
  raw: boolean;
}) {
  const t = useT();

  if (status === "running") {
    return <p className="text-muted-foreground">{t("Waiting for the result…")}</p>;
  }

  if (result === null) {
    return <p className="text-muted-foreground">{t("Nothing was returned.")}</p>;
  }

  switch (result.kind) {
    case "error":
      return (
        <p className="text-destructive flex items-start gap-1.5">
          <CircleAlertIcon className="mt-0.5 size-3.5 shrink-0" />
          {result.message}
        </p>
      );
    case "json":
      return raw ? (
        <div className="bg-muted/40 scrollbar-overlay max-h-72 overflow-auto rounded-md p-2">
          <JsonViewer data={result.value as never} collapsed={2} />
        </div>
      ) : (
        <RecordSummary value={result.value} />
      );
    default:
      return (
        <div className="flex flex-col gap-1">
          <pre className="bg-muted/40 scrollbar-overlay max-h-72 overflow-auto rounded-md p-2 font-mono text-xs whitespace-pre-wrap">
            {result.text}
          </pre>
          {result.truncated && (
            <p className="text-muted-foreground">
              {t("The result was cut short for the model; the record itself is complete.")}
            </p>
          )}
        </div>
      );
  }
}

const SUMMARY_LIMIT = 8;

function scalar(value: unknown): string | null {
  if (value === null || value === undefined || value === "") return null;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return null;
}

/**
 * A record as a handful of labelled values, which is how a person reads one;
 * the raw JSON is a click away for anyone who wants the rest.
 */
function RecordSummary({ value }: { value: unknown }) {
  const t = useT();

  if (Array.isArray(value)) {
    return (
      <p className="text-muted-foreground">
        {t("{0, plural, one {# record} other {# records}}", value.length)}
        {value.length > 0 && typeof value[0] === "object" && value[0] !== null
          ? ` · ${Object.keys(value[0] as object)
              .slice(0, 4)
              .join(", ")}`
          : ""}
      </p>
    );
  }

  if (typeof value !== "object" || value === null) {
    return <p className="font-mono">{String(value)}</p>;
  }

  const entries = Object.entries(value as Record<string, unknown>)
    .map(([key, entry]) => [key, scalar(entry)] as const)
    .filter((entry): entry is readonly [string, string] => entry[1] !== null)
    .slice(0, SUMMARY_LIMIT);

  if (entries.length === 0) {
    return (
      <p className="text-muted-foreground">{t("A structured record; use Show raw to read it.")}</p>
    );
  }

  return (
    <dl className="grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] gap-x-3 gap-y-0.5">
      {entries.map(([key, entry]) => (
        <div key={key} className="contents">
          <dt className="text-muted-foreground truncate">{key}</dt>
          <dd className="truncate">{entry}</dd>
        </div>
      ))}
    </dl>
  );
}

function StatusLabel({ status }: { status: ToolActivityStatus }) {
  const t = useT();

  switch (status) {
    case "running":
      return <span className="text-muted-foreground shrink-0 text-xs">{t("Running…")}</span>;
    case "failed":
      return null;
    case "proposed":
      return (
        <Badge variant="warning" className="h-4 shrink-0 px-1 text-2xs">
          {t("Needs approval")}
        </Badge>
      );
    default:
      return null;
  }
}
