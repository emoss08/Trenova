import { useT } from "@trenova/shared/i18n/use-t";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Link, useParams } from "react-router";
import { DeskConversation } from "./_components/desk-conversation";
import { useDesk } from "./_components/desk-layout";

export function DeskConversationPage() {
  const t = useT();
  const { threadId } = useParams<{ threadId: string }>();
  const desk = useDesk();
  const thread = desk.threads.find((candidate) => candidate.id === threadId) ?? null;

  if (desk.isLoading) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-24 w-2/3" />
      </div>
    );
  }

  if (thread === null) {
    return (
      <EmptySheet
        className="my-auto"
        title={t("This conversation is not here")}
        description={t("It may have been deleted, or it belongs to someone else.")}
        action={
          <Button size="sm" variant="outline" nativeButton={false} render={<Link to="/desk" />}>
            {t("Back to the desk")}
          </Button>
        }
        sketch={
          <div className="flex flex-col gap-3">
            <GhostLine className="w-2/3" />
            <GhostLine className="w-1/2" />
          </div>
        }
      />
    );
  }

  const agent = desk.agentsById.get(thread.agentDefinitionId) ?? null;

  return (
    <DeskConversation
      key={thread.id}
      thread={thread}
      agent={agent}
      agentsUnavailable={desk.agentsUnavailable}
      onStartNew={agent && !desk.isStarting ? () => desk.start(agent.id) : undefined}
      onDelete={desk.remove}
      onTogglePin={desk.togglePin}
    />
  );
}
