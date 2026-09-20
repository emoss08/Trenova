import { useT } from "@trenova/shared/i18n/use-t";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { toneVar } from "@/components/kpi/tone";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Message,
  MessageAvatar,
  MessageContent,
  MessageFooter,
} from "@trenova/shared/components/ui/message";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useAssistantAgent } from "@/components/agent-identity/agent-context";
import { AgentTile, type AgentTileSize } from "@/components/agent-identity/agent-tile";
import type { AssistantMessage, AssistantPageContext, AssistantProposal } from "@/types/assistant";
import { CheckIcon, CopyIcon, MapPinIcon, ShieldAlertIcon } from "lucide-react";
import { useCallback, useState, type ReactNode } from "react";
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
  const t = useT();

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
          <span className="text-muted-foreground inline-flex max-w-full items-center gap-1 text-xs">
            <MapPinIcon className="size-3 shrink-0" />
            <span className="truncate">
              {t("Asked from {0}", context.title || record || context.path)}
            </span>
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

/** A person's message, or the one they are about to send. */
export function UserBubble({
  content,
  sentAt,
  pageContext,
}: {
  content: string;
  sentAt?: number;
  pageContext?: AssistantPageContext | null;
}) {
  return (
    <Message align="end">
      <MessageAvatar className="self-start bg-transparent">
        <UserAvatar />
      </MessageAvatar>
      <MessageContent className="items-end">
        <div className="bg-secondary text-secondary-foreground border-border/60 max-w-[85%] rounded-2xl rounded-tr-md border px-3.5 py-2 text-sm whitespace-pre-wrap">
          {content}
        </div>
        {(sentAt !== undefined || pageContext) && (
          <MessageFooter className="flex-wrap gap-x-2 px-1">
            {sentAt !== undefined && <time>{formatUnixInUserTimezone(sentAt, TIME_FORMAT)}</time>}
            <PageContextChip context={pageContext} />
          </MessageFooter>
        )}
      </MessageContent>
    </Message>
  );
}

/** The frame around anything the assistant says, live or saved. */
export function AssistantFrame({ children, footer }: { children: ReactNode; footer?: ReactNode }) {
  return (
    <Message>
      <MessageAvatar className="self-start bg-transparent">
        <AgentAvatar />
      </MessageAvatar>
      <MessageContent className="gap-3">
        {children}
        {footer && <MessageFooter className="px-0.5">{footer}</MessageFooter>}
      </MessageContent>
    </Message>
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
    <div className="min-w-0 text-sm">
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

function CopyButton({ text }: { text: string }) {
  const t = useT();
  const [copied, setCopied] = useState(false);

  const copy = useCallback(async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }, [text]);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-xxs"
            className="text-muted-foreground hover:text-foreground opacity-0 transition-opacity group-hover/message:opacity-100 focus-visible:opacity-100"
            onClick={copy}
            aria-label={t("Copy reply")}
          />
        }
      >
        {copied ? <CheckIcon className="size-3" /> : <CopyIcon className="size-3" />}
      </TooltipTrigger>
      <TooltipContent>{copied ? t("Copied") : t("Copy")}</TooltipContent>
    </Tooltip>
  );
}

/** A saved assistant turn: what it said, what it looked up, what it asked to do. */
export function AssistantEntry({
  entry,
  proposals,
  threadId,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
  proposals: AssistantProposal[];
  threadId: string;
}) {
  const t = useT();
  const { message, tools } = entry;

  // A report run outlives the turn that started it, so the thread follows it
  // rather than leaving the reader to ask again for the outcome.
  const reportRuns = reportRunsFrom(tools);

  const steps: ToolStep[] = tools.map((exchange) => ({
    id: exchange.call.id,
    name: exchange.call.name,
    arguments: exchange.call.arguments,
    status: toolStatus(exchange.result),
    content: exchange.result?.content ?? "",
  }));

  return (
    <AssistantFrame
      footer={
        <span className="flex items-center gap-2">
          {/* Which model answered and what it cost belong to whoever is tuning
              the agent, not to the dispatcher reading the answer. The whole
              model identifier printed beside every reply was the longest thing
              in the footer and told a dispatcher nothing. */}
          {message.model === "" ? (
            <time>{formatUnixInUserTimezone(message.createdAt, TIME_FORMAT)}</time>
          ) : (
            <Tooltip>
              <TooltipTrigger
                render={
                  <time className="cursor-default border-b border-dotted border-current/40" />
                }
              >
                {formatUnixInUserTimezone(message.createdAt, TIME_FORMAT)}
              </TooltipTrigger>
              <TooltipContent className="flex flex-col gap-0.5">
                <span className="font-mono">{message.model}</span>
                <span>
                  {t("{0} in, {1} out", message.inputTokens, message.outputTokens)} · {t("tokens")}
                </span>
              </TooltipContent>
            </Tooltip>
          )}
          {message.content !== "" && <CopyButton text={message.content} />}
        </span>
      }
    >
      {steps.length > 0 && <ToolTimeline steps={steps} />}
      {message.content !== "" && <AssistantProse content={message.content} />}
      {reportRuns.map((run) => (
        <ReportRunCard key={run.runId} run={run} />
      ))}
      {proposals.map((proposal) => (
        <ProposalCard key={proposal.id} proposal={proposal} threadId={threadId} />
      ))}
    </AssistantFrame>
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
    <AssistantFrame>
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
    </AssistantFrame>
  );
}

/** The question that was declined, kept in place so the thread reads in order. */
export function DeclinedBubble({ content, sentAt }: { content: string; sentAt: number }) {
  const t = useT();

  return (
    <Message align="end">
      <MessageAvatar className="self-start bg-transparent">
        <UserAvatar />
      </MessageAvatar>
      <MessageContent className="items-end">
        <div className="border-border text-muted-foreground max-w-[85%] rounded-2xl rounded-tr-md border border-dashed px-3.5 py-2.5 text-sm whitespace-pre-wrap">
          {content}
        </div>
        <MessageFooter className="px-1">
          <time>{formatUnixInUserTimezone(sentAt, TIME_FORMAT)}</time> · {t("Not answered")}
        </MessageFooter>
      </MessageContent>
    </Message>
  );
}
