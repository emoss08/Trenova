import {
  DeskThread,
  type DeskThreadArtifacts,
  type DeskThreadSchedules,
} from "@/components/desk-chat/desk-thread";
import type { TurnHandoff } from "@/components/assistant/turn-stream";
import { turnTime } from "@/components/desk-chat/turn-time";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { whenIdle } from "@/lib/when-idle";
import { useAssistantEnabled } from "@/components/assistant/use-assistant-enabled";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskStore } from "@/stores/desk-store";
import { conversationPath } from "@/lib/conversation-path";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import type { AssistantEntityRef, AssistantMessage, AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback, useEffect, useMemo } from "react";
import { useNavigate } from "react-router";
import { LazyDeskWorkspace, preloadDeskWorkspace } from "./artifacts/desk-workspace-lazy";
import { DeskCaseCard } from "./case/desk-case-card";
import { useDeskCase } from "./case/use-desk-case";
import { useSnoozeClock } from "./case/use-snooze-clock";
import { withoutPendingLookups } from "./artifacts/pending-lookups";
import { DeskFactsBar } from "./composer/desk-facts";
import { DeskScheduleCard } from "./conversation/desk-schedule-card";
import {
  isScheduleRequest,
  useCreateSchedule,
  useScheduleActions,
  useScheduleClock,
} from "./conversation/desk-schedules";
import { useDesk } from "./desk-layout";
import { DeskHandoffCard } from "./handoff/desk-handoff-card";
import { useHandOffThread } from "./handoff/use-hand-off-thread";

export type DeskConversationProps = {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agentsUnavailable: boolean;
  onStartNew?: () => void;
};

/**
 * A conversation at the Desk: the shared thread with everything the Desk adds
 * to it — the workspace that slides in beside it and the conversation
 * narrowing to make room, schedules, chapters, pinned facts and hand-offs
 * between agents.
 */
