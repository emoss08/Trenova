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
import {
  CheckIcon,
  ChevronDownIcon,
  CircleAlertIcon,
  CodeIcon,
  SearchIcon,
  ShieldQuestionIcon,
} from "lucide-react";
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

  return (
    <Collapsible
      open={open || (live && failed > 0)}
      onOpenChange={setOpen}
      className="border-border/70 bg-card/60 max-w-[92%] overflow-hidden rounded-xl border"
    >
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs transition-colors">
        {running > 0 ? (
          <Spinner className="size-3.5 shrink-0" />
        ) : failed > 0 ? (
          <CircleAlertIcon className="text-destructive size-3.5 shrink-0" />
        ) : (
          <SearchIcon className="text-muted-foreground size-3.5 shrink-0" />
        )}
        <span className="text-muted-foreground min-w-0 flex-1 truncate font-medium">{summary}</span>
        <ChevronDownIcon
          className={cn(
            "text-muted-foreground size-3.5 shrink-0 transition-transform",
            (open || (live && failed > 0)) && "rotate-180",
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="border-border/70 border-t">
        <ol className="flex flex-col px-3 py-2">
          {steps.map((step, index) => (
            <ToolStepRow key={step.id} step={step} last={index === steps.length - 1} live={live} />
          ))}
        </ol>
      </CollapsibleContent>
    </Collapsible>
  );
}

function ToolStepRow({ step, last, live }: { step: ToolStep; last: boolean; live: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const description = describeToolCall(step.name, step.arguments);
  const expanded = open || (live && step.status === "failed");

  return (
    <m.li
      initial={live ? { opacity: 0, y: 4 } : false}
      animate={{ opacity: 1, y: 0 }}
      className="relative flex gap-2.5 pb-2 last:pb-0"
    >
      <span className="flex flex-col items-center">
        <StatusDot status={step.status} />
        {!last && <span aria-hidden className="bg-border mt-1 w-px flex-1" />}
      </span>
      <div className="min-w-0 flex-1">
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          aria-expanded={expanded}
          aria-label={t("Show details for {0}", description.title)}
          className="flex w-full items-center gap-2 text-left text-xs"
        >
          <span className="min-w-0 flex-1 truncate">
            <span className="font-medium">{description.title}</span>
            {description.subject !== "" && (
              <span className="text-muted-foreground"> · {description.subject}</span>
            )}
          </span>
          <StatusLabel status={step.status} />
        </button>
        {expanded && <ToolStepDetails step={step} />}
      </div>
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
    <div className="bg-muted/30 mt-1.5 flex flex-col gap-2.5 rounded-lg px-2.5 py-2 text-xs">
      {rows.length > 0 && (
        <section className="flex flex-col gap-1">
          <h4 className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
            {t("Asked for")}
          </h4>
          <div className="flex flex-wrap gap-1">
            {rows.map((row) => (
              <span
                key={row.key}
                className="bg-background border-border/70 inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0.5"
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
          <h4 className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
            {step.status === "proposed" ? t("Outcome") : t("Returned")}
          </h4>
          {result?.kind === "json" && (
            <Button
              size="xs"
              variant="ghost"
              className="text-muted-foreground h-5 px-1.5 text-[10px]"
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
        <div className="bg-background max-h-72 overflow-auto rounded-md p-2">
          <JsonViewer data={result.value as never} collapsed={2} />
        </div>
      ) : (
        <RecordSummary value={result.value} />
      );
    default:
      return (
        <div className="flex flex-col gap-1">
          <pre className="bg-background max-h-72 overflow-auto rounded-md p-2 font-mono text-[11px] whitespace-pre-wrap">
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

function StatusDot({ status }: { status: ToolActivityStatus }) {
  const base = "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full";
  switch (status) {
    case "running":
      return (
        <span className={cn(base, "bg-primary/15 text-primary")}>
          <Spinner className="size-2.5" />
        </span>
      );
    case "failed":
      return (
        <span className={cn(base, "bg-destructive/15 text-destructive")}>
          <CircleAlertIcon className="size-2.5" />
        </span>
      );
    case "proposed":
      return (
        <span className={cn(base, "bg-warning/20 text-warning")}>
          <ShieldQuestionIcon className="size-2.5" />
        </span>
      );
    default:
      return (
        <span className={cn(base, "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400")}>
          <CheckIcon className="size-2.5" />
        </span>
      );
  }
}

function StatusLabel({ status }: { status: ToolActivityStatus }) {
  const t = useT();

  switch (status) {
    case "running":
      return <span className="text-muted-foreground shrink-0 text-[11px]">{t("Running…")}</span>;
    case "failed":
      return (
        <Badge variant="inactive" className="h-4 shrink-0 px-1 text-[10px]">
          {t("Failed")}
        </Badge>
      );
    case "proposed":
      return (
        <Badge variant="warning" className="h-4 shrink-0 px-1 text-[10px]">
          {t("Needs approval")}
        </Badge>
      );
    default:
      return null;
  }
}
