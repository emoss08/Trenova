import { useT } from "@trenova/shared/i18n/use-t";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Message,
  MessageAvatar,
  MessageContent,
  MessageFooter,
} from "@trenova/shared/components/ui/message";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { AssistantMessage, AssistantPageContext, AssistantProposal } from "@/types/assistant";
import { CheckIcon, CopyIcon, MapPinIcon, ShieldAlertIcon, SparklesIcon } from "lucide-react";
import { useCallback, useState, type ReactNode } from "react";
import { ProposalCard } from "./proposal-card";
import type { ThreadEntry } from "./thread-view";
import { ToolTimeline, type ToolStep } from "./tool-activity";

const TIME_FORMAT = { hour: "numeric", minute: "2-digit" } as const;

/** The agent's face in the thread: one mark, used everywhere it speaks. */
export function AgentAvatar({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "from-primary flex size-7 shrink-0 items-center justify-center rounded-md bg-gradient-to-br to-violet-500 text-white shadow-sm",
        className,
      )}
      aria-hidden
    >
      <SparklesIcon className="size-3.5" />
    </span>
  );
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
          <span className="text-muted-foreground inline-flex max-w-full items-center gap-1 text-[11px]">
            <MapPinIcon className="size-3 shrink-0" />
            <span className="truncate">
              {t("Asked from {0}", context.title || record || context.path)}
            </span>
          </span>
        }
      />
      <TooltipContent className="max-w-xs font-mono text-[11px]">
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
        <div className="from-primary text-primary-foreground max-w-[85%] rounded-2xl rounded-tr-md bg-gradient-to-br to-violet-600 px-3.5 py-2.5 text-sm whitespace-pre-wrap shadow-sm">
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
      <MessageContent>
        {children}
        {footer && <MessageFooter className="px-1">{footer}</MessageFooter>}
      </MessageContent>
    </Message>
  );
}

/** Prose from the assistant, in the same card its history uses. */
export function AssistantProse({
  content,
  streaming = false,
}: {
  content: string;
  streaming?: boolean;
}) {
  return (
    <div className="bg-card border-border/70 max-w-[92%] rounded-2xl rounded-tl-md border px-3.5 py-2.5 shadow-xs">
      <AiMarkdown content={content} />
      {streaming && (
        <span
          aria-hidden
          className="assistant-caret bg-primary ml-0.5 inline-block h-[1em] w-[2px] rounded-full align-text-bottom"
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
          <time>{formatUnixInUserTimezone(message.createdAt, TIME_FORMAT)}</time>
          {message.model !== "" && (
            <Tooltip>
              <TooltipTrigger render={<span className="cursor-default truncate" />}>
                · {message.model}
              </TooltipTrigger>
              <TooltipContent>
                {t("{0} in, {1} out", message.inputTokens, message.outputTokens)} · {t("tokens")}
              </TooltipContent>
            </Tooltip>
          )}
          {message.content !== "" && <CopyButton text={message.content} />}
        </span>
      }
    >
      {steps.length > 0 && <ToolTimeline steps={steps} />}
      {message.content !== "" && <AssistantProse content={message.content} />}
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
  const t = useT();

  return (
    <AssistantFrame>
      <Alert variant="warning" className="max-w-[92%]">
        <ShieldAlertIcon className="size-4" />
        <AlertTitle>{t("Outside what this assistant covers")}</AlertTitle>
        <AlertDescription>{message}</AlertDescription>
      </Alert>
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
