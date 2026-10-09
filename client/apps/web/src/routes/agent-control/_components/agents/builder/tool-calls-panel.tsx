import { describeToolCall } from "@/components/assistant/tool-presentation";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { AgentToolVerdictRow } from "@/lib/graphql/agent-scorecard";
import { AlertCircleIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useMemo, useState } from "react";
import {
  formatFailingShare,
  toolCallRecords,
  toolCallTotals,
  verdictLabel,
  verdictTone,
  wentThrough,
  type ToolCallRecord,
  type VerdictTone,
} from "./tool-calls-model";

/** Tools listed before "Show all"; the worst come first, so these are the ones to fix. */
const VISIBLE_TOOLS = 8;
const SKELETON_ROWS = 3;

const TONE_FILL: Record<VerdictTone, string> = {
  success: "bg-success",
  info: "bg-info",
  neutral: "bg-neutral",
  warning: "bg-warning",
  danger: "bg-danger",
};

type ToolCallsPanelProps = {
  /** The scorecard's verdict rows; undefined until it has loaded. */
  verdicts: readonly AgentToolVerdictRow[] | undefined;
  loading: boolean;
  failed: boolean;
};

/**
 * How each tool's calls ended over the scorecard's window, worst first.
 *
 * The approval figures count what people did with proposals; a call the
 * runtime refused — not permitted, arguments not accepted, a repeat, out of
 * budget — never became one, so a tool the model keeps calling wrongly looked
 * healthy there. Here every call that reached a tool is counted, each tool
 * opens onto the reasons its calls gave most often.
 */
export function ToolCallsPanel({ verdicts, loading, failed }: ToolCallsPanelProps) {
  const t = useT();
  const idPrefix = useId();
  const records = useMemo(() => toolCallRecords(verdicts ?? []), [verdicts]);
  const totals = useMemo(() => toolCallTotals(records), [records]);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set());
  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? records : records.slice(0, VISIBLE_TOOLS);

  const toggle = (toolName: string) =>
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(toolName)) {
        next.delete(toolName);
      } else {
        next.add(toolName);
      }
      return next;
    });

  const hint =
    records.length === 0
      ? undefined
      : totals.failing > 0
        ? t(
            "{0} of {1, plural, one {# call} other {# calls}} didn't go through",
            totals.failing.toLocaleString(),
            totals.calls,
          )
        : t("Every call went through");

  return (
    <SectionPanel
      className="mt-4"
      title={t("Tool calls")}
      count={records.length}
      hint={hint}
      help={t(
        "Every tool call it made in the last 30 days, including the ones turned away before a tool ran. A call the runtime turned away never became a proposal, so the figures above cannot show it. The tools with the most calls that didn't go through come first; open one to read the reasons its calls gave most often.",
      )}
    >
      {loading ? (
        <div className="flex flex-col gap-2 px-3 py-3" aria-busy="true">
          {Array.from({ length: SKELETON_ROWS }, (_, index) => (
            <Skeleton key={index} className="h-9 w-full" />
          ))}
        </div>
      ) : failed ? (
        <div className="p-3">
          <Alert variant="destructive" size="sm">
            <AlertCircleIcon />
            <AlertDescription>{t("The tool calls could not be loaded.")}</AlertDescription>
          </Alert>
        </div>
      ) : records.length === 0 ? (
        <SectionPanelQuiet>{t("No tool calls in the last 30 days.")}</SectionPanelQuiet>
      ) : (
        <>
          <div className="bg-sunken border-border-subtle text-foreground-subtle flex h-(--row-head-h) items-center justify-between gap-3 border-b px-(--cell-px) text-xs font-medium">
            <span className="pl-5.5">{t("Tool")}</span>
            <span>{t("Didn't go through")}</span>
          </div>
          <ul className="divide-border-subtle divide-y">
            {shown.map((record, index) => (
              <ToolCallRow
                key={record.toolName}
                record={record}
                open={expanded.has(record.toolName)}
                regionId={`${idPrefix}-${index}`}
                onToggle={() => toggle(record.toolName)}
              />
            ))}
          </ul>
          {records.length > VISIBLE_TOOLS && (
            <div className="border-border-subtle border-t px-1.5 py-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-expanded={showAll}
                onClick={() => setShowAll((all) => !all)}
              >
                {showAll ? t("Show fewer") : t("Show all {0} tools", records.length)}
              </Button>
            </div>
          )}
        </>
      )}
    </SectionPanel>
  );
}

