import { formatLatency, formatUsd, formatWorkDuration } from "@/lib/ai-usage-format";
import { cn } from "@trenova/shared/lib/utils";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixDateTimeMedium, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { useAssistantAgent } from "@/components/agent-identity/agent-context";
import { AgentTile, type AgentTileSize } from "@/components/agent-identity/agent-tile";
import type {
  AssistantArtifact,
  AssistantEntityRef,
  AssistantMessage,
  AssistantMessageAttachment,
  AssistantPageContext,
  AssistantProposal,
} from "@/types/assistant";
import { ArtifactKindIcon, ARTIFACT_KINDS } from "./voice/artifact-chrome";
import { FeedbackControl } from "@/components/ai-feedback/feedback-control";
import { TURN_FEEDBACK_REVEAL } from "@/components/ai-feedback/feedback-targets";
import {
  CheckCheckIcon,
  CheckIcon,
  ChevronRightIcon,
  CopyIcon,
  AtSignIcon,
  FileIcon,
  MapPinIcon,
  RotateCcwIcon,
  ShieldAlertIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { stepsFromExchanges } from "./activity";
import { askRequestsFrom } from "./ask-requests";
import { ChoicePrompt } from "./choice-prompt";
import { PlanCard } from "./plan-card";
import type { PlanGroup } from "./plan-state";
import { ProposalCard } from "./proposal-card";
import { ReportRunCard } from "./report-run-card";
import { reportRunsFrom } from "./report-runs";
import { decisionHeadline, type ThreadEntry, type TurnPlacement } from "./thread-view";
import { ToolActivity } from "./tool-activity";

const TIME_FORMAT = { hour: "numeric", minute: "2-digit" } as const;

/** The agent's face in the thread: one mark, used everywhere it speaks. */
export function AgentAvatar({
  className,
  size = "md",
}: {
  className?: string;
  size?: AgentTileSize;
}) {
  const agent = useAssistantAgent();

  return <AgentTile agent={agent} size={size} className={className} />;
}

/** Where the question was asked from, so an answer can be read against its page. */
export function PageContextChip({ context }: { context: AssistantPageContext | null | undefined }) {
  if (!context || (context.title === "" && context.entityType === "")) {
    return null;
  }

  const record = context.entityType
    ? `${context.entityType.replace(/_/g, " ")}${context.entityId ? ` ${context.entityId}` : ""}`
    : "";

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="text-muted-foreground inline-flex max-w-full min-w-0 items-center gap-1 text-xs">
            <MapPinIcon className="size-3 shrink-0" />
            <span className="truncate">{context.title || record || context.path}</span>
          </span>
        }
      />
      <TooltipContent className="max-w-xs font-mono text-xs">
        {context.path}
        {record ? ` · ${record}` : ""}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * What a person handed over with their words: the files, and the records
 * they named. Drawn as chips under the message so an answer can be read
 * against what it was given.
 */
export function TurnContextChips({
  attachments,
  mentions,
}: {
  attachments?: readonly AssistantMessageAttachment[] | null;
  mentions?: readonly AssistantEntityRef[] | null;
}) {
  const files = attachments ?? [];
  const records = mentions ?? [];
  if (files.length === 0 && records.length === 0) {
    return null;
  }

  return (
    <ul className="flex flex-wrap gap-1.5">
      {files.map((file) => (
        <li
          key={file.documentId}
          className="border-border bg-surface inline-flex h-6 max-w-[16rem] items-center gap-1.5 rounded-full border px-2 text-xs"
        >
          <FileIcon className="text-muted-foreground size-3 shrink-0" />
          <span className="min-w-0 truncate">{file.fileName}</span>
        </li>
      ))}
      {records.map((record) => (
        <li
          key={`${record.type}:${record.id}`}
          className="border-border bg-surface inline-flex h-6 max-w-[16rem] items-center gap-1.5 rounded-full border px-2 text-xs"
          title={`${record.type.replaceAll("_", " ")} ${record.id}`}
        >
          <AtSignIcon className="text-muted-foreground size-3 shrink-0" />
          <span className="min-w-0 truncate">{record.label || record.id}</span>
        </li>
      ))}
    </ul>
  );
}

/** When a turn was said, and the full date behind a hover. */
function TurnTime({ at }: { at: number }) {
  return (
    <time
      dateTime={new Date(at * 1000).toISOString()}
      title={formatUnixDateTimeMedium(at)}
      className="text-foreground-subtle shrink-0 tabular-nums"
    >
      {formatUnixInUserTimezone(at, TIME_FORMAT)}
    </time>
  );
}

/** The actions a turn offers, shown when the pointer or focus is on it. */
function TurnActions({ children }: { children: ReactNode }) {
  return (
    <span className="ml-auto flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover/turn:opacity-100 has-[:focus-visible]:opacity-100">
      {children}
    </span>
  );
}

/**
 * A person's message, or the one they are about to send.
 *
 * It is held in a quiet well and labelled "You", and the reply below it runs
 * open across the column. The difference is one of voice, not of side: both
 * share the left edge, so the eye never crosses the panel, and the question
 * reads as the thing given while the answer reads as the work done with it.
 */
export function UserTurn({
  content,
  sentAt,
  pageContext,
  attachments,
  mentions,
  onResend,
}: {
  content: string;
  sentAt?: number;
  pageContext?: AssistantPageContext | null;
  attachments?: readonly AssistantMessageAttachment[] | null;
  mentions?: readonly AssistantEntityRef[] | null;
  /** Asks the same thing again, for a reply that went wrong or went stale. */
  onResend?: () => void;
}) {
  const t = useT();

  return (
    <article className="group/turn flex min-w-0 flex-col gap-1.5">
      <header className="flex h-5 min-w-0 items-center gap-2 text-xs">
        <span className="text-foreground shrink-0 font-medium">{t("You")}</span>
        {sentAt !== undefined && sentAt > 0 && <TurnTime at={sentAt} />}
        <PageContextChip context={pageContext} />
        <TurnActions>
          <IconAction label={t("Copy")} done={t("Copied")} onClick={() => copyText(content)}>
            <CopyIcon className="size-3" />
          </IconAction>
          {onResend && (
            <IconAction label={t("Ask again")} onClick={onResend}>
              <RotateCcwIcon className="size-3" />
            </IconAction>
          )}
        </TurnActions>
      </header>
      <div className="bg-sunken w-fit max-w-full rounded-lg px-3 py-2">
        <p className="text-sm leading-relaxed break-words whitespace-pre-wrap">{content}</p>
      </div>
      <TurnContextChips attachments={attachments} mentions={mentions} />
    </article>
  );
}

/**
 * The frame around anything the assistant says, live or saved.
 *
 * The agent's mark and name head the reply once; a reply that took several
 * steps continues under that one header rather than repeating it. The header
 * says when the reply was made and how long the work took; while it is still
 * going, the working line at the foot of the reply says what it is doing.
 */
export function AssistantTurn({
  at,
  meta,
  actions,
  feedback,
  continued = false,
  workedSeconds = null,
  working = false,
  children,
}: {
  at?: number;
  meta?: ReactNode;
  actions?: ReactNode;
  /** The rating control; it manages its own fade so a chosen thumb stays in view. */
  feedback?: ReactNode;
  /** A later step of a reply already headed above: no header of its own. */
  continued?: boolean;
  /** How long the reply took, from the question to its last step. */
  workedSeconds?: number | null;
  /** The reply is still being written. */
  working?: boolean;
  children: ReactNode;
}) {
  const t = useT();
  const agent = useAssistantAgent();

  if (continued) {
    return (
      <article className="group/turn relative -mt-2 flex min-w-0 flex-col gap-2.5">
        {(actions || feedback) && (
          <span className="absolute top-0 right-0 z-1 flex items-center gap-0.5">
            {actions && <TurnActions>{actions}</TurnActions>}
            {feedback}
          </span>
        )}
        {children}
      </article>
    );
  }

  return (
    <article className="group/turn flex min-w-0 flex-col gap-2.5">
      <header className="flex h-7 min-w-0 items-center gap-2 text-xs">
        <AgentAvatar />
        <span className="text-foreground min-w-0 truncate text-sm font-medium">
          {agent?.name ?? t("Assistant")}
        </span>
        {!working && (
          <>
            {at !== undefined && at > 0 && <TurnTime at={at} />}
            {workedSeconds !== null && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="text-foreground-subtle shrink-0 cursor-default font-mono text-2xs tabular-nums" />
                  }
                >
                  {formatWorkDuration(workedSeconds)}
                </TooltipTrigger>
                <TooltipContent>
                  {t("Worked for {0}", formatWorkDuration(workedSeconds))}
                </TooltipContent>
              </Tooltip>
            )}
          </>
        )}
        {meta}
        {(actions || feedback) && (
          <span className="ml-auto flex shrink-0 items-center gap-0.5">
            {actions && <TurnActions>{actions}</TurnActions>}
            {feedback}
          </span>
        )}
      </header>
      {children}
    </article>
  );
}

