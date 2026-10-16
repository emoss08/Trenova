import { DeskConversationSkeleton } from "@/components/desk-chat/desk-skeletons";
import { useEffect } from "react";
import { useParams, useSearchParams } from "react-router";
import { DeskConversation } from "./_components/desk-conversation";
import { DeskMissingConversation } from "./_components/desk-missing-conversation";
import { useDesk } from "./_components/desk-layout";

export function DeskConversationPage() {
  const { threadId } = useParams<{ threadId: string }>();
  const desk = useDesk();
  const thread = desk.activeThread?.id === threadId ? desk.activeThread : null;

  // A link to one artifact opens the conversation with that artifact beside it.
  const [search, setSearch] = useSearchParams();
  const linkedArtifact = search.get("a");
  const { openArtifact } = desk;
  useEffect(() => {
    if (!thread || !linkedArtifact) {
      return;
    }
    openArtifact(thread.id, linkedArtifact);
    setSearch((current) => {
      const next = new URLSearchParams(current);
      next.delete("a");
      return next;
    }, { replace: true });
  }, [linkedArtifact, openArtifact, setSearch, thread]);

  if (desk.isLoading) {
    return <DeskConversationSkeleton />;
  }

  // Deleted from here, it is on its way out rather than missing: the page
  // stays blank while the Desk moves off it.
  if (desk.deletingActive) {
    return null;
  }

  if (thread === null) {
    return <DeskMissingConversation onSearch={desk.openSearch} />;
  }

  const agent = desk.agentsById.get(thread.agentDefinitionId) ?? null;

  return (
    <DeskConversation
      key={thread.id}
      thread={thread}
      agent={agent}
      agentsUnavailable={desk.agentsUnavailable}
      onStartNew={agent && !desk.isStarting ? () => desk.start(agent.id) : undefined}
    />
  );
}