function ToolCallRow({
  record,
  open,
  regionId,
  onToggle,
}: {
  record: ToolCallRecord;
  open: boolean;
  regionId: string;
  onToggle: () => void;
}) {
  const t = useT();
  const title = describeToolCall(record.toolName, null).title;

  return (
    <li>
      <button
        type="button"
        className="hover:bg-surface-hover ui-inset-focus-ring grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-x-2 px-(--cell-px) py-2 text-left transition-colors"
        aria-expanded={open}
        aria-controls={regionId}
        onClick={onToggle}
      >
        <ChevronRightIcon
          className={cn(
            "text-foreground-subtle mt-0.5 size-3.5 transition-transform duration-200",
            open && "rotate-90",
          )}
          aria-hidden
        />
        <span className="flex min-w-0 flex-col gap-1.5">
          <span className="truncate text-sm">{title}</span>
          <VerdictBar record={record} />
          <span className="text-foreground-muted flex flex-wrap gap-x-3 gap-y-0.5 text-xs">
            {record.verdicts.map((tally) => (
              <span key={tally.verdict} className="inline-flex items-center gap-1">
                <span
                  className={cn("size-1.5 rounded-full", TONE_FILL[verdictTone(tally.verdict)])}
                  aria-hidden
                />
                {verdictLabel(tally.verdict, t)}
                <span className="tabular-nums">{tally.calls.toLocaleString()}</span>
              </span>
            ))}
          </span>
        </span>
        <span className="flex flex-col items-end">
          <span
            className={cn(
              "text-sm tabular-nums",
              record.failing > 0 ? "text-danger" : "text-foreground-subtle",
            )}
          >
            {formatFailingShare(record.failingShare)}
          </span>
          <span className="text-foreground-subtle text-xs tabular-nums">
            {t(
              "{0} of {1, plural, one {# call} other {# calls}}",
              record.failing.toLocaleString(),
              record.calls,
            )}
          </span>
        </span>
      </button>
      {open && (
        <div id={regionId} className="bg-sunken border-border-subtle border-t py-2 pr-(--cell-px) pl-8">
          <VerdictReasons record={record} />
        </div>
      )}
    </li>
  );
}

/** Each verdict's share of the tool's calls, in one line. */
function VerdictBar({ record }: { record: ToolCallRecord }) {
  const t = useT();
  const label = record.verdicts
    .map((tally) =>
      t("{0}: {1, plural, one {# call} other {# calls}}", verdictLabel(tally.verdict, t), tally.calls),
    )
    .join(", ");

  return (
    <span
      role="img"
      aria-label={label}
      className="bg-sunken flex h-1.5 w-full max-w-60 gap-px overflow-hidden rounded-full"
    >
      {record.verdicts.map((tally) => (
        <span
          key={tally.verdict}
          className={cn("h-full", TONE_FILL[verdictTone(tally.verdict)])}
          style={{ width: `${(tally.calls / record.calls) * 100}%` }}
        />
      ))}
    </span>
  );
}

/** The calls that did not go through, by how they ended, each with its commonest reasons. */
function VerdictReasons({ record }: { record: ToolCallRecord }) {
  const t = useT();
  const failing = record.verdicts.filter((tally) => !wentThrough(tally.verdict));

  if (failing.length === 0) {
    return <p className="text-foreground-muted text-xs">{t("Every call went through.")}</p>;
  }

  return (
    <div className="flex flex-col gap-2.5">
      {failing.map((tally) => {
        const label = verdictLabel(tally.verdict, t);
        return (
          <section key={tally.verdict} aria-label={label}>
            <h4 className="flex items-center gap-1.5 text-xs font-medium">
              <span
                className={cn("size-1.5 rounded-full", TONE_FILL[verdictTone(tally.verdict)])}
                aria-hidden
              />
              {label}
              <span className="text-foreground-subtle font-normal tabular-nums">
                {tally.calls.toLocaleString()}
              </span>
            </h4>
            {tally.reasons.length === 0 ? (
              <p className="text-foreground-subtle mt-1 text-xs">{t("No reason was recorded.")}</p>
            ) : (
              <ul className="mt-1 flex flex-col gap-0.5">
                {tally.reasons.map((reason) => (
                  <li
                    key={reason.reason}
                    className="flex items-baseline justify-between gap-3 text-xs"
                  >
                    <span className="text-foreground-muted min-w-0 break-words">
                      {reason.reason}
                    </span>
                    <span className="text-foreground-subtle shrink-0 tabular-nums">
                      {t("{0, plural, one {# call} other {# calls}}", reason.calls)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        );
      })}
    </div>
  );
}