/**
 * Prose from the assistant. No bubble: an answer that runs the width of the
 * panel reads as a tool doing work, where a coloured bubble reads as a chat
 * partner, and this one is looking at the organization's freight.
 */
export function AssistantProse({
  content,
  streaming = false,
}: {
  content: string;
  streaming?: boolean;
}) {
  return (
    <div className="min-w-0 text-sm leading-relaxed">
      <AiMarkdown content={content} />
      {streaming && (
        <span
          aria-hidden
          className="assistant-caret bg-foreground ml-0.5 inline-block h-[1em] w-[2px] rounded-full align-text-bottom"
        />
      )}
    </div>
  );
}

async function copyText(text: string) {
  await navigator.clipboard.writeText(text);
}

/**
 * A small action in a turn's header. A copy lands with a tick that stays a
 * moment, which is the only confirmation a copy needs.
 */
function IconAction({
  label,
  done,
  onClick,
  children,
}: {
  label: string;
  /** The label while the action's result stands, when it has one. */
  done?: string;
  onClick: () => void | Promise<void>;
  children: ReactNode;
}) {
  const [confirmed, setConfirmed] = useState(false);

  const act = useCallback(async () => {
    await onClick();
    if (done) {
      setConfirmed(true);
      window.setTimeout(() => setConfirmed(false), 1500);
    }
  }, [done, onClick]);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:text-foreground"
            onClick={() => void act()}
            aria-label={label}
          />
        }
      >
        {confirmed ? <CheckIcon className="animate-confirm size-3" /> : children}
      </TooltipTrigger>
      <TooltipContent>{confirmed && done ? done : label}</TooltipContent>
    </Tooltip>
  );
}

