import type { DeskThreadChapters } from "@/components/desk-chat/desk-thread";
import { useLiveTurns } from "@/components/assistant/use-active-turns";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { AssistantChapter, AssistantMessage, AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { useHandOffThread } from "../handoff/use-hand-off-thread";
import { chapterEntries, MIN_SHOWN_CHAPTERS, shownChapters } from "./chapter-work";
import { DeskChapterHeading, type ChapterActions } from "./desk-chapter-heading";
import { DeskChapterRail } from "./desk-chapter-rail";

const NO_CHAPTERS: readonly AssistantChapter[] = [];

/**
 * A Desk conversation's chapters, in the shape the shared thread takes them:
 * the heading of each, the rail, and starting one where a reply was asked.
 * Renaming shows the new title at once and sends the version it was read at,
 * so a rename someone made meanwhile is not overwritten. Handing a chapter on
 * opens the conversation it went to.
 */
export function useDeskChapters({
  thread,
  agent,
  agents,
  timezone,
  onHandedOff,
  linkedChapterId = null,
  onLinkShown,
}: {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agents: readonly AgentChoice[];
  timezone: string;
  onHandedOff: (threadId: string) => void;
  /** A chapter a link asked to open the conversation at. */
  linkedChapterId?: string | null;
  onLinkShown?: () => void;
}): DeskThreadChapters {
  const threadId = thread.id;
  const queryClient = useQueryClient();
  const chaptersQuery = useQuery(assistant.chapters(threadId));
  const artifactsQuery = useQuery(assistant.artifacts(threadId)._ctx.summary);
  const proposalsQuery = useQuery(assistant.proposals(threadId));
  const waitsQuery = useQuery(assistant.waits(threadId));
  const liveTurns = useLiveTurns();
  const liveTurnId = liveTurns.find((turn) => turn.threadId === threadId)?.turnId ?? null;
  const chaptersKey = assistant.chapters(threadId).queryKey;

  const entries = useMemo(
    () =>
      chapterEntries(
        shownChapters(chaptersQuery.data ?? NO_CHAPTERS, liveTurnId),
        {
          artifacts: artifactsQuery.data?.results ?? [],
          proposals: proposalsQuery.data?.results ?? [],
          waits: waitsQuery.data?.items ?? [],
        },
        liveTurnId,
      ),
    [artifactsQuery.data, chaptersQuery.data, liveTurnId, proposalsQuery.data, waitsQuery.data],
  );
  const byTurn = useMemo(
    () => new Map(entries.map((entry) => [entry.chapter.turnId, entry])),
    [entries],
  );
  // A conversation about one thing has one chapter, and naming it would only
  // repeat the conversation: chapters show once there is a second.
  const shown = entries.length >= MIN_SHOWN_CHAPTERS;

  const refresh = () => queryClient.invalidateQueries({ queryKey: chaptersKey });

  const rename = useApiMutation({
    mutationFn: ({ chapter, title }: { chapter: AssistantChapter; title: string }) =>
      apiService.assistantService.renameChapter(threadId, chapter.id, title, chapter.version),
    onSuccess: (saved) => {
      queryClient.setQueryData<AssistantChapter[]>(chaptersKey, (chapters) =>
        chapters?.map((chapter) => (chapter.id === saved.id ? saved : chapter)),
      );
    },
    onError: () => void refresh(),
    resourceName: "Chapter",
  });
  const open = useApiMutation({
    mutationFn: (messageId: string) => apiService.assistantService.openChapter(threadId, messageId),
    onSuccess: () => void refresh(),
    resourceName: "Chapter",
  });
  const merge = useApiMutation({
    mutationFn: (chapterId: string) =>
      apiService.assistantService.removeChapter(threadId, chapterId),
    onMutate: (chapterId) => {
      queryClient.setQueryData<AssistantChapter[]>(chaptersKey, (chapters) =>
        chapters?.filter((chapter) => chapter.id !== chapterId),
      );
    },
    onSettled: () => void refresh(),
    resourceName: "Chapter",
  });
  const handOff = useHandOffThread(threadId, thread.pinnedFacts ?? []);

  const otherAgents = useMemo(
    () => agents.filter((choice) => choice.id !== agent?.id),
    [agent?.id, agents],
  );
  const busy = handOff.isPending || merge.isPending;

  const actions: ChapterActions = {
    rename: (entry, title) => rename.mutate({ chapter: entry.chapter, title }),
    continueAlone:
      agent && thread.canContinue
        ? (entry) =>
            handOff.mutate(
              { agentId: agent.id, chapterId: entry.chapter.id },
              { onSuccess: (result) => onHandedOff(result.thread.id) },
            )
        : undefined,
    handOff: thread.canContinue
      ? (entry, agentId) =>
          handOff.mutate(
            { agentId, chapterId: entry.chapter.id },
            { onSuccess: (result) => onHandedOff(result.thread.id) },
          )
      : undefined,
    merge: (entry) => merge.mutate(entry.chapter.id),
    agents: otherAgents,
    busy,
  };

  const linkedTurnId =
    linkedChapterId === null
      ? null
      : (entries.find((entry) => entry.chapter.id === linkedChapterId)?.chapter.turnId ?? null);

  const opensChapter = (message: AssistantMessage) =>
    message.turnId ? byTurn.has(message.turnId) : true;

  return {
    heading: (turnId: string) => {
      const entry = shown ? byTurn.get(turnId) : undefined;
      return entry ? (
        <DeskChapterHeading entry={entry} actions={actions} timezone={timezone} />
      ) : null;
    },
    liveTurnId: shown && liveTurnId !== null && byTurn.has(liveTurnId) ? liveTurnId : null,
    rail: (nav) => <DeskChapterRail entries={entries} nav={nav} />,
    startAt: (message: AssistantMessage) =>
      opensChapter(message) || open.isPending ? undefined : () => open.mutate(message.id),
    linkedTurnId,
    onLinkShown,
  };
}

