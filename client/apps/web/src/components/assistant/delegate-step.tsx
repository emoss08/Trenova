import { useDelegateIdentity } from "@/components/agent-identity/agent-context";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DelegateDocument } from "@/types/assistant";
import {
  CheckIcon,
  ChevronRightIcon,
  CircleAlertIcon,
  FlaskConicalIcon,
  HourglassIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { currentActivity, type ToolStep } from "./activity";
import { useArtifactOpener } from "./artifact-opener";
import { useWatchedChange } from "./decision-chrome";
import {
  awaitingLine,
  delegateView,
  handOffHasDetail,
  handOffHeadline,
  handOffStatus,
  handOffWorking,
  madeLine,
  publishedKind,
  splitOnSubject,
  type DelegateView,
  type HandOffTone,
  type WriteLine,
} from "./delegation";
import { ToolActivity } from "./tool-activity";
import { ArtifactKindIcon } from "./voice/artifact-chrome";
import { WorkingDot } from "./voice/working-dot";

const TONE_TEXT: Record<HandOffTone, string> = {
  muted: "text-foreground-muted",
  warning: "text-warning",
  danger: "text-danger",
};

/**
 * A task the conversation's agent handed to another agent, drawn as one step
 * of the turn: "Asked Report Builder", the other agent's mark beside it and
 * the task on the same line.
 *
 * While the other agent works, its own steps land beneath the hand-off and a
 * quiet line says what it is doing now; the breathing dot stays with the turn,
 * at the foot of the reply. When it finishes, the hand-off settles into what
 * came of it — how it ended, what it made, what waits on the person and what
 * it published — and opens onto the whole task, every step it took and its
 * answer. The agent is told apart by its mark and nothing else: no colour of
 * its own, no bar.
 */
export function DelegateStep({
  step,
  live,
  running,
}: {
  step: ToolStep;
  /** Part of a reply still being written: arrivals are animated. */
  live: boolean;
  /** The other agent is still working on the task. */
  running: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const view = useMemo(() => delegateView(step), [step]);
  const fallback = useMemo(
    () => ({
      id: view?.agentId ?? "",
      name: view?.agentName ?? "",
      icon: view?.icon ?? "",
      accent: view?.accent ?? "",
    }),
    [view?.agentId, view?.agentName, view?.icon, view?.accent],
  );
  const identity = useDelegateIdentity(fallback);
  const headline = view ? handOffHeadline(view, t) : "";
  const changed = useWatchedChange(headline);

  if (!view) {
    return null;
  }

  const expandable = handOffHasDetail(view);
  const settled = !running;

  return (
    <li className={cn("min-w-0", live && "animate-land")}>
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger
          disabled={!expandable}
          className={cn(
            "group/handoff ui-focus-ring -mx-1.5 flex w-[calc(100%+0.75rem)] min-w-0 items-center gap-2 rounded-control px-1.5 py-1 text-left text-xs transition-colors",
            "hover:bg-surface-hover disabled:cursor-default disabled:hover:bg-transparent",
          )}
        >
          <AgentTile agent={identity} size="xs" />
          <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
            <span
              key={headline}
              className={cn(
                "min-w-0 shrink-0 truncate",
                view.outcome === "declined" ? "text-danger" : "text-foreground",
                live && changed && "animate-rise",
              )}
            >
              {headline}
            </span>
            {view.task !== "" && (
              <span className="text-foreground-subtle min-w-0 flex-1 truncate">{view.task}</span>
            )}
          </span>
          {settled && step.durationSeconds !== null && (
            <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
              {formatWorkDuration(step.durationSeconds)}
            </span>
          )}
          {expandable && (
            <ChevronRightIcon
              aria-hidden
              className={cn(
                "text-foreground-subtle size-3 shrink-0 opacity-0 transition-[opacity,rotate] group-hover/handoff:opacity-100 group-focus-visible/handoff:opacity-100",
                open && "rotate-90 opacity-100",
              )}
            />
          )}
        </CollapsibleTrigger>

        <div className="flex min-w-0 flex-col gap-1 pl-7">
          {running ? (
            <HandOffProgress view={view} />
          ) : (
            <HandOffSummary view={view} arrived={live && changed} />
          )}
        </div>

        {expandable && (
          <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
            <HandOffDetails view={view} running={running} />
          </CollapsibleContent>
        )}
      </Collapsible>
    </li>
  );
}

/**
 * The other agent's work while it goes: the steps it has finished, each
 * landing with its check, and one line for what it is doing now. The dot
 * beside that line holds still, because the turn's own working line is where
 * the product breathes.
 */
function HandOffProgress({ view }: { view: DelegateView }) {
  const t = useT();
  const underway = view.steps.some((step) => step.status === "running");
  const label = underway
    ? (currentActivity(view.steps, t)?.phrase ?? handOffWorking(view, t))
    : handOffWorking(view, t);

  return (
    <>
      <ToolActivity steps={view.steps} live />
      <div
        role="status"
        aria-live="polite"
        className="text-foreground-muted flex h-6 min-w-0 items-center gap-2 text-xs"
      >
        <WorkingDot working still className="mx-0.75" />
        <span key={label} className="animate-rise min-w-0 truncate">
          {label}
        </span>
      </div>
    </>
  );
}

/**
 * What came of the task, once it is over: how it ended, then what it made,
 * what it left for the person to approve and what it published. The lines
 * rise into place when the hand-off settles while someone watches.
 */
function HandOffSummary({ view, arrived }: { view: DelegateView; arrived: boolean }) {
  const t = useT();
  const status = handOffStatus(view, t);
  const report = view.report;
  const made = report?.made ?? [];
  const awaiting = report?.awaiting ?? [];
  const published = report?.published ?? [];

  const more = (report?.moreMade ?? 0) + (report?.moreAwaiting ?? 0) + (report?.morePublished ?? 0);
  if (
    !status &&
    made.length === 0 &&
    awaiting.length === 0 &&
    published.length === 0 &&
    more === 0
  ) {
    return null;
  }

  return (
    <ul className={cn("flex min-w-0 flex-col gap-0.5 pb-1 text-xs", arrived && "animate-rise")}>
      {status && (
        <li className={cn("flex min-w-0 items-baseline gap-1.5", TONE_TEXT[status.tone])}>
          <span className="min-w-0 break-words">{status.text}</span>
          {report && report.toolCallsUsed > 0 && (
            <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
              {t("{0, plural, one {# step} other {# steps}}", report.toolCallsUsed)}
            </span>
          )}
        </li>
      )}
      {made.map((write, index) => (
        <WriteRow key={write.callId || `made-${index}`} line={madeLine(write, t)} />
      ))}
      {report && report.moreMade > 0 && (
        <MoreRow
          text={t("{0, plural, one {# more change} other {# more changes}}", report.moreMade)}
        />
      )}
      {(awaiting.length > 0 || (report?.moreAwaiting ?? 0) > 0) && (
        <li className="text-foreground-subtle pt-0.5">{t("Waiting for your approval")}</li>
      )}
      {awaiting.map((write, index) => (
        <WriteRow key={write.callId || `awaiting-${index}`} line={awaitingLine(write, t)} />
      ))}
      {report && report.moreAwaiting > 0 && (
        <MoreRow
          text={t(
            "{0, plural, one {# more proposal} other {# more proposals}}",
            report.moreAwaiting,
          )}
        />
      )}
      {published.map((document) => (
        <PublishedRow key={document.id} document={document} />
      ))}
      {report && report.morePublished > 0 && (
        <MoreRow
          text={t(
            "{0, plural, one {# more document} other {# more documents}}",
            report.morePublished,
          )}
        />
      )}
    </ul>
  );
}

/**
 * What a saved account left out of a list, counted: the thread keeps only the
 * first few writes of a long task, and says so rather than implying it made
 * no more.
 */
function MoreRow({ text }: { text: string }) {
  return <li className="text-foreground-subtle pl-4.5">{text}</li>;
}

function WriteMark({ state }: { state: WriteLine["state"] }) {
  switch (state) {
    case "failed":
      return <CircleAlertIcon aria-hidden className="text-danger mt-0.5 size-3 shrink-0" />;
    case "simulated":
      return (
        <FlaskConicalIcon aria-hidden className="text-foreground-muted mt-0.5 size-3 shrink-0" />
      );
    case "awaiting":
      return <HourglassIcon aria-hidden className="text-warning mt-0.5 size-3 shrink-0" />;
    default:
      return <CheckIcon aria-hidden className="text-foreground-muted mt-0.5 size-3 shrink-0" />;
  }
}

/** One write, in the words of the write, with the record's name linking to it where it has a page. */
function WriteRow({ line }: { line: WriteLine }) {
  const parts = splitOnSubject(line);

  return (
    <li className="flex min-w-0 items-start gap-1.5">
      <WriteMark state={line.state} />
      <span className="min-w-0 flex-1 break-words">
        <span className={line.state === "failed" ? "text-danger" : "text-foreground"}>
          {parts && line.path ? (
            <>
              {parts.before}
              <Link
                to={line.path}
                className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
              >
                {parts.subject}
              </Link>
              {parts.after}
            </>
          ) : (
            line.text
          )}
        </span>
        {line.error !== "" && <span className="text-foreground-subtle"> · {line.error}</span>}
      </span>
    </li>
  );
}

/** A document the other agent published, opened in the pane where the surface has one. */
function PublishedRow({ document }: { document: DelegateDocument }) {
  const t = useT();
  const open = useArtifactOpener();
  const kind = publishedKind(document);
  const title = document.title !== "" ? document.title : t("Untitled");
  const mark = kind ? (
    <ArtifactKindIcon kind={kind} className="text-foreground-muted mt-0.5 size-3 shrink-0" />
  ) : (
    <CheckIcon aria-hidden className="text-foreground-muted mt-0.5 size-3 shrink-0" />
  );

  return (
    <li className="flex min-w-0 items-start gap-1.5">
      {mark}
      <span className="text-foreground min-w-0 flex-1 break-words">
        {open && kind ? (
          <>
            {t("Published")}{" "}
            <button
              type="button"
              onClick={() => open(document.id)}
              aria-label={t("Open {0}", title)}
              className="ui-focus-ring text-brand rounded-control text-left underline-offset-2 hover:underline"
            >
              {title}
            </button>
          </>
        ) : (
          t("Published {0}", title)
        )}
      </span>
    </li>
  );
}

/**
 * The whole hand-off, behind the step: the task as the other agent was given
 * it, every step it took, and its answer. Its steps open onto their own
 * details the same way the turn's do.
 */
function HandOffDetails({ view, running }: { view: DelegateView; running: boolean }) {
  const t = useT();

  return (
    <div className="flex min-w-0 flex-col gap-2.5 pt-1 pb-2 pl-7 text-xs">
      {view.task !== "" && (
        <section className="flex min-w-0 flex-col gap-1">
          <h4 className="text-foreground-subtle font-medium">{t("Task")}</h4>
          <p className="text-foreground-muted leading-relaxed break-words whitespace-pre-wrap">
            {view.task}
          </p>
        </section>
      )}
      {!running && view.steps.length > 0 && (
        <section className="flex min-w-0 flex-col gap-1">
          <h4 className="text-foreground-subtle font-medium">{t("What it did")}</h4>
          <ToolActivity steps={view.steps} />
        </section>
      )}
      {view.reply.trim() !== "" && (
        <section className="flex min-w-0 flex-col gap-1">
          <h4 className="text-foreground-subtle font-medium">{t("Its answer")}</h4>
          <AiMarkdown
            content={view.reply}
            className="text-foreground-muted text-xs leading-relaxed"
          />
        </section>
      )}
    </div>
  );
}