/**
 * What the model thought, shown apart from what it said.
 *
 * Open while the thinking is still arriving, because that is the minute a
 * heavy model would otherwise spend looking hung. Folded away once the answer
 * begins: the reasoning is there for whoever wants to check the answer
 * against it, not in the way of reading the answer. The fold is animated so
 * the answer rising into its place reads as the thought giving way to it.
 */
export function ReasoningDisclosure({
  text,
  streaming = false,
}: {
  text: string;
  streaming?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(streaming);
  const wasStreaming = useRef(streaming);

  useEffect(() => {
    if (wasStreaming.current && !streaming) {
      setOpen(false);
    }
    wasStreaming.current = streaming;
  }, [streaming]);

  if (text === "" && !streaming) {
    return null;
  }

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="min-w-0">
      <CollapsibleTrigger className="group/reasoning text-foreground-muted hover:text-foreground ui-focus-ring -mx-1.5 flex items-center gap-1.5 rounded-control px-1.5 py-0.5 text-xs transition-colors">
        <ChevronRightIcon
          className={cn("size-3 transition-transform duration-200", open && "rotate-90")}
          aria-hidden
        />
        <span key={streaming ? "thinking" : "thought"} className={cn(!streaming && "animate-rise")}>
          {streaming ? t("Thinking it through") : t("Thought it through")}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
        <div className="text-foreground-muted pt-1.5 pb-1 pl-4.5 text-xs leading-relaxed whitespace-pre-wrap">
          {text}
          {streaming && (
            <span
              aria-hidden
              className="assistant-caret bg-foreground-subtle ml-0.5 inline-block h-[1em] w-[2px] rounded-full align-text-bottom"
            />
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/** A saved assistant turn: what it said, what it did, what it asked to do. */
export function AssistantEntry({
  entry,
  placement,
  proposals,
  plans = [],
  artifacts = [],
  threadId,
  latestUserSequence,
  onAnswer,
  onOpenArtifact,
  ratable = false,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
  /** Where this step sits in its reply; the first step carries the header. */
  placement?: TurnPlacement;
  proposals: AssistantProposal[];
  /** Several writes this turn asked for as one; each is shown once, as a plan. */
  plans?: PlanGroup[];
  /** What this turn produced besides words, when the surface has a pane to open it in. */
  artifacts?: AssistantArtifact[];
  /** Where the newest user turn sits, so a settled question stops asking. */
  latestUserSequence: number;
  onAnswer: (value: string) => void;
  onOpenArtifact?: (id: string) => void;
  threadId: string;
  /** This step is the answer of its reply, and the person may rate it. */
  ratable?: boolean;
}) {
  const t = useT();
  const { message, tools } = entry;

  // A report run outlives the turn that started it, so the thread follows it
  // rather than leaving the reader to ask again for the outcome.
  const reportRuns = reportRunsFrom(tools);
  const asks = askRequestsFrom(tools);
  const steps = stepsFromExchanges(tools, message.createdAt);

  return (
    <AssistantTurn
      at={message.createdAt}
      continued={placement?.continued ?? false}
      workedSeconds={placement?.workedSeconds ?? null}
      meta={message.model !== "" ? <ModelNote message={message} /> : null}
      actions={
        message.content !== "" ? (
          <IconAction
            label={t("Copy reply")}
            done={t("Copied")}
            onClick={() => copyText(message.content)}
          >
            <CopyIcon className="size-3" />
          </IconAction>
        ) : null
      }
      feedback={
        ratable ? (
          <FeedbackControl
            target={{ targetType: "AssistantMessage", targetId: message.id }}
            revealClassName={TURN_FEEDBACK_REVEAL}
          />
        ) : null
      }
    >
      {message.reasoning?.text ? <ReasoningDisclosure text={message.reasoning.text} /> : null}
      {steps.length > 0 && <ToolActivity steps={steps} />}
      {artifacts.length > 0 && onOpenArtifact && (
        <ArtifactChips artifacts={artifacts} onOpen={onOpenArtifact} />
      )}
      {message.content !== "" && <AssistantProse content={message.content} />}
      {reportRuns.map((run) => (
        <ReportRunCard key={run.runId} run={run} />
      ))}
      {asks.map((ask) => (
        <ChoicePrompt
          key={ask.callId}
          request={ask}
          answered={latestUserSequence > ask.sequence}
          onAnswer={onAnswer}
        />
      ))}
      {plans.map((group) => (
        <PlanCard key={group.plan.id} plan={group.plan} steps={group.steps} threadId={threadId} />
      ))}
      {proposals.map((proposal) => (
        <ProposalCard key={proposal.id} proposal={proposal} threadId={threadId} />
      ))}
    </AssistantTurn>
  );
}

/**
 * What a turn produced, as references the pane opens. The transcript points
 * at a table or a draft; it does not carry a second copy of it. Only the
 * kinds the pane can render are offered.
 */
export function ArtifactChips({
  artifacts,
  onOpen,
}: {
  artifacts: AssistantArtifact[];
  onOpen: (id: string) => void;
}) {
  const t = useT();

  return (
    <ul className="flex flex-wrap gap-1.5">
      {artifacts.map((artifact) => (
        <li key={artifact.id}>
          <button
            type="button"
            onClick={() => onOpen(artifact.id)}
            className="ui-focus-ring border-border hover:bg-surface-hover flex h-7 max-w-64 items-center gap-1.5 rounded-full border px-2.5 text-xs transition-colors"
            aria-label={t("Open {0}", artifact.title)}
          >
            <ArtifactKindIcon
              kind={artifact.kind}
              className="text-muted-foreground size-3 shrink-0"
            />
            <span className="text-muted-foreground shrink-0">
              {t(ARTIFACT_KINDS[artifact.kind].label)}
            </span>
            <span className="truncate">{artifact.title}</span>
          </button>
        </li>
      ))}
    </ul>
  );
}

/**
 * Which model answered and what it cost, behind a hover. That belongs to
 * whoever is tuning the agent, not to the dispatcher reading the answer, so
 * it is a mark in the header rather than a line under every reply. The
 * header already says how long the reply took, so the model's own latency
 * waits in the tooltip rather than reading as a second duration.
 */
function ModelNote({ message }: { message: AssistantMessage }) {
  const t = useT();
  const cost = formatUsd(message.costUsd);
  const latency = (message.latencyMs ?? 0) > 0 ? formatLatency(message.latencyMs ?? 0) : null;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="text-foreground-subtle hidden max-w-32 cursor-default truncate border-b border-dotted border-current/40 sm:inline" />
        }
      >
        {message.model}
      </TooltipTrigger>
      <TooltipContent className="flex flex-col gap-0.5">
        <span className="font-mono">{message.model}</span>
        <span>
          {t("{0} in, {1} out", message.inputTokens, message.outputTokens)} · {t("tokens")}
        </span>
        {(latency || cost) && (
          <span>{[latency, cost].filter((part): part is string => part !== null).join(" · ")}</span>
        )}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * A refusal is rendered as a boundary rather than as an error. The assistant did
 * not fail; it declined, and the message explains what it does cover.
 */
export function RefusalNotice({ message }: { message: string }) {
  return (
    <AssistantTurn>
      {/* A refusal is a sentence, not an incident. Framing it as a filled alert
          box made declining to write Python look like something had gone wrong,
          when the assistant simply answered. */}
      <div className="text-muted-foreground flex gap-2 text-sm">
        <ShieldAlertIcon className="text-warning mt-0.5 size-3.5 shrink-0" />
        <p className="min-w-0 flex-1">{message}</p>
      </div>
    </AssistantTurn>
  );
}

/** The question that was declined, kept in place so the thread reads in order. */
export function DeclinedTurn({ content, sentAt }: { content: string; sentAt: number }) {
  const t = useT();

  return (
    <article className="flex min-w-0 flex-col gap-1.5">
      <header className="flex h-5 min-w-0 items-center gap-2 text-xs">
        <span className="text-foreground-muted shrink-0 font-medium">{t("You")}</span>
        {sentAt > 0 && <TurnTime at={sentAt} />}
        <span className="text-foreground-subtle shrink-0">· {t("Not answered")}</span>
      </header>
      <div className="border-border-subtle w-fit max-w-full rounded-lg border border-dashed px-3 py-2">
        <p className="text-foreground-muted text-sm leading-relaxed break-words whitespace-pre-wrap">
          {content}
        </p>
      </div>
    </article>
  );
}

/**
 * A decision on a proposal, where the person's message would be.
 *
 * The turn after a decision starts from a note the application wrote: the
 * decision on its first line, then instructions to the agent. Drawn as the
 * person's message it read as though they had typed it, and the instructions
 * were never theirs to read. So it is an event in the thread — the decision,
 * in one line, between hairlines — and nothing after the first line is shown.
 */
export function DecisionNote({ content, at }: { content: string; at?: number }) {
  const t = useT();
  const line = decisionHeadline(content);
  const Icon = /^Rejected\b/u.test(line) ? XIcon : CheckCheckIcon;

  return (
    <div
      role="note"
      aria-label={t("Decision")}
      className="text-foreground-muted flex min-w-0 items-center gap-2.5 text-xs"
    >
      <span aria-hidden className="bg-border-subtle h-px w-3 shrink-0" />
      <span className="bg-sunken text-foreground-muted flex size-5 shrink-0 items-center justify-center rounded-full">
        <Icon className="size-3" aria-hidden />
      </span>
      <span className="text-foreground shrink-0 font-medium">{t("Decision")}</span>
      <span className="min-w-0 truncate" title={line || undefined}>
        {line || t("Following up on your decision")}
      </span>
      {at !== undefined && at > 0 && <TurnTime at={at} />}
      <span aria-hidden className="bg-border-subtle h-px min-w-3 flex-1" />
    </div>
  );
}

/** A date between turns, so a thread that spans days reads with them in it. */
export function DayDivider({ at, daysAgo }: { at: number; daysAgo: number }) {
  const t = useT();
  const label =
    daysAgo <= 0
      ? t("Today")
      : daysAgo === 1
        ? t("Yesterday")
        : formatUnixInUserTimezone(at, { month: "short", day: "numeric" });

  return (
    <div className="text-foreground-subtle flex items-center gap-3 py-1 text-xs" role="separator">
      <span className="bg-border-subtle h-px flex-1" />
      <span className="shrink-0 font-medium">{label}</span>
      <span className="bg-border-subtle h-px flex-1" />
    </div>
  );
}
