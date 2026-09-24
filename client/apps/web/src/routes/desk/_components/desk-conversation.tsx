import { MessageThread } from "@/components/assistant/message-thread";
import { queries } from "@/lib/queries";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useOpeningQuestion } from "@/components/assistant/use-opening-question";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { useDesk } from "./desk-layout";

export type DeskConversationProps = {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agentsUnavailable: boolean;
  onStartNew?: () => void;
};

/**
 * One conversation at the Desk.
 *
 * It is only the conversation. The title, the pin, the transcript and the
 * delete used to live in a second header inside this component, underneath
 * the one the room already had, which gave a person two strips of chrome to
 * read before the first sentence. They belong to the room, so the room has
 * them, and this column is the thread and nothing else.
 *
 * What the turn produces goes to the workspace beside it rather than being
 * drawn twice: the transcript refers to an artifact, the workspace holds it.
 */
export function DeskConversation({
  thread,
  agent,
  agentsUnavailable,
  onStartNew,
}: DeskConversationProps) {
  const desk = useDesk();
  const opening = useOpeningQuestion(thread.id);
  const artifactsQuery = useQuery(queries.assistant.artifacts(thread.id));
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);

  // Opening a page leaves the Desk, so the conversation moves to the
  // floating assistant, which is on every page, before the page changes; it
  // picks up the reply still being written there.
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const carryConversation = useCallback(() => {
    setActiveThreadId(thread.id);
    openWidget();
  }, [openWidget, setActiveThreadId, thread.id]);

  const openArtifact = useCallback(
    (artifactId: string) => desk.openArtifact(thread.id, artifactId),
    [desk, thread.id],
  );

  return (
    <MessageThread
      key={thread.id}
      thread={thread}
      agent={agent}
      agentsUnavailable={agentsUnavailable}
      expanded
      onStartNew={onStartNew}
      artifacts={artifacts}
      onOpenArtifact={openArtifact}
      onLiveArtifact={desk.noteLiveArtifact}
      onWorkingChange={desk.setWorking}
      onNavigate={carryConversation}
      openingQuestion={opening.openingQuestion}
      onOpeningQuestionSent={opening.onOpeningQuestionSent}
      agentAccent
    />
  );
}
