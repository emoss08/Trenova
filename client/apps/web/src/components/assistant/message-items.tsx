import { formatLatency, formatUsd } from "@/lib/ai-usage-format";
import { cn } from "@trenova/shared/lib/utils";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { toneVar } from "@/components/kpi/tone";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
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
import {
  CheckIcon,
  ChevronRightIcon,
  CopyIcon,
  AtSignIcon,
  FileIcon,
  MapPinIcon,
  RotateCcwIcon,
  ShieldAlertIcon,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { askRequestsFrom } from "./ask-requests";
import { ChoicePrompt } from "./choice-prompt";
import { PlanCard } from "./plan-card";
import type { PlanGroup } from "./plan-state";
import { ProposalCard } from "./proposal-card";
import { ReportRunCard } from "./report-run-card";
import { reportRunsFrom } from "./report-runs";
import type { ThreadEntry } from "./thread-view";
import { ToolTimeline, type ToolStep } from "./tool-activity";

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

function UserAvatar() {
  const user = useAuthStore((s) => s.user);

  return (
    <ResolvedUserAvatar
      userId={user?.id}
      name={user?.name}
      profilePicUrl={user?.profilePicUrl}
      thumbnailUrl={user?.thumbnailUrl}
      className="size-7 rounded-md"
    />
  );
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

/**
 * One turn of the ledger: who spoke in a narrow gutter, then what they said
 * running the full width.
 *
 * There are no bubbles. A bubble says "chat partner" and puts the two sides
 * on opposite walls, so a reader's eye crosses the panel on every exchange.
 * Here both sides share one left edge, the way a transcript reads, and the
 * gutter mark is what tells them apart. The header line carries the name
 * and the time and, on hover, the actions.
 */
function Turn({
  mark,
  name,
  at,
  meta,
  actions,
  muted = false,
  children,
}: {
  mark: ReactNode;
  name: ReactNode;
  at?: number;
  meta?: ReactNode;
  actions?: ReactNode;
  muted?: boolean;
  children: ReactNode;
}) {
  return (
    <article className="group/turn grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-2.5">
      <div className="flex justify-center pt-px">{mark}</div>
      <div className="flex min-w-0 flex-col gap-1.5">
        <header className="text-muted-foreground flex h-5 min-w-0 items-center gap-2 text-xs">
          <span className={cn("shrink-0 font-medium", !muted && "text-foreground")}>{name}</span>
          {at !== undefined && at > 0 && (
            <time className="shrink-0 tabular-nums">
              {formatUnixInUserTimezone(at, TIME_FORMAT)}
            </time>
          )}
          {meta}
          {actions && (
            <span className="ml-auto flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover/turn:opacity-100 has-[:focus-visible]:opacity-100">
              {actions}
            </span>
          )}
        </header>
        {children}
      </div>
    </article>
  );
}

/** A person's message, or the one they are about to send. */
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
  const user = useAuthStore((s) => s.user);

  return (
    <Turn
      mark={<UserAvatar />}
      name={user?.name ? user.name.split(" ")[0] : t("You")}
      at={sentAt}
      meta={<PageContextChip context={pageContext} />}
      actions={
        <>
          <IconAction label={t("Copy")} done={t("Copied")} onClick={() => copyText(content)}>
            <CopyIcon className="size-3" />
          </IconAction>
          {onResend && (
            <IconAction label={t("Ask again")} onClick={onResend}>
              <RotateCcwIcon className="size-3" />
            </IconAction>
          )}
        </>
      }
    >
      <p className="text-sm leading-relaxed font-medium whitespace-pre-wrap">{content}</p>
      <TurnContextChips attachments={attachments} mentions={mentions} />
    </Turn>
  );
}

/** The frame around anything the assistant says, live or saved. */
export function AssistantTurn({
  at,
  meta,
  actions,
  children,
}: {
  at?: number;
  meta?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  const t = useT();
  const agent = useAssistantAgent();

  return (
    <Turn
      mark={<AgentAvatar />}
      name={agent?.name ?? t("Assistant")}
      at={at}
      meta={meta}
      actions={actions}
    >
      <div className="flex min-w-0 flex-col gap-3">{children}</div>
    </Turn>
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
 * heavy model would otherwise spend looking hung. Collapsed once the answer
 * begins: the reasoning is there for whoever wants to check the answer
 * against it, not in the way of reading the answer.
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
      <CollapsibleTrigger
        className={cn(
          "text-muted-foreground ui-focus-ring flex items-center gap-1.5 rounded-control text-xs",
          "hover:text-foreground",
        )}
      >
        <ChevronRightIcon
          className={cn("size-3.5 transition-transform", open && "rotate-90")}
          aria-hidden
        />
        {streaming ? (
          <TextShimmer as="span">{t("Thinking…")}</TextShimmer>
        ) : (
          <span>{t("Thought it through")}</span>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="text-muted-foreground mt-2 pl-5 text-xs leading-relaxed whitespace-pre-wrap">
          {text}
          {streaming && (
            <span
              aria-hidden
              className="assistant-caret bg-muted-foreground ml-0.5 inline-block h-[1em] w-[2px] rounded-full align-text-bottom"
            />
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/** A saved assistant turn: what it said, what it looked up, what it asked to do. */
export function AssistantEntry({
  entry,
  proposals,
  plans = [],
  artifacts = [],
  threadId,
  latestUserSequence,
  onAnswer,
  onOpenArtifact,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
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
}) {
  const t = useT();
  const { message, tools } = entry;

  // A report run outlives the turn that started it, so the thread follows it
  // rather than leaving the reader to ask again for the outcome.
  const reportRuns = reportRunsFrom(tools);
  const asks = askRequestsFrom(tools);

  const steps: ToolStep[] = tools.map((exchange) => ({
    id: exchange.call.id,
    name: exchange.call.name,
    arguments: exchange.call.arguments,
    status: toolStatus(exchange.result),
    content: exchange.result?.content ?? "",
  }));

  return (
    <AssistantTurn
      at={message.createdAt}
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
    >
      {message.reasoning?.text ? <ReasoningDisclosure text={message.reasoning.text} /> : null}
      {steps.length > 0 && <ToolTimeline steps={steps} />}
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
 * it is a mark in the header rather than a line under every reply.
 */
function ModelNote({ message }: { message: AssistantMessage }) {
  const t = useT();
  const cost = formatUsd(message.costUsd);
  const latency = (message.latencyMs ?? 0) > 0 ? formatLatency(message.latencyMs ?? 0) : null;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="hidden max-w-32 cursor-default truncate border-b border-dotted border-current/40 sm:inline" />
        }
      >
        {latency ?? message.model}
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

function toolStatus(result: AssistantMessage | null): ToolStep["status"] {
  if (result === null) return "done";
  if (result.toolFailed) return "failed";
  return result.content.startsWith("Recorded a proposal") ? "proposed" : "done";
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
        <ShieldAlertIcon
          className="mt-0.5 size-3.5 shrink-0"
          style={{ color: toneVar("warning") }}
        />
        <p className="min-w-0 flex-1">{message}</p>
      </div>
    </AssistantTurn>
  );
}

/** The question that was declined, kept in place so the thread reads in order. */
export function DeclinedTurn({ content, sentAt }: { content: string; sentAt: number }) {
  const t = useT();

  return (
    <Turn
      mark={<UserAvatar />}
      name={t("You")}
      at={sentAt}
      muted
      meta={<span className="shrink-0">· {t("Not answered")}</span>}
    >
      <p className="text-muted-foreground text-sm leading-relaxed whitespace-pre-wrap">{content}</p>
    </Turn>
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
    <div className="text-muted-foreground flex items-center gap-3 py-1 text-xs" role="separator">
      <span className="bg-border h-px flex-1" />
      <span className="shrink-0">{label}</span>
      <span className="bg-border h-px flex-1" />
    </div>
  );
}
