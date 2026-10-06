import {
  DeskThread,
  type DeskThreadArtifacts,
  type DeskThreadSchedules,
} from "@/components/desk-chat/desk-thread";
import { turnTime } from "@/components/desk-chat/turn-time";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantMessage, AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DeskWorkspace } from "./artifacts/desk-workspace";
import { withoutPendingLookups } from "./artifacts/pending-lookups";
import { DeskFactsBar } from "./composer/desk-facts";
import { DeskScheduleCard } from "./conversation/desk-schedule-card";
import {
  isScheduleRequest,
  useCreateSchedule,
  useScheduleActions,
} from "./conversation/desk-schedules";
import { useDesk } from "./desk-layout";
import { DeskHandoffCard } from "./handoff/desk-handoff-card";

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
  const activeArtifactId = useDeskStore((state) => state.activeArtifactByThread[thread.id] ?? null);

  // A page the agent opens takes the person out of the Desk; the conversation
  // goes with them, open in the corner panel.
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const carryConversation = useCallback(() => {
    setActiveThreadId(thread.id);
    openWidget();
  }, [openWidget, setActiveThreadId, thread.id]);

  // The running turn's lookups are not counted until the reply keeps them.
  const artifactsQuery = useQuery(queries.assistant.artifacts(thread.id));
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
          <DeskWorkspace
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
      onStartNew={onStartNew}
      onWorkingChange={desk.setWorking}
      onNavigate={carryConversation}
      followNavigation={false}
      artifacts={artifacts}
      schedules={schedules}
      chapters={chapterApi}
      renderHandoff={renderHandoff}
      dockTop={thread.canContinue ? <DeskFactsBar thread={thread} /> : null}
    />
  );
}

/** The Desk's schedules for a conversation, in the shape the shared thread takes them. */
function useDeskSchedules(threadId: string, timezone: string): DeskThreadSchedules {
  const scheduling = useCreateSchedule(threadId);
  const schedulesQuery = useQuery(queries.assistant.schedules(threadId));
  const actions = useScheduleActions(threadId);
  const [clock, setClock] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    // "Today" and "Tomorrow" on a schedule's next run move at midnight; a
    // minute is close enough for a label.
    const timer = window.setInterval(() => setClock(Math.floor(Date.now() / 1000)), 60_000);
    return () => window.clearInterval(timer);
  }, []);
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
