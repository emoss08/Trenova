import { useT } from "@trenova/shared/i18n/use-t";
import { JsonViewer } from "@/components/elements/json-viewer";
import { Badge } from "@trenova/shared/components/ui/badge";
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
  SearchIcon,
  ShieldQuestionIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { argumentRows } from "./proposal-state";
import { describeToolCall, parseToolResult } from "./tool-presentation";

export type ToolActivityStatus = "running" | "done" | "failed" | "proposed";

export type ToolActivityProps = {
  name: string;
  arguments: Record<string, unknown> | null | undefined;
  status: ToolActivityStatus;
  /** The saved or streamed result. Empty while running. */
  content: string;
  /** Live activity opens itself when it finishes; history stays folded. */
  live?: boolean;
};

/**
 * One lookup, as a row the reader can open.
 *
 * "Which records did it read before saying that" is the first question anyone
 * asks of an answer, so the row names the record in plain words and keeps the
 * raw arguments and result one click away rather than hidden or spread across
 * the conversation.
 */
export function ToolActivity({
  name,
  arguments: args,
  status,
  content,
  live = false,
}: ToolActivityProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const description = describeToolCall(name, args);
  const rows = argumentRows(args ?? null);
  const result = useMemo(
    () => (status === "running" || content === "" ? null : parseToolResult(content)),
    [status, content],
  );

  // A failure is the one outcome worth surfacing unasked, live or not.
  const expanded = open || (live && status === "failed");

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setOpen}
      className="border-border bg-card overflow-hidden rounded-md border"
    >
      <CollapsibleTrigger
        className="hover:bg-muted/50 flex w-full items-center gap-2.5 px-2.5 py-1.5 text-left text-xs transition-colors"
        aria-label={t("Show details for {0}", description.title)}
      >
        <StatusIcon status={status} />
        <span className="min-w-0 flex-1 truncate">
          <span className="font-medium">{description.title}</span>
          {description.subject !== "" && (
            <span className="text-muted-foreground"> · {description.subject}</span>
          )}
        </span>
        <StatusBadge status={status} />
        <ChevronDownIcon
          className={cn(
            "text-muted-foreground size-3.5 shrink-0 transition-transform",
            expanded && "rotate-180",
          )}
        />
      </CollapsibleTrigger>

      <CollapsibleContent className="border-border border-t">
        <div className="flex flex-col gap-3 px-3 py-2.5 text-xs">
          {rows.length > 0 && (
            <section className="flex flex-col gap-1">
              <h4 className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
                {t("Asked for")}
              </h4>
              <dl className="grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] gap-x-3 gap-y-0.5">
                {rows.map((row) => (
                  <div key={row.key} className="contents">
                    <dt className="text-muted-foreground font-mono">{row.key}</dt>
                    <dd className="break-all">{row.value}</dd>
                  </div>
                ))}
              </dl>
            </section>
          )}

          <section className="flex flex-col gap-1">
            <h4 className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
              {status === "proposed" ? t("Outcome") : t("Returned")}
            </h4>
            <ToolResultBody status={status} result={result} />
          </section>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function ToolResultBody({
  status,
  result,
}: {
  status: ToolActivityStatus;
  result: ReturnType<typeof parseToolResult> | null;
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
      return (
        <div className="bg-muted/40 max-h-72 overflow-auto rounded-md p-2">
          <JsonViewer data={result.value as never} collapsed={2} />
        </div>
      );
    default:
      return (
        <div className="flex flex-col gap-1">
          <pre className="bg-muted/40 max-h-72 overflow-auto rounded-md p-2 font-mono text-[11px] whitespace-pre-wrap">
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

function StatusIcon({ status }: { status: ToolActivityStatus }) {
  switch (status) {
    case "running":
      return <Spinner className="size-3.5 shrink-0" />;
    case "failed":
      return <CircleAlertIcon className="text-destructive size-3.5 shrink-0" />;
    case "proposed":
      return <ShieldQuestionIcon className="text-warning size-3.5 shrink-0" />;
    default:
      return <SearchIcon className="text-muted-foreground size-3.5 shrink-0" />;
  }
}

function StatusBadge({ status }: { status: ToolActivityStatus }) {
  const t = useT();

  switch (status) {
    case "running":
      return (
        <Badge variant="info" className="shrink-0">
          {t("Running")}
        </Badge>
      );
    case "failed":
      return (
        <Badge variant="inactive" className="shrink-0">
          {t("Failed")}
        </Badge>
      );
    case "proposed":
      return (
        <Badge variant="warning" className="shrink-0">
          {t("Needs approval")}
        </Badge>
      );
    default:
      return (
        <Badge variant="active" className="shrink-0 gap-1">
          <CheckIcon className="size-3" />
          {t("Done")}
        </Badge>
      );
  }
}