export function DeskConversation({
  thread,
  agent,
  agentsUnavailable,
  onStartNew,
}: DeskConversationProps) {
  const desk = useDesk();
  const t = useT();
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const chapters = useDeskStore((state) => state.chaptersByThread[thread.id]);
  const toggleChapter = useDeskStore((state) => state.toggleChapter);
  // What the reply being written says about a hand-off, for the top bar's
  // menu, which watches the store rather than the reply.
  const setLiveHandoff = useDeskStore((state) => state.setLiveHandoff);
  const noteLiveHandoff = useCallback(
    (live: TurnHandoff | null) => setLiveHandoff(thread.id, live),
    [setLiveHandoff, thread.id],
  );
  const activeArtifactId = useDeskStore((state) => state.activeArtifactByThread[thread.id] ?? null);

  // A reply another agent holds the tools for is handed to it: the new
  // conversation opens with this one's summary and pinned facts, and the
  // reply, with the records it names, is asked there as its first question.
  const handOffThread = useHandOffThread(thread.id);
  const navigate = useNavigate();
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);
  const setStartExtras = useDeskHandoffStore((state) => state.setHandoff);
  const handOff = async (
    agentId: string,
    question: string,
    mentions: readonly AssistantEntityRef[],
  ) => {
    const result = await handOffThread.mutateAsync(agentId);
    const next = result.thread.id;
    if (mentions.length > 0) {
      setStartExtras({ threadId: next, files: [], mentions: [...mentions], providerId: "" });
    }
    if (question !== "") {
      setOpeningQuestion({ threadId: next, text: question });
    }
    void navigate(conversationPath(next));
  };

  // A page the agent opens takes the person out of the Desk; the conversation
  // goes with them, open in the corner panel, unless they have turned the
  // corner panel off.
  const assistantEnabled = useAssistantEnabled();
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const carryConversation = useCallback(() => {
    if (!assistantEnabled) {
      return;
    }
    setActiveThreadId(thread.id);
    openWidget();
  }, [assistantEnabled, openWidget, setActiveThreadId, thread.id]);

  // The running turn's lookups are not counted until the reply keeps them.
  const artifactsQuery = useQuery(assistant.artifacts(thread.id)._ctx.summary);
  const { setArtifactCount, pendingLookups } = desk;
  const artifactTotal = useMemo(() => {
    const visible = withoutPendingLookups(
      artifactsQuery.data?.results ?? [],
      artifactsQuery.data?.counts,
      pendingLookups,
    );
    return visible.counts?.all ?? visible.results.length;
  }, [artifactsQuery.data, pendingLookups]);
  useEffect(() => setArtifactCount(artifactTotal), [artifactTotal, setArtifactCount]);
  // A conversation with something to show will likely have it shown: the
  // workspace's code is read while the page is idle, before it is asked for.
  const hasArtifacts = artifactTotal > 0;
  useEffect(() => {
    if (!hasArtifacts) {
      return;
    }
    return whenIdle(() => void preloadDeskWorkspace().catch(() => undefined));
  }, [hasArtifacts]);

  const { openArtifact, setWorkspaceOpen, workspaceOpen, liveArtifacts, noteLiveArtifacts } = desk;
  const open = useCallback(
    (artifactId: string) => openArtifact(thread.id, artifactId),
    [openArtifact, thread.id],
  );
  const artifacts = useMemo<DeskThreadArtifacts>(
    () => ({
      open,
      activeId: activeArtifactId,
      onLive: noteLiveArtifacts,
      workspace: {
        open: workspaceOpen,
        setOpen: setWorkspaceOpen,
        content: (
          <LazyDeskWorkspace
            key={thread.id}
            threadId={thread.id}
            liveArtifacts={liveArtifacts}
            pendingLookups={pendingLookups}
            onClose={() => setWorkspaceOpen(false)}
          />
        ),
      },
    }),
    [
      activeArtifactId,
      liveArtifacts,
      noteLiveArtifacts,
      open,
      pendingLookups,
      setWorkspaceOpen,
      thread.id,
      workspaceOpen,
    ],
  );

  const schedules = useDeskSchedules(thread.id, timezone);
  const deskCase = useDeskCase(thread);
  const queryClient = useQueryClient();
  const refreshCase = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
    void queryClient.invalidateQueries({ queryKey: assistant.case(thread.id).queryKey });
  }, [queryClient, thread.id]);
  const now = useSnoozeClock(
    thread.case?.state === "Snoozed" ? thread.case.snoozedUntil : null,
    refreshCase,
  );
  const chapterApi = useMemo(
    () => ({
      of: (messageId: string) => (chapters ? chapters.indexOf(messageId) + 1 : 0),
      toggle: (messageId: string) => toggleChapter(thread.id, messageId),
    }),
    [chapters, thread.id, toggleChapter],
  );
  const { agentsById } = desk;
  const renderHandoff = useCallback(
    (message: AssistantMessage) =>
      message.handoff ? (
        <DeskHandoffCard
          handoff={message.handoff}
          incoming={message.kind === "HandoffBrief"}
          time={turnTime(message.createdAt, timezone, t)}
          agentsById={agentsById}
        />
      ) : null,
    [agentsById, t, timezone],
  );

  return (
    <DeskThread
      thread={thread}
      agent={agent}
      agentsUnavailable={agentsUnavailable}
      threads={desk.threads}
      onSwitchAgent={desk.start}
      onHandOff={handOff}
      onStartNew={onStartNew}
      onWorkingChange={desk.setWorking}
      onLiveHandoff={noteLiveHandoff}
      onNavigate={carryConversation}
      followNavigation={false}
      artifacts={artifacts}
      schedules={schedules}
      chapters={chapterApi}
      renderHandoff={renderHandoff}
      dockTop={(dock) => (
        <>
          {thread.case && (
            <DeskCaseCard
              thread={thread}
              deskCase={deskCase}
              dock={dock}
              now={now}
              timezone={timezone}
            />
          )}
          {thread.canContinue && <DeskFactsBar thread={thread} />}
        </>
      )}
    />
  );
}

/** The Desk's schedules for a conversation, in the shape the shared thread takes them. */
function useDeskSchedules(threadId: string, timezone: string): DeskThreadSchedules {
  const scheduling = useCreateSchedule(threadId);
  const schedulesQuery = useQuery(assistant.schedules(threadId));
  const actions = useScheduleActions();
  const clock = useScheduleClock();
  const byId = useMemo(
    () => new Map((schedulesQuery.data?.items ?? []).map((item) => [item.id, item])),
    [schedulesQuery.data],
  );
  const loaded = schedulesQuery.data !== undefined;
  const { create, pending } = scheduling;

  return useMemo(
    () => ({
      isRequest: isScheduleRequest,
      create,
      pending,
      card: (scheduleId: string) => (
        <DeskScheduleCard
          schedule={loaded ? (byId.get(scheduleId) ?? null) : undefined}
          now={clock}
          timezone={timezone}
          disabled={actions.busy}
          onToggle={actions.toggle}
          onDelete={actions.remove}
        />
      ),
    }),
    [actions.busy, actions.remove, actions.toggle, byId, clock, create, loaded, pending, timezone],
  );
}
