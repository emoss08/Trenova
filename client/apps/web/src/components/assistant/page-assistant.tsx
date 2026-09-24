import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import type { AssistantArtifact, PageAgent, PageThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { LockIcon } from "lucide-react";
import { useCallback, useMemo, useRef, type ReactNode } from "react";
import { MessageThread, type PageBinding, type PageRequest } from "./message-thread";
import { OutsideContentBadge } from "./outside-content-badge";
import { pageAssistantAvailability } from "./page-assistant-availability";

export type PageAssistantProps = {
  /**
   * What the conversation is about, as a cache key: the document, the saved
   * template, or a key of the page's own while a template is unsaved.
   */
  conversationKey: readonly unknown[];
  /** Opens, or returns, the person's conversation for this page. */
  open: (signal?: AbortSignal) => Promise<PageThread>;
  page: PageBinding;
  /** Sent once, into a conversation that is still empty. */
  openingQuestion?: string;
  pageRequest?: PageRequest | null;
  onPageRequestSent?: (key: string) => void;
  /**
   * Lists what the conversation produced under the turns that made it, and
   * opens one when chosen. Left out, nothing is listed.
   */
  onOpenArtifact?: (artifact: AssistantArtifact) => void;
  /** Drawn under the agent's name: the page's own progress, for example. */
  header?: ReactNode;
  className?: string;
};

/** The cache key of a page's opened conversation, for a page that must start it afresh. */
export function pageAssistantThreadKey(conversationKey: readonly unknown[]): readonly unknown[] {
  return ["page-assistant-thread", ...conversationKey];
}

function pageAgentChoice(agent: PageAgent): AgentChoice {
  return {
    id: agent.id,
    name: agent.name,
    description: agent.description,
    template: agent.template ?? null,
    icon: agent.icon,
    accent: agent.accent,
    toolNames: agent.toolNames,
    systemKey: agent.systemKey,
    starters: agent.starters,
  };
}

/**
 * A page's own assistant: the import assistant beside a document, the formula
 * assistant beside the editor. It is the shared conversation — the same turns,
 * approvals and history as the Desk — bound to the page, so the page's unsaved
 * work rides on every question and the changes the assistant hands back land
 * on the page.
 */
export function PageAssistant({
  conversationKey,
  open,
  page,
  openingQuestion,
  pageRequest,
  onPageRequestSent,
  onOpenArtifact,
  header,
  className,
}: PageAssistantProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canAsk, isLoading: permissionsLoading } = usePermission(
    Resource.Assistant,
    Operation.Create,
  );

  // Opening is idempotent for a subject: the server returns the person's live
  // conversation about it. It is asked once per page and kept.
  const opened = useQuery({
    queryKey: pageAssistantThreadKey(conversationKey),
    queryFn: ({ signal }) => open(signal),
    enabled: canAsk && !permissionsLoading,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
    retry: false,
  });

  const threadId = opened.data?.thread.id ?? "";
  // The conversation read fresh after each reply, so a document it read or a
  // change of access shows without reopening the page.
  const threadQuery = useQuery({
    ...queries.assistant.thread(threadId),
    enabled: threadId !== "",
    initialData: opened.data?.thread,
  });
  const thread = threadQuery.data ?? opened.data?.thread ?? null;

  const artifactsQuery = useQuery({
    ...queries.assistant.artifacts(threadId),
    enabled: threadId !== "" && onOpenArtifact !== undefined,
  });
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);
  const openArtifact = useCallback(
    (id: string) => {
      const artifact = artifacts.find((candidate) => candidate.id === id);
      if (artifact) {
        onOpenArtifact?.(artifact);
      }
    },
    [artifacts, onOpenArtifact],
  );

  const working = useRef(false);
  const onWorkingChange = useCallback(
    (active: boolean) => {
      const finished = working.current && !active;
      working.current = active;
      if (finished && threadId !== "") {
        void queryClient.invalidateQueries({
          queryKey: queries.assistant.thread(threadId).queryKey,
        });
      }
    },
    [queryClient, threadId],
  );

  const agent = useMemo(
    () => (opened.data ? pageAgentChoice(opened.data.agent) : null),
    [opened.data],
  );

  const availability = pageAssistantAvailability({
    permissionsLoading,
    canAsk,
    error: opened.error,
  });

  if (availability.state === "no-permission") {
    return (
      <PageAssistantNotice className={className}>
        <AlertTitle>{t("The assistant is not available to you")}</AlertTitle>
        <AlertDescription>
          {t(
            "Using the assistant needs permission to start assistant conversations. An administrator can add it to one of your roles.",
          )}
        </AlertDescription>
      </PageAssistantNotice>
    );
  }

  if (availability.state === "no-access" || availability.state === "unavailable") {
    return (
      <PageAssistantNotice className={className}>
        <AlertTitle>{t("The assistant cannot be used here")}</AlertTitle>
        <AlertDescription>{availability.message}</AlertDescription>
        {availability.state === "unavailable" && (
          <AlertAction>
            <Button variant="ghost" size="xs" onClick={() => void opened.refetch()}>
              {t("Try again")}
            </Button>
          </AlertAction>
        )}
      </PageAssistantNotice>
    );
  }

  if (thread === null || agent === null) {
    return (
      <div className={cn("flex flex-col gap-3 p-3", className)} aria-busy>
        <Skeleton className="h-5 w-2/5" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="ml-auto h-8 w-1/2" />
      </div>
    );
  }

  // The server closes a page's conversation when what it is about changes
  // under it — a document re-extracted, for one — and refuses further turns.
  if (thread.status === "Archived") {
    return (
      <PageAssistantNotice className={className}>
        <AlertTitle>{t("This conversation was closed")}</AlertTitle>
        <AlertDescription>
          {t("What it was about changed since, so it cannot continue. Start a new one to go on.")}
        </AlertDescription>
        <AlertAction>
          <Button
            variant="ghost"
            size="xs"
            onClick={() => void opened.refetch()}
            disabled={opened.isFetching}
          >
            {t("Start a new conversation")}
          </Button>
        </AlertAction>
      </PageAssistantNotice>
    );
  }

  const tainted = (thread.taintedAt ?? 0) > 0;

  return (
    <div className={cn("flex min-h-0 flex-col", className)}>
      <div className="flex shrink-0 flex-col gap-2 border-b px-3 py-2.5">
        <div className="flex min-w-0 items-center gap-2">
          <span className="min-w-0 flex-1 truncate text-sm font-medium">{agent.name}</span>
          {tainted && (
            <OutsideContentBadge
              t={t}
              title={t(
                "This conversation has read text from outside your organization, so every change it proposes waits for your approval.",
              )}
            />
          )}
        </div>
        {header}
      </div>
      <MessageThread
        key={thread.id}
        thread={thread}
        agent={agent}
        expanded={false}
        page={page}
        openingQuestion={openingQuestion}
        pageRequest={pageRequest}
        onPageRequestSent={onPageRequestSent}
        artifacts={onOpenArtifact ? artifacts : undefined}
        onOpenArtifact={onOpenArtifact ? openArtifact : undefined}
        onWorkingChange={onWorkingChange}
      />
    </div>
  );
}

function PageAssistantNotice({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("p-3", className)}>
      <Alert size="sm">
        <LockIcon />
        {children}
      </Alert>
    </div>
  );
}
